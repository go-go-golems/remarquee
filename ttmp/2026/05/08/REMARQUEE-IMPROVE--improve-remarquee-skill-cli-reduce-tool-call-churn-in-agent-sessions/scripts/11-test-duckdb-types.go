package main

import (
	"database/sql"
	"fmt"
	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT COUNT(*) AS cnt, 42 AS fixed, 3.14 AS pi")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			scanArgs[i] = &values[i]
		}
		rows.Scan(scanArgs...)
		for i, col := range columns {
			fmt.Printf("%s: type=%T value=%v\n", col, values[i], values[i])
		}
	}
}
