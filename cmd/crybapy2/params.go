package main

import (
	"strconv"

	"github.com/shopspring/decimal"
	"github.com/zeebo/clingy"
)

func stringFlag(params clingy.Parameters, name, desc, def string) string {
	return params.Flag(name, desc, def).(string)
}

func toggleFlag(params clingy.Parameters, name, desc string, def bool) bool {
	return params.Flag(name, desc, def, clingy.Transform(strconv.ParseBool), clingy.Boolean).(bool)
}

func optDecimalFlag(params clingy.Parameters, name, desc, def string) decimal.Decimal {
	return params.Flag(name, desc, decimal.RequireFromString(def), clingy.Transform(decimal.NewFromString)).(decimal.Decimal)
}

func stringArg(params clingy.Parameters, name, desc string) string {
	return params.Arg(name, desc).(string)
}
