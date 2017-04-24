package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type txFunc func(*sql.Tx) ([]interface{}, *sql.Stmt, error)

type db struct {
	*sql.DB
}

func (d *db) transaction(fn txFunc) error {
	var tx *sql.Tx
	var err error

	if tx, err = d.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction: %s\n", err)
	}

	args, stmt, err := fn(tx)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s\n", err)
	}
	defer stmt.Close()

	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to exec statemnt: %s\n", err)
	}

	// sleep for 1 second because the current implementation of `tid` in the
	// tickets schema is based on timestamp. Having two tickets submitted within
	// the second will cause a conflict
	time.Sleep(time.Second)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %s\n", err)
	}

	return nil
}
