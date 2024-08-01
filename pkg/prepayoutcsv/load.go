// Package prepayoutcsv provides functions for loading prepayout CSV files
package prepayoutcsv

import (
	"bytes"
	"encoding/csv"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
)

const (
	expectHeader = "address,amount,address-kind,mandatory,sanctioned,bonus"
)

type Row struct {
	// Line number in the CSV file
	Line int

	// Address is the destination address of the payout
	Address common.Address `csv:"address"`

	// Amount is the payout mount in USD
	Amount decimal.Decimal `csv:"amount"`

	// Kind is the type of payout (eth, zksync2)
	Kind string `csv:"address-kind"`

	// Mandatory is whether or not this payout should be issued even if
	// it falls below the payment threshold.
	Mandatory bool `csv:"mandatory"`

	// Sanctioned is whether or not the payee is sanctioned. These rows are
	// skipped when issuing payouts.
	Sanctioned bool `csv:"sanctioned"`

	// Bonus indicates whether or not this payee qualifies for a bonus.
	Bonus bool `csv:"bonus"`
}

func Load(path string) ([]Row, error) {
	csvBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return Parse(csvBytes)
}

func Parse(csvBytes []byte) ([]Row, error) {
	r := csv.NewReader(bytes.NewReader(csvBytes))
	r.FieldsPerRecord = -1
	r.Comment = '#'
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, errs.Wrap(err)
	}

	const expectFields = 6

	inHeader := true
	var rows []Row
	for i, record := range records {
		// TODO: this isn't quite accurate. It does not accurately count
		// commented lines and empty lines. We could manually handle commented
		// lines but the csv package skips empty lines so we might as well have
		// it handle comments too since the line number will be inaccurate
		// anyway.
		line := i + 1

		// first non-empty, non-comment line must be the header
		if inHeader {
			inHeader = false
			header := strings.Join(record, ",")
			if header != expectHeader {
				return nil, errs.New("record on line %d: invalid header %q: expected %q", line, header, expectHeader)
			}
			continue
		}

		// wrong number of fields
		if len(record) != expectFields {
			return nil, errs.New("record on line %d: expected %d fields but got %d", line, expectFields, len(record))
		}

		var (
			addressValue    = record[0]
			amountValue     = record[1]
			kindValue       = record[2]
			mandatoryValue  = record[3]
			sanctionedValue = record[4]
			bonusValue      = record[5]
		)

		// The address field is sometimes empty in the prepayouts CSVs. They
		// are included in the results for statistics purposes and will be
		// ignored in a later step in the process.
		var address common.Address
		if len(addressValue) > 0 {
			var ok bool
			address, ok = parseAddress(addressValue)
			if !ok {
				return nil, errs.New("record on line %d: invalid ETH address %q", line, addressValue)
			}
		}

		amount, err := decimal.NewFromString(amountValue)
		if err != nil {
			return nil, errs.New("record on line %d: invalid amount %q: %v", line, amountValue, err)
		}

		if kindValue == "" {
			return nil, errs.New("record on line %d: invalid kind %q: cannot be empty", line, kindValue)
		}

		mandatory, err := strconv.ParseBool(mandatoryValue)
		if err != nil {
			return nil, errs.New(`record on line %d: invalid boolean value %q for "mandatory" column`, line, mandatoryValue)
		}

		sanctioned, err := strconv.ParseBool(sanctionedValue)
		if err != nil {
			return nil, errs.New(`record on line %d: invalid boolean value %q for "sanctioned" column`, line, sanctionedValue)
		}

		bonus, err := strconv.ParseBool(bonusValue)
		if err != nil {
			return nil, errs.New(`record on line %d: invalid boolean value %q for "bonus" column`, line, bonusValue)
		}

		rows = append(rows, Row{
			Line:       line,
			Address:    address,
			Amount:     amount,
			Kind:       kindValue,
			Mandatory:  mandatory,
			Sanctioned: sanctioned,
			Bonus:      bonus,
		})
	}

	return rows, nil
}

func parseAddress(s string) (common.Address, bool) {
	if !common.IsHexAddress(s) {
		return common.Address{}, false
	}
	return common.HexToAddress(s), true
}
