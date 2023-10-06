package pipelinedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"storj.io/crypto-batch-payment/pkg"

	"github.com/ethereum/go-ethereum/common"
	//	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/payoutdb"
)

const (
	dbVersion = 2
)

type DB struct {
	db       *payoutdb.DB
	metadata *payoutdb.Metadata
}

func NewDB(ctx context.Context, path string) (*DB, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return nil, errs.New("database already exists at %q", path)
	case !os.IsNotExist(err):
		return nil, errs.Wrap(err)
	}
	if err := initDB(ctx, path); err != nil {
		return nil, err
	}

	return OpenDB(ctx, path, false)
}

func OpenInMemoryDB(ctx context.Context) (*DB, error) {
	db, err := payoutdb.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, errs.Wrap(err)
	}
	db.Hooks.Now = func() time.Time {
		// Row timestamps are for audit only. Nanosecond precision is overkill
		// and makes the output harder to read.
		return time.Now().Truncate(time.Millisecond)
	}
	if _, err := db.Exec(db.Schema()); err != nil {
		return nil, errs.Wrap(err)
	}
	if err := db.CreateNoReturn_Metadata(ctx,
		payoutdb.Metadata_Version(dbVersion),
		payoutdb.Metadata_Attempts(0),
		payoutdb.Metadata_Create_Fields{},
	); err != nil {
		return nil, errs.Wrap(err)
	}
	metadata, err := db.First_Metadata(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	if metadata == nil {
		return nil, errs.New("database metadata is missing")
	}

	return &DB{
		db:       db,
		metadata: metadata,
	}, nil
}

func OpenDB(ctx context.Context, path string, readOnly bool) (_ *DB, err error) {

	db, err := openDB(path, readOnly)
	if err != nil {
		return nil, err
	}
	defer func() {
		if db != nil && err != nil {
			err = errs.Combine(err, db.Close())
		}
	}()

	row, err := db.First_Metadata_Version(ctx)
	if err != nil {
		return nil, err
	}

	switch {
	case row.Version < dbVersion:
		// Database is old. Migrate forward and retry.
		if err := migrateDB(ctx, db, row.Version); err != nil {
			return nil, err
		}
		if err := db.Close(); err != nil {
			return nil, err
		}
		db, err = openDB(path, readOnly)
		if err != nil {
			return nil, err
		}
	case row.Version > dbVersion:
		// Database version is from a future tool. It is not safe to continue.
		return nil, errs.New("database version is in the future (%d); upgrade your tool (%d)", row.Version, dbVersion)
	}

	// read out the metadata and check the version for compatability
	metadata, err := db.First_Metadata(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	if metadata == nil {
		return nil, errs.New("database metadata is missing")
	}

	return &DB{
		db:       db,
		metadata: metadata,
	}, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) RecordStart(ctx context.Context, spender common.Address, owner *common.Address) error {
	var update payoutdb.Metadata_Update_Fields
	switch {
	case db.metadata.Attempts == 0:
		update.Spender = payoutdb.Metadata_Spender(spender.String())
	case *db.metadata.Spender != spender.String():
		return errs.New("spender cannot change once payouts have been initiated; expected=%v got=%v", *db.metadata.Spender, spender.String())
	}

	switch {
	case db.metadata.Attempts == 0:
		if owner != nil {
			update.Owner = payoutdb.Metadata_Owner(owner.String())
		}
	case owner == nil && db.metadata.Owner == nil:
	case owner != nil && db.metadata.Owner == nil:
		return errs.New("owner cannot change once payouts have started; expected=%v got=%v", nil, owner.String())
	case owner == nil && db.metadata.Owner != nil:
		return errs.New("owner cannot change once payouts have started; expected=%v got=%v", *db.metadata.Owner, nil)
	case owner != nil && db.metadata.Owner != nil:
		if *db.metadata.Owner != owner.String() {
			return errs.New("owner cannot change once payouts have started; expected=%v got=%v", *db.metadata.Owner, owner.String())
		}
	}

	update.Attempts = payoutdb.Metadata_Attempts(db.metadata.Attempts + 1)
	if err := db.db.UpdateNoReturn_Metadata_By_Pk(ctx, payoutdb.Metadata_Pk(db.metadata.Pk), update); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

func (db *DB) CreatePayoutGroup(ctx context.Context, payoutGroupID int64, payouts []*Payout) error {
	return db.db.WithTx(ctx, func(tx *payoutdb.Tx) error {
		if err := tx.CreateNoReturn_PayoutGroup(ctx,
			payoutdb.PayoutGroup_Id(payoutGroupID),
			payoutdb.PayoutGroup_Create_Fields{},
		); err != nil {
			return err
		}
		for _, payout := range payouts {
			payout.PayoutGroupID = payoutGroupID
			if err := tx.CreateNoReturn_Payout(ctx,
				payoutdb.Payout_CsvLine(payout.CSVLine),
				payoutdb.Payout_Payee(payout.Payee.String()),
				payoutdb.Payout_Usd(payout.USD.String()),
				payoutdb.Payout_PayoutGroupId(payoutGroupID),
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (db *DB) FetchPayouts(ctx context.Context) ([]*Payout, error) {
	rows, err := db.db.All_Payout(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return PayoutsFromRows(rows)
}

func (db *DB) FetchUnfinishedTransactionsSortedIntoNonceGroups(ctx context.Context) ([]*NonceGroup, error) {
	rows, err := db.db.All_Transaction_By_State_OrderBy_Asc_Nonce(ctx,
		payoutdb.Transaction_State(string(TxPending)))
	if err != nil {
		return nil, errs.Wrap(err)
	}

	txs, err := TransactionsFromRows(rows)
	if err != nil {
		return nil, err
	}

	groups := make([]*NonceGroup, 0, len(txs))
	for _, tx := range txs {
		// add the tx to the last group if the nonces match.
		if len(groups) > 0 {
			lastGroup := groups[len(groups)-1]
			if lastGroup.Nonce == tx.Nonce {
				if lastGroup.PayoutGroupID != tx.PayoutGroupID {
					return nil, errs.New("expected payout group %d on nonce group %d transaction %s; got %d",
						lastGroup.PayoutGroupID,
						lastGroup.Nonce,
						tx.Hash,
						tx.PayoutGroupID)
				}
				lastGroup.Txs = append(lastGroup.Txs, *tx)
				continue
			}
		}

		groups = append(groups, &NonceGroup{
			Nonce:         tx.Nonce,
			PayoutGroupID: tx.PayoutGroupID,
			Txs:           []Transaction{*tx},
		})
	}

	return groups, nil
}

func (db *DB) CountUnfinishedUnattachedPayoutGroup(ctx context.Context) (int64, error) {
	count, err := db.db.CountUnfinishedUnattachedPayoutGroup(ctx)
	if err != nil {
		return 0, errs.Wrap(err)
	}
	return count, nil
}

func (db *DB) FetchFirstUnfinishedUnattachedPayoutGroup(ctx context.Context) (*PayoutGroup, error) {
	row, err := db.db.FirstUnfinishedUnattachedPayoutGroup(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	if row == nil {
		return nil, nil
	}
	return PayoutGroupFromRow(row)
}

// FetchPayoutGroupPayouts returns all of the payouts for a given payout group.
func (db *DB) FetchPayoutGroupPayouts(ctx context.Context, payoutGroupID int64) ([]*Payout, error) {
	rows, err := db.db.All_Payout_By_PayoutGroupId(ctx, payoutdb.Payout_PayoutGroupId(payoutGroupID))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return PayoutsFromRows(rows)
}

// FetchPayoutGroupPayoutCount returns the count of payouts in a payout group
func (db *DB) FetchPayoutGroupPayoutCount(ctx context.Context, payoutGroupID int64) (int64, error) {
	count, err := db.db.Count_Payout_By_PayoutGroupId(ctx, payoutdb.Payout_PayoutGroupId(payoutGroupID))
	if err != nil {
		return 0, errs.Wrap(err)
	}
	return count, nil
}

// FetchPayoutGroupTransactions returns all of the transactions for a given payout group.
func (db *DB) FetchPayoutGroupTransactions(ctx context.Context, payoutGroupID int64) ([]*Transaction, error) {
	rows, err := db.db.All_Transaction_By_PayoutGroupId(ctx, payoutdb.Transaction_PayoutGroupId(payoutGroupID))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return TransactionsFromRows(rows)
}

// FinalizeNonceGroup finalizes the transaction state for transactions in a
// nonce group. It also sets the final tx hash on the payout group for the
// confirmed transaction.
func (db *DB) FinalizeNonceGroup(ctx context.Context, nonceGroup *NonceGroup, statuses []*TxStatus) error {
	return db.db.WithTx(ctx, func(tx *payoutdb.Tx) error {
		for _, status := range statuses {
			if err := setTransactionStatus(ctx, tx, status); err != nil {
				return err
			}
			if status.State == TxConfirmed {
				err := tx.UpdateNoReturn_PayoutGroup_By_Id(ctx,
					payoutdb.PayoutGroup_Id(nonceGroup.PayoutGroupID),
					payoutdb.PayoutGroup_Update_Fields{
						FinalTxHash: payoutdb.PayoutGroup_FinalTxHash(status.Hash),
					})
				if err != nil {
					return errs.Wrap(err)
				}
			}
		}
		return nil
	})
}

func (db *DB) CreateTransaction(ctx context.Context, tx Transaction) (*Transaction, error) {

	row, err := db.db.Create_Transaction(ctx,
		payoutdb.Transaction_Hash(tx.Hash),
		payoutdb.Transaction_Owner(tx.Owner.String()),
		payoutdb.Transaction_Spender(tx.Spender.String()),
		payoutdb.Transaction_Nonce(tx.Nonce),
		payoutdb.Transaction_EstimatedGasPrice("0"),
		payoutdb.Transaction_StorjPrice(tx.StorjPrice.String()),
		payoutdb.Transaction_StorjTokens(tx.StorjTokens.String()),
		payoutdb.Transaction_PayoutGroupId(tx.PayoutGroupID),
		payoutdb.Transaction_Raw(string(tx.Raw)),
		payoutdb.Transaction_State(string(TxPending)),
		payoutdb.Transaction_Create_Fields{},
	)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return TransactionFromRow(row)
}

func (db *DB) UpdateTransactionState(ctx context.Context, hash string, state TxState) error {
	update := payoutdb.Transaction_Update_Fields{
		State: payoutdb.Transaction_State(string(state)),
	}
	return db.db.UpdateNoReturn_Transaction_By_Hash(ctx,
		payoutdb.Transaction_Hash(hash),
		update,
	)
}

func (db *DB) FetchPayoutGroup(ctx context.Context, id int64) (*PayoutGroup, error) {
	row, err := db.db.Find_PayoutGroup_By_Id(ctx, payoutdb.PayoutGroup_Id(id))
	if err != nil {
		return nil, err
	}
	return PayoutGroupFromRow(row)
}

func (db *DB) FetchTransaction(ctx context.Context, hash common.Hash) (*Transaction, error) {
	row, err := db.db.Find_Transaction_By_Hash(ctx, payoutdb.Transaction_Hash(hash.String()))
	if err != nil {
		return nil, err
	}
	return TransactionFromRow(row)
}

func (db *DB) FetchTransactions(ctx context.Context) ([]*Transaction, error) {
	rows, err := db.db.All_Transaction(ctx)
	if err != nil {
		return nil, err
	}
	return TransactionsFromRows(rows)
}

func (db *DB) FetchPayoutProgress(ctx context.Context) (int64, int64, error) {
	total, err := db.db.Count_PayoutGroup(ctx)
	if err != nil {
		return 0, 0, errs.Wrap(err)
	}

	pending, err := db.db.Count_PayoutGroup_By_FinalTxHash_Is_Null(ctx)
	if err != nil {
		return 0, 0, errs.Wrap(err)
	}
	return pending, total, nil
}

type DBStats struct {
	Spender               *common.Address
	Owner                 *common.Address
	Payees                int64
	TotalPayouts          int64
	TotalUSD              decimal.Decimal
	PendingPayouts        int64
	PendingUSD            decimal.Decimal
	TotalPayoutGroups     int64
	PendingPayoutGroups   int64
	TotalTransactions     int64
	PendingTransactions   int64
	FailedTransactions    int64
	ConfirmedTransactions int64
	DroppedTransactions   int64
}

func (db *DB) Stats(ctx context.Context) (_ *DBStats, err error) {
	stats := new(DBStats)

	payoutRows, err := db.db.All_Payout(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	payouts, err := PayoutsFromRows(payoutRows)
	if err != nil {
		return nil, err
	}

	payees := make(map[common.Address]struct{})
	stats.TotalPayouts = int64(len(payouts))
	for _, payout := range payouts {
		payees[payout.Payee] = struct{}{}
		stats.TotalUSD = stats.TotalUSD.Add(payout.USD)
	}
	stats.Payees = int64(len(payees))

	payoutRows, err = db.db.All_Payout_By_PayoutGroup_FinalTxHash_Is_Null(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	payouts, err = PayoutsFromRows(payoutRows)
	if err != nil {
		return nil, err
	}

	stats.PendingPayouts = int64(len(payouts))
	for _, payout := range payouts {
		stats.PendingUSD = stats.PendingUSD.Add(payout.USD)
	}

	stats.TotalPayoutGroups, err = db.db.Count_PayoutGroup(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	stats.PendingPayoutGroups, err = db.db.Count_PayoutGroup_By_FinalTxHash_Is_Null(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	stats.TotalTransactions, err = db.db.Count_Transaction(ctx)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	stats.PendingTransactions, err = db.db.Count_Transaction_By_State(ctx, payoutdb.Transaction_State(string(TxPending)))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	stats.FailedTransactions, err = db.db.Count_Transaction_By_State(ctx, payoutdb.Transaction_State(string(TxFailed)))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	stats.ConfirmedTransactions, err = db.db.Count_Transaction_By_State(ctx, payoutdb.Transaction_State(string(TxConfirmed)))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	stats.DroppedTransactions, err = db.db.Count_Transaction_By_State(ctx, payoutdb.Transaction_State(string(TxDropped)))
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return stats, nil
}

func setTransactionStatus(ctx context.Context, db payoutdb.Methods, status *TxStatus) error {
	update := payoutdb.Transaction_Update_Fields{
		State: payoutdb.Transaction_State(string(status.State)),
	}

	if status.Receipt != nil {
		receiptJSON, err := json.Marshal(status.Receipt)
		if err != nil {
			return errs.Wrap(err)
		}
		update.Receipt = payoutdb.Transaction_Receipt(string(receiptJSON))
	}

	return db.UpdateNoReturn_Transaction_By_Hash(ctx,
		payoutdb.Transaction_Hash(status.Hash),
		update,
	)
}

type Payout struct {
	CSVLine       int
	Payee         common.Address
	USD           decimal.Decimal
	PayoutGroupID int64
}

func PayoutsFromRows(rows []*payoutdb.Payout) ([]*Payout, error) {
	payouts := make([]*Payout, 0, len(rows))
	for _, row := range rows {
		payout, err := PayoutFromRow(row)
		if err != nil {
			return nil, err
		}
		payouts = append(payouts, payout)
	}
	return payouts, nil
}

func PayoutFromRow(row *payoutdb.Payout) (*Payout, error) {
	payee, err := batchpayment.AddressFromString(row.Payee)
	if err != nil {
		return nil, errs.New("unable to convert payee for payout %d: %v", row.Pk, err)
	}
	usd, err := decimal.NewFromString(row.Usd)
	if err != nil {
		return nil, errs.New("unable to convert USD for payout %d: %v", row.Pk, err)
	}
	return &Payout{
		CSVLine:       row.CsvLine,
		Payee:         payee,
		USD:           usd,
		PayoutGroupID: row.PayoutGroupId,
	}, nil
}

type PayoutGroup struct {
	ID          int64
	FinalTxHash *common.Hash
}

func PayoutGroupsFromRows(rows []*payoutdb.PayoutGroup) ([]*PayoutGroup, error) {
	payoutGroups := make([]*PayoutGroup, 0, len(rows))
	for _, row := range rows {
		payoutGroup, err := PayoutGroupFromRow(row)
		if err != nil {
			return nil, err
		}
		payoutGroups = append(payoutGroups, payoutGroup)
	}
	return payoutGroups, nil
}

func PayoutGroupFromRow(row *payoutdb.PayoutGroup) (*PayoutGroup, error) {
	var finalTxHash *common.Hash
	if row.FinalTxHash != nil {
		hash, err := batchpayment.HashFromString(*row.FinalTxHash)
		if err != nil {
			return nil, errs.New("unable to convert final tx hash for payout group pk %d: %v", row.Pk, err)
		}
		finalTxHash = &hash
	}

	return &PayoutGroup{
		ID:          row.Id,
		FinalTxHash: finalTxHash,
	}, nil
}

type Receipt struct {
	GasUsed uint64
}

type Transaction struct {
	Hash              string
	Owner             common.Address
	Spender           common.Address
	Nonce             uint64
	EstimatedGasPrice *big.Int
	StorjPrice        decimal.Decimal
	StorjTokens       *big.Int
	PayoutGroupID     int64
	Raw               []byte
	State             TxState
	Receipt           *Receipt
}

func TransactionsFromRows(rows []*payoutdb.Transaction) ([]*Transaction, error) {
	transactions := make([]*Transaction, 0, len(rows))
	for _, row := range rows {
		transaction, err := TransactionFromRow(row)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func TransactionFromRow(row *payoutdb.Transaction) (*Transaction, error) {
	owner, err := batchpayment.AddressFromString(row.Owner)
	if err != nil {
		return nil, errs.New("unable to convert owner for transaction pk %d: %v", row.Pk, err)
	}
	spender, err := batchpayment.AddressFromString(row.Spender)
	if err != nil {
		return nil, errs.New("unable to convert spender for transaction pk %d: %v", row.Pk, err)
	}
	estimatedGasPrice, ok := new(big.Int).SetString(row.EstimatedGasPrice, 10)
	if !ok {
		return nil, errs.New("unable to convert estimated gas price for transaction pk %d", row.Pk)
	}
	storjPrice, err := decimal.NewFromString(row.StorjPrice)
	if err != nil {
		return nil, errs.New("unable to convert storj price for transaction pk %d: %v", row.Pk, err)
	}
	storjTokens, ok := new(big.Int).SetString(row.StorjTokens, 10)
	if !ok {
		return nil, errs.New("unable to convert storj tokens for transaction pk %d", row.Pk)
	}

	raw := []byte(row.Raw)

	var receipt *Receipt
	if row.Receipt != nil {
		var jsonReceipt struct {
			GasUsed string `json:"gasUsed"`
		}
		if err := json.Unmarshal([]byte(*row.Receipt), &jsonReceipt); err != nil {
			return nil, errs.New("unable to convert receipt for transaction pk %d: %v", row.Pk, err)
		}
		gasUsed, err := strconv.ParseUint(jsonReceipt.GasUsed, 0, 64)
		if err != nil {
			return nil, errs.New("unable to convert gas used for transaction pk %d: %v", row.Pk, err)
		}
		receipt = &Receipt{GasUsed: gasUsed}
	}

	state, ok := TxStateFromString(row.State)
	if !ok {
		return nil, errs.New("unable to convert state for transaction pk %d: %v", row.Pk, err)
	}

	return &Transaction{
		Hash:              row.Hash,
		Owner:             owner,
		Spender:           spender,
		Nonce:             row.Nonce,
		EstimatedGasPrice: estimatedGasPrice,
		StorjPrice:        storjPrice,
		StorjTokens:       storjTokens,
		PayoutGroupID:     row.PayoutGroupId,
		Raw:               raw,
		State:             state,
		Receipt:           receipt,
	}, nil
}

type NonceGroup struct {
	Nonce         uint64
	PayoutGroupID int64
	Txs           []Transaction
}

// initDB initializes the database. It is resilient against crashes but does
// not protect against concurrent initialization. But then again, the program
// isn't intended to have more than one instance running against a given
// payout.
func initDB(ctx context.Context, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errs.Wrap(err)
	}
	tmpPath := path + ".tmp"
	db, err := openDB(path+".tmp", false)
	if err != nil {
		return errs.Wrap(err)
	}
	defer db.Close()

	if _, err := db.Exec(db.Schema()); err != nil {
		return errs.Wrap(err)
	}
	if err := db.CreateNoReturn_Metadata(ctx,
		payoutdb.Metadata_Version(dbVersion),
		payoutdb.Metadata_Attempts(0),
		payoutdb.Metadata_Create_Fields{},
	); err != nil {
		return errs.Wrap(err)
	}
	if err := db.Close(); err != nil {
		return errs.Wrap(err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

func openDB(path string, readOnly bool) (*payoutdb.DB, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	dbURI := "file:" + path + "?_journal_mode=WAL&_foreign_keys=true&_locking_mode=EXCLUSIVE"
	if readOnly {
		dbURI += "&mode=ro"
	}
	db, err := payoutdb.Open("sqlite3", dbURI)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	db.Hooks.Now = func() time.Time {
		// Row timestamps are for audit only. Nanosecond precision is overkill
		// and makes the output harder to read.
		return time.Now().Truncate(time.Millisecond)
	}

	return db, nil
}

func migrateDB(ctx context.Context, db *payoutdb.DB, version int) (err error) {
	rx := db.NewRx()
	defer func() {
		if err != nil {
			err = errs.Combine(err, rx.Rollback())
		}
	}()

	from := version
	for from < dbVersion {
		tx, err := rx.UnsafeTx(ctx)
		if err != nil {
			return err
		}

		to := from + 1
		switch to {
		case 2:
			if err := migrateV2(tx); err != nil {
				return err
			}
		default:
			return errs.New("no migration to version %d available", version)
		}

		if err := rx.Commit(); err != nil {
			return err
		}

		from = to
	}

	return nil
}

func migrateV2(tx *sql.Tx) error {
	// version 2 renamed the "payer" column in both metadata and
	// transaction tables to "owner".
	stmts := []string{
		// Rename owner in metadata table
		`CREATE TABLE __metadata_new(
			pk INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			version INTEGER NOT NULL,
			attempts INTEGER NOT NULL,
			spender TEXT,
			owner TEXT,
			PRIMARY KEY ( pk )
		);`,
		`INSERT INTO __metadata_new(pk, created_at, updated_at, version, attempts, spender, owner)
			SELECT pk, created_at, updated_at, version, attempts, spender, payer FROM metadata;`,
		`DROP TABLE metadata;`,
		`ALTER TABLE __metadata_new RENAME TO metadata;`,
		// Rename owner in tx table
		`CREATE TABLE __tx_new (
			pk INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			hash TEXT NOT NULL,
			owner TEXT NOT NULL,
			spender TEXT NOT NULL,
			nonce INTEGER NOT NULL,
			estimated_gas_price TEXT NOT NULL,
			storj_price TEXT NOT NULL,
			storj_tokens TEXT NOT NULL,
			payout_group_id INTEGER NOT NULL REFERENCES payout_group( id ),
			raw TEXT NOT NULL,
			state TEXT NOT NULL,
			receipt TEXT,
			PRIMARY KEY ( pk ),
			UNIQUE ( hash )
		);`,
		`INSERT INTO __tx_new(pk, created_at, updated_at, hash, owner, spender, nonce, estimated_gas_price, storj_price, storj_tokens, payout_group_id, raw, state, receipt)
			SELECT pk, created_at, updated_at, hash, payer, spender, nonce, estimated_gas_price, storj_price, storj_tokens, payout_group_id, raw, state, receipt FROM tx;`,
		`DROP TABLE tx;`,
		`ALTER TABLE __tx_new RENAME TO tx;`,
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return errs.Wrap(err)
		}
	}
	return nil
}
