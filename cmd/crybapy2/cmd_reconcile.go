package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/zeebo/clingy"
	"storj.io/crypto-batch-payment/pkg/config"
	"storj.io/crypto-batch-payment/pkg/fancy"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

type cmdReconcile struct {
	config   string
	apply    bool
	complete []completeSpec
}

type completeSpec struct {
	PayoutGroupID int64
	Hash          string
}

func parseCompleteSpec(s string) (completeSpec, error) {
	i := strings.Index(s, ":")
	if i <= 0 || i == len(s)-1 {
		return completeSpec{}, fmt.Errorf("expected --complete=<payout-group-id>:<hash>, got %q", s)
	}
	id, err := strconv.ParseInt(s[:i], 10, 64)
	if err != nil {
		return completeSpec{}, fmt.Errorf("invalid payout group id in %q: %w", s, err)
	}
	return completeSpec{PayoutGroupID: id, Hash: s[i+1:]}, nil
}

func (cmd *cmdReconcile) Setup(params clingy.Parameters) {
	cmd.config = stringFlag(params, "config", "The configuration file", "./config.toml")
	cmd.apply = toggleFlag(params, "apply", "Write changes to the DB (default: dry-run)", false)
	cmd.complete = params.Flag("complete",
		`Mark a payout group complete with an externally-verified confirmed tx hash not present in the DB. Format: "<payout-group-id>:<hash>". Repeatable.`,
		[]completeSpec(nil),
		clingy.Repeated,
		clingy.Transform(parseCompleteSpec),
	).([]completeSpec)
}

func (cmd *cmdReconcile) Execute(ctx context.Context) error {
	stdout := clingy.Stdout(ctx)
	stderr := clingy.Stderr(ctx)

	cfg, err := config.Load(cmd.config)
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	auditors, err := cfg.NewAuditors(ctx)
	if err != nil {
		return fmt.Errorf("failed to init auditors: %w", err)
	}
	defer auditors.Close()

	dbs, err := loadDBs(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = dbs.Close() }()

	if !cmd.apply {
		fancy.Fwarnln(stdout, "DRY RUN: no changes will be written. Re-run with --apply to persist.")
	}

	completes := groupCompletesByGroupID(cmd.complete)
	usedCompletes := make(map[int64]bool)

	var anyBad bool
	for payerType, db := range dbs {
		auditor, ok := auditors[payerType]
		if !ok {
			fancy.Ferrorf(stderr, "No auditor for payer type %q\n", payerType)
			anyBad = true
			continue
		}

		fancy.Finfof(stdout, "Reconciling %q pending transactions...\n", payerType)
		bad, err := reconcilePayer(ctx, stdout, stderr, payerType, db, auditor, completes, usedCompletes, cmd.apply)
		if err != nil {
			return err
		}
		if bad {
			anyBad = true
		}
	}

	for _, spec := range cmd.complete {
		if !usedCompletes[spec.PayoutGroupID] {
			fancy.Ferrorf(stderr, "--complete for payout group %d did not match any payer's data\n", spec.PayoutGroupID)
			anyBad = true
		}
	}

	if anyBad {
		fancy.Ferrorln(stderr, "Reconcile finished with warnings; see above.")
	}
	fancy.Finfoln(stdout, "Done.")
	return nil
}

func groupCompletesByGroupID(specs []completeSpec) map[int64]completeSpec {
	m := make(map[int64]completeSpec, len(specs))
	for _, s := range specs {
		m[s.PayoutGroupID] = s
	}
	return m
}

// reconcilePayer inspects pending txs for one payer, decides an action per
// payout-group/nonce, and applies it (or prints it) depending on apply.
// Returns bad=true if any warning was emitted.
func reconcilePayer(
	ctx context.Context,
	stdout, stderr io.Writer,
	payerType payer.Type,
	db *pipelinedb.DB,
	auditor payer.Auditor,
	completes map[int64]completeSpec,
	usedCompletes map[int64]bool,
	apply bool,
) (bool, error) {
	allTxs, err := db.FetchTransactions(ctx)
	if err != nil {
		return false, err
	}

	var pending []*pipelinedb.Transaction
	for _, tx := range allTxs {
		if tx.State == pipelinedb.TxPending {
			pending = append(pending, tx)
		}
	}
	if len(pending) == 0 && len(completes) == 0 {
		fancy.Finfof(stdout, "  no pending transactions.\n")
		return false, nil
	}

	// Query chain state for each pending tx.
	states := make(map[string]pipelinedb.TxState, len(pending))
	for _, tx := range pending {
		state, err := auditor.CheckTransactionState(ctx, tx.Hash)
		if err != nil {
			return false, fmt.Errorf("failed to check tx %s: %w", tx.Hash, err)
		}
		states[tx.Hash] = state
	}

	// Group pending txs by payout group.
	byGroup := make(map[int64][]*pipelinedb.Transaction)
	for _, tx := range pending {
		byGroup[tx.PayoutGroupID] = append(byGroup[tx.PayoutGroupID], tx)
	}

	// Also make sure any --complete spec's group is visited even if it has no
	// pending rows recorded here (unusual, but handle it gracefully).
	for id := range completes {
		if _, seen := byGroup[id]; !seen {
			byGroup[id] = nil
		}
	}

	var bad bool
	sortedGroupIDs := make([]int64, 0, len(byGroup))
	for id := range byGroup {
		sortedGroupIDs = append(sortedGroupIDs, id)
	}
	sort.Slice(sortedGroupIDs, func(i, j int) bool { return sortedGroupIDs[i] < sortedGroupIDs[j] })

	for _, groupID := range sortedGroupIDs {
		groupPending := byGroup[groupID]

		// Bucket A: operator supplied a confirming hash.
		if spec, ok := completes[groupID]; ok {
			usedCompletes[groupID] = true

			state, err := auditor.CheckConfirmedTransactionState(ctx, spec.Hash)
			if err != nil {
				fancy.Ferrorf(stderr, "  payout-group %d: --complete hash %s could not be verified on chain: %v\n",
					groupID, spec.Hash, err)
				bad = true
				continue
			}
			if state != pipelinedb.TxConfirmed {
				fancy.Ferrorf(stderr, "  payout-group %d: --complete hash %s is not confirmed on chain (state=%s); refusing\n",
					groupID, spec.Hash, state)
				bad = true
				continue
			}

			var droppedHashes []string
			for _, tx := range groupPending {
				droppedHashes = append(droppedHashes, tx.Hash)
			}
			fancy.Finfof(stdout, "  payout-group %d: import external confirming tx %s; mark %d recorded pending tx(s) dropped\n",
				groupID, spec.Hash, len(droppedHashes))
			if apply {
				if err := db.MarkPayoutGroupComplete(ctx, groupID, spec.Hash, droppedHashes); err != nil {
					return false, fmt.Errorf("failed to mark payout group %d complete: %w", groupID, err)
				}
			}
			continue
		}

		// Classify chain states within this group.
		var confirmed, dropped, failed, stillPending []*pipelinedb.Transaction
		for _, tx := range groupPending {
			switch states[tx.Hash] {
			case pipelinedb.TxConfirmed:
				confirmed = append(confirmed, tx)
			case pipelinedb.TxDropped:
				dropped = append(dropped, tx)
			case pipelinedb.TxFailed:
				failed = append(failed, tx)
			case pipelinedb.TxPending:
				stillPending = append(stillPending, tx)
			}
		}

		switch {
		case len(confirmed) > 1:
			// Multiple confirmed txs in the same payout group means chain
			// paid the same group twice. Serious — surface it and do nothing.
			hashes := make([]string, 0, len(confirmed))
			for _, tx := range confirmed {
				hashes = append(hashes, tx.Hash)
			}
			fancy.Ferrorf(stderr, "  payout-group %d: %d confirmed txs on chain (DOUBLE PAY): %s; leaving DB unchanged\n",
				groupID, len(confirmed), strings.Join(hashes, ", "))
			bad = true

		case len(confirmed) == 1:
			// Bucket B: one tx confirmed, others dropped/failed.
			ng := &pipelinedb.NonceGroup{PayoutGroupID: groupID}
			var statuses []*pipelinedb.TxStatus
			for _, tx := range groupPending {
				statuses = append(statuses, &pipelinedb.TxStatus{
					Hash:  tx.Hash,
					State: states[tx.Hash],
				})
			}
			fancy.Finfof(stdout, "  payout-group %d: confirmed=%s; drop %d, fail %d, keep-pending %d\n",
				groupID, confirmed[0].Hash, len(dropped), len(failed), len(stillPending))
			if apply {
				if err := db.FinalizeNonceGroup(ctx, ng, statuses); err != nil {
					return false, fmt.Errorf("failed to finalize payout group %d: %w", groupID, err)
				}
			}

		case len(stillPending) > 0:
			// Something is still in flight — leave it alone.
			fancy.Finfof(stdout, "  payout-group %d: %d tx(s) still pending on chain; leaving alone\n",
				groupID, len(stillPending))

		case len(failed) > 0 && len(dropped) == 0:
			// All accounted-for txs failed. Record failure; don't touch payout group.
			fancy.Fwarnf(stderr, "  payout-group %d: all %d recorded tx(s) FAILED on chain; marking failed; payout group not completed\n",
				groupID, len(failed))
			bad = true
			if apply {
				for _, tx := range failed {
					if err := db.UpdateTransactionState(ctx, tx.Hash, pipelinedb.TxFailed); err != nil {
						return false, err
					}
				}
			}

		default:
			// All dropped (with possible failed mixed in). Bucket C or A.
			// Report and mark them dropped. The pipeline will re-attempt
			// this payout group as unattached-unfinished after this.
			owner := ""
			var nonce uint64
			if len(groupPending) > 0 {
				owner = groupPending[0].Owner.String()
				nonce = groupPending[0].Nonce
			}
			fancy.Fwarnf(stderr, "  payout-group %d: all %d recorded tx(s) DROPPED on chain (nonce=%d owner=%s).\n",
				groupID, len(dropped)+len(failed), nonce, owner)
			fancy.Fwarnf(stderr, "    If nonce %d is consumed on chain, look up %s's tx at that nonce and re-run with --complete %d:<hash>.\n",
				nonce, owner, groupID)
			fancy.Fwarnf(stderr, "    Otherwise the pipeline will re-attempt this payout group at the current chain nonce.\n")
			bad = true
			if apply {
				for _, tx := range groupPending {
					state := states[tx.Hash]
					if state != pipelinedb.TxDropped && state != pipelinedb.TxFailed {
						state = pipelinedb.TxDropped
					}
					if err := db.UpdateTransactionState(ctx, tx.Hash, state); err != nil {
						return false, err
					}
				}
			}
		}
	}

	return bad, nil
}
