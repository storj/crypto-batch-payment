// Package payoutdb provides generated database code for the payouts database
package payoutdb

//go:generate go run storj.io/dbx@723915b3a1861ddfb5431fce3890e9aa90482e05 golang -p payoutdb -d sqlite3 -t templates/ payoutdb.dbx .
