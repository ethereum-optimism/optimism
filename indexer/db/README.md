# db

Files to specify the DB schema and the DAO (data-access-object) for the application to communicate with the PostgreSQL Database

- **Overview**

Most of the logic for the DB lives in [db.go](./db.go).  The Database uses raw sql with the cannonical golang sql library [dtabase/sql](https://pkg.go.dev/database/sql)

#### Schema

The schema is specified in [sql.go](./sql.go)

#### DAO

The db and it's DAO layer are specified in [db.go](./db.go)

- **NewDatabase**

```go
type Database struct {
	db     *sql.DB
	config string
}

func NewDatabase(config string) (*Database, error)
```

NewDatabase handles executing the migrations specified in [db.go](./db.go) and initializing a DB

- **DAO Methods**


The rest of the application does not write sql directly but instead uses access methods that abstract the sql queries.

- **Example*

```go
func (d *Database) AddL1Token(address string, token *Token) error {
	const insertTokenStatement = `
	INSERT INTO l1_tokens
		(address, name, symbol, decimals)
	VALUES
		($1, $2, $3, $4)
	`

	return txn(d.db, func(tx *sql.Tx) error {
		_, err := tx.Exec(
			insertTokenStatement,
			address,
			token.Name,
			token.Symbol,
			token.Decimals,
		)
		return err
	})
}
```

The types for the DAO are specified in type files such as [l1block.go](./l1block.go)

- **See also:**

- [database.sql](https://pkg.go.dev/database/sql)
