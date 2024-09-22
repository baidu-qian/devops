/*
 * @Author: magician
 * @Date: 2024-06-15 22:33:25
 * @LastEditors: magician
 * @LastEditTime: 2024-06-15 23:32:48
 * @FilePath: /go/src/go_code/postgres/postgres.go
 * @Description:
 *
 * Copyright (c) 2024 by ${git_name_email}, All Rights Reserved.
 */
package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func checkPostgres(db *sql.DB) bool {
	result, err := db.Query("SELECT 1")
	if err != nil {
		return false
	}
	defer result.Close()

	var value int
	for result.Next() {
		err := result.Scan(&value)
		if err != nil {
			return false
		}
	}

	if value == 1 {
		return true
	} else {
		return false
	}
}

func main() {
	connStr := "host=172.16.xx.xx  user=postgres password=123456  dbname=postgres  sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Error opening database:", err)
	}
	defer db.Close()
	if checkPostgres(db) {
		fmt.Println("postgres is working correctly.")
	} else {
		fmt.Println("postgres is not working correctly.")
	}

}
