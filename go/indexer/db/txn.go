package db

import "database/sql"

func txn(db *sql.DB, apply func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			// Ignore since we're panicking anyway
			_ = tx.Rollback()
			panic(p)
		}
	}()

	err = apply(tx)
	if err != nil {
		// Don't swallow application error
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
