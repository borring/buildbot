package main

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func testingDB(t *testing.T, fn func(*db)) {
	dsn := &mysql.Config{
		User:      "moymoy",
		Passwd:    "paspass",
		DBName:    "tickets_testing",
		Net:       "tcp",
		Addr:      "localhost:3306",
		ParseTime: true,
	}
	d, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}

	fn(&db{d})

	d.Close()
}

func TestTransaction(t *testing.T) {
	testingDB(t, func(d *db) {
		wg := sync.WaitGroup{}
		for i := 'a'; i < 'g'; i++ {
			wg.Add(1)
			go func() {
				d.transaction(func(tx *sql.Tx) ([]interface{}, *sql.Stmt, error) {
					stmt, err := tx.Prepare("INSERT INTO txtest (first, second, now) VALUES (?, ?, ?)")
					return []interface{}{string(i), i, time.Now().Format(timestampfmt)}, stmt, err
				})
				wg.Done()
			}()
		}
	})
}
