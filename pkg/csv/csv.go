// Package csv provides functions for loading the payouts CSV file
package csv

import (
	"bytes"
	"encoding/csv"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
)

var (
	usdHeaders = []string{"usdAmnt", "amnt"}
)

type Row struct {
	// Line number in the CSV file
	Line int

	// Address is the destination address of the payout
	Address common.Address

	// USD is the payout amount in USD
	USD decimal.Decimal
}

func Load(path string) ([]Row, error) {
	csvBytes, err := ioutil.ReadFile(path)
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

	inHeader := true
	var rows []Row
	for i, record := range records {
		line := i + 1
		switch {
		// skip empty lines and comments
		case record[0] == "", record[0][0] == '#':
			continue
		// wrong number of fields
		case len(record) != 2:
			return nil, errs.New("record on line %d: wrong number of fields", line)
		}

		// first non-empty, non-comment line must be the header
		if inHeader {
			if record[0] != "addr" && !stringInSet(record[1], usdHeaders) {
				return nil, errs.New("record on line %d: invalid header %q; expected \"addr,usdAmnt\"", line, strings.Join(record, ","))
			}
			inHeader = false
			continue
		}

		address, ok := parseAddress(record[0])
		if !ok {
			return nil, errs.New("record on line %d: invalid ETH address %q", line, record[0])
		}

		usd, err := decimal.NewFromString(record[1])
		if err != nil {
			return nil, errs.New("record on line %d: invalid amount %q: %v", line, record[1], err)
		}
		if !usd.IsPositive() {
			return nil, errs.New("record on line %d: invalid amount %q: must be a positive value", line, record[1])
		}

		rows = append(rows, Row{
			Line:    line,
			Address: address,
			USD:     usd,
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

func SortRows(rows []Row) {
	sort.Slice(rows, func(i, j int) bool {
		cmp := bytes.Compare(rows[i].Address[:], rows[j].Address[:])
		if cmp < 0 {
			return true
		}
		if cmp > 0 {
			return false
		}

		return rows[i].USD.LessThan(rows[j].USD)
	})
}

func stringInSet(s string, ss []string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}
