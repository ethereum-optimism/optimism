package db

import (
	"database/sql"
	"errors"
	"math/big"

	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq"
)

var createL1BlocksTable = `
CREATE TABLE IF NOT EXISTS l1_blocks (
	hash TEXT NOT NULL PRIMARY KEY,
	parent_hash TEXT NOT NULL,
	number INTEGER NOT NULL,
	timestamp INTEGER NOT NULL
)
`

var createL2BlocksTable = `
CREATE TABLE IF NOT EXISTS l2_blocks (
	hash TEXT NOT NULL PRIMARY KEY,
	parent_hash TEXT NOT NULL,
	number INTEGER NOT NULL,
	timestamp INTEGER NOT NULL
)
`

var createDepositsTable = `
CREATE TABLE IF NOT EXISTS deposits (
	from_address TEXT NOT NULL,
	to_address TEXT NOT NULL,
	l1_token TEXT NOT NULL REFERENCES l1_tokens(address),
	l2_token TEXT NOT NULL,
	amount TEXT NOT NULL,
	data BYTEA NOT NULL,
	log_index INTEGER NOT NULL,
	block_hash TEXT NOT NULL REFERENCES l1_blocks(hash) ,
	block_timestamp TEXT NOT NULL,
	tx_hash TEXT NOT NULL
)
`

var createL1TokensTable = `
CREATE TABLE IF NOT EXISTS l1_tokens (
	address TEXT NOT NULL PRIMARY KEY,
	name TEXT NOT NULL,
	symbol TEXT NOT NULL UNIQUE,
	decimals INTEGER NOT NULL
)
`

var createWithdrawalsTable = `
CREATE TABLE IF NOT EXISTS withdrawals (
	from_address TEXT NOT NULL,
	to_address TEXT NOT NULL,
	l1_token TEXT NOT NULL,
	l2_token TEXT NOT NULL,
	amount TEXT NOT NULL,
	data BYTEA NOT NULL,
	log_index INTEGER NOT NULL,
	block_hash TEXT NOT NULL REFERENCES l2_blocks(hash) ,
	block_timestamp TEXT NOT NULL,
	tx_hash TEXT NOT NULL
)
`

var insertETHL1Token = `
	INSERT INTO l1_tokens
		(address, name, symbol, decimals)
	VALUES ('0x0000000000000000000000000000000000000000', 'Ethereum', 'ETH', 18);
`

type PaginationParam struct {
	Limit  uint64
	Offset uint64
}

var schema = []string{
	createL1BlocksTable,
	createL2BlocksTable,
	createL1TokensTable,
	insertETHL1Token,
	createDepositsTable,
	createWithdrawalsTable,
}

type TxnEnqueuedEvent struct {
	BlockNumber uint64
	Timestamp   uint64
	TxHash      common.Hash
	Data        []byte
}

func (e TxnEnqueuedEvent) String() string {
	return e.TxHash.String()
}

type Deposit struct {
	TxHash      common.Hash
	L1Token     common.Address
	L2Token     common.Address
	FromAddress common.Address
	ToAddress   common.Address
	Amount      *big.Int
	Data        []byte
	LogIndex    uint
}

type Withdrawal struct {
	TxHash      l2common.Hash
	L1Token     l2common.Address
	L2Token     l2common.Address
	FromAddress l2common.Address
	ToAddress   l2common.Address
	Amount      *big.Int
	Data        []byte
	LogIndex    uint
}

type IndexedL1Block struct {
	Hash       common.Hash
	ParentHash common.Hash
	Number     uint64
	Timestamp  uint64
	Deposits   []Deposit
}

func (b IndexedL1Block) String() string {
	return b.Hash.String()
}

type IndexedL2Block struct {
	Hash        l2common.Hash
	ParentHash  l2common.Hash
	Number      uint64
	Timestamp   uint64
	Withdrawals []Withdrawal
}

func (b IndexedL2Block) String() string {
	return b.Hash.String()
}

type TokenBridgeMessage struct {
	FromAddress    string `json:"from"`
	ToAddress      string `json:"to"`
	L1Token        string `json:"l1token"`
	L2Token        string `json:"l2token"`
	Amount         string `json:"amount"`
	Data           []byte `json:"data"`
	LogIndex       uint64 `json:"logIndex"`
	BlockNumber    uint64 `json:"blockNumber"`
	BlockTimestamp string `json:"blockTimestamp"`
	TxHash         string `json:"transactionHash"`
}

func (d Deposit) String() string {
	return d.TxHash.String()
}

func (b *IndexedL1Block) Events() []TxnEnqueuedEvent {
	nDeposits := len(b.Deposits)
	if nDeposits == 0 {
		return nil
	}

	var events = make([]TxnEnqueuedEvent, 0, nDeposits)
	for _, deposit := range b.Deposits {
		events = append(events, TxnEnqueuedEvent{
			BlockNumber: b.Number,
			Timestamp:   b.Timestamp,
			TxHash:      deposit.TxHash,
			Data:        deposit.Data, // TODO: copy?
		})
	}

	return events
}

type Token struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

type Database struct {
	db     *sql.DB
	config string
}

func (d *Database) GetL1TokenByAddress(address string) (*Token, error) {
	const selectL1TokenStatement = `
	SELECT name, symbol, decimals FROM l1_tokens WHERE address = $1;
	`

	var token *Token
	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectL1TokenStatement)
		if err != nil {
			return err
		}

		rows, err := queryStmt.Query(address)
		if err != nil {
			return err
		}

		if !rows.Next() {
			return nil
		}

		var name string
		var symbol string
		var decimals uint8
		err = rows.Scan(&name, &symbol, &decimals)
		if err != nil {
			return err
		}

		if rows.Next() {
			return errors.New("address should be unique")
		}

		token = &Token{
			Name:     name,
			Symbol:   symbol,
			Decimals: decimals,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (d *Database) AddL1Token(address string, token *Token) error {
	const insertTokenStatement = `
	INSERT INTO l1_tokens
		(address, name, symbol, decimals)
	VALUES
		($1, $2, $3, $4)
	`

	return txn(d.db, func(tx *sql.Tx) error {
		tokenStmt, err := tx.Prepare(insertTokenStatement)
		if err != nil {
			return err
		}

		_, err = tokenStmt.Exec(
			address,
			token.Name,
			token.Symbol,
			token.Decimals,
		)
		if err != nil {
			return err
		}

		return nil
	})
}

func (d *Database) AddIndexedL1Block(block *IndexedL1Block) error {
	const insertBlockStatement = `
	INSERT INTO l1_blocks
		(hash, parent_hash, number, timestamp)
	VALUES
		($1, $2, $3, $4)
	`

	const insertDepositStatement = `
	INSERT INTO deposits
		(from_address, to_address, l1_token, l2_token, amount, tx_hash, log_index, block_hash, block_timestamp, data)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	return txn(d.db, func(tx *sql.Tx) error {
		blockStmt, err := tx.Prepare(insertBlockStatement)
		if err != nil {
			return err
		}

		_, err = blockStmt.Exec(
			block.Hash.String(),
			block.ParentHash.String(),
			block.Number,
			block.Timestamp,
		)
		if err != nil {
			return err
		}

		if len(block.Deposits) == 0 {
			return nil
		}

		depositStmt, err := tx.Prepare(insertDepositStatement)
		if err != nil {
			return err
		}

		for _, deposit := range block.Deposits {
			_, err = depositStmt.Exec(
				deposit.FromAddress.String(),
				deposit.ToAddress.String(),
				deposit.L1Token.String(),
				deposit.L1Token.String(),
				deposit.Amount.String(),
				deposit.TxHash.String(),
				deposit.LogIndex,
				block.Hash.String(),
				block.Timestamp,
				deposit.Data,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Database) AddIndexedL2Block(block *IndexedL2Block) error {
	const insertBlockStatement = `
	INSERT INTO l2_blocks
		(hash, parent_hash, number, timestamp)
	VALUES
		($1, $2, $3, $4)
	`

	const insertWithdrawalStatement = `
	INSERT INTO withdrawals
		(from_address, to_address, l1_token, l2_token, amount, tx_hash, log_index, block_hash, block_timestamp, data)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	return txn(d.db, func(tx *sql.Tx) error {
		blockStmt, err := tx.Prepare(insertBlockStatement)
		if err != nil {
			return err
		}

		_, err = blockStmt.Exec(
			block.Hash.String(),
			block.ParentHash.String(),
			block.Number,
			block.Timestamp,
		)
		if err != nil {
			return err
		}

		if len(block.Withdrawals) == 0 {
			return nil
		}

		withdrawalStmt, err := tx.Prepare(insertWithdrawalStatement)
		if err != nil {
			return err
		}

		for _, withdrawal := range block.Withdrawals {
			_, err = withdrawalStmt.Exec(
				withdrawal.FromAddress.String(),
				withdrawal.ToAddress.String(),
				withdrawal.L1Token.String(),
				withdrawal.L1Token.String(),
				withdrawal.Amount.String(),
				withdrawal.TxHash.String(),
				withdrawal.LogIndex,
				block.Hash.String(),
				block.Timestamp,
				withdrawal.Data,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Database) GetDepositsByAddress(address common.Address, page PaginationParam) ([]TokenBridgeMessage, error) {
	const selectDepositsStatement = `
	SELECT
		deposits.from_address, deposits.to_address,
		deposits.l1_token, deposits.l2_token,
		deposits.amount, deposits.tx_hash, deposits.data,
		l1_blocks.number, l1_blocks.timestamp
	FROM deposits
		INNER JOIN l1_blocks ON deposits.block_hash=l1_blocks.hash
	WHERE deposits.from_address = $1 ORDER BY deposits.block_timestamp LIMIT $2 OFFSET $3;
	`
	var deposits []TokenBridgeMessage

	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectDepositsStatement)
		if err != nil {
			return err
		}

		rows, err := queryStmt.Query(address.String(), page.Limit, page.Offset)
		if err != nil {
			return err
		}

		for rows.Next() {
			var deposit TokenBridgeMessage
			if err := rows.Scan(
				&deposit.FromAddress, &deposit.ToAddress,
				&deposit.L1Token, &deposit.L2Token,
				&deposit.Amount, &deposit.TxHash, &deposit.Data,
				&deposit.BlockNumber, &deposit.BlockTimestamp,
			); err != nil {
				return err
			}
			deposits = append(deposits, deposit)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return deposits, nil
}

func (d *Database) GetWithdrawalsByAddress(address l2common.Address, page PaginationParam) ([]TokenBridgeMessage, error) {
	const selectWithdrawalsStatement = `
	SELECT
		withdrawals.from_address, withdrawals.to_address,
		withdrawals.l1_token, withdrawals.l2_token,
		withdrawals.amount, withdrawals.tx_hash, withdrawals.data,
		l2_blocks.number, l2_blocks.timestamp
	FROM withdrawals
		INNER JOIN l2_blocks ON withdrawals.block_hash=l2_blocks.hash
	WHERE withdrawals.from_address = $1 ORDER BY withdrawals.block_timestamp LIMIT $2 OFFSET $3;
	`
	var withdrawals []TokenBridgeMessage

	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectWithdrawalsStatement)
		if err != nil {
			return err
		}

		rows, err := queryStmt.Query(address.String(), page.Limit, page.Offset)
		if err != nil {
			return err
		}

		for rows.Next() {
			var withdrawal TokenBridgeMessage
			if err := rows.Scan(
				&withdrawal.FromAddress, &withdrawal.ToAddress,
				&withdrawal.L1Token, &withdrawal.L2Token,
				&withdrawal.Amount, &withdrawal.TxHash, &withdrawal.Data,
				&withdrawal.BlockNumber, &withdrawal.BlockTimestamp,
			); err != nil {
				return err
			}
			withdrawals = append(withdrawals, withdrawal)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return withdrawals, nil
}

type L1BlockLocator struct {
	Number uint64      `json:"number"`
	Hash   common.Hash `json:"hash"`
}

type L2BlockLocator struct {
	Number uint64        `json:"number"`
	Hash   l2common.Hash `json:"hash"`
}

func (d *Database) GetHighestL1Block() (*L1BlockLocator, error) {
	const selectHighestBlockStatement = `
	SELECT number, hash FROM l1_blocks ORDER BY number DESC LIMIT 1
	`

	var highestBlock *L1BlockLocator
	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectHighestBlockStatement)
		if err != nil {
			return err
		}

		rows, err := queryStmt.Query()
		if err != nil {
			return err
		}

		if !rows.Next() {
			return nil
		}

		var number uint64
		var hash string
		err = rows.Scan(&number, &hash)
		if err != nil {
			return err
		}

		if rows.Next() {
			return errors.New("number of rows should be at most 1 since LIMIT is 1")
		}

		highestBlock = &L1BlockLocator{
			Number: number,
			Hash:   common.HexToHash(hash),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return highestBlock, nil
}

func (d *Database) GetHighestL2Block() (*L2BlockLocator, error) {
	const selectHighestBlockStatement = `
	SELECT number, hash FROM l2_blocks ORDER BY number DESC LIMIT 1
	`

	var highestBlock *L2BlockLocator
	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectHighestBlockStatement)
		if err != nil {
			return err
		}

		rows, err := queryStmt.Query()
		if err != nil {
			return err
		}

		if !rows.Next() {
			return nil
		}

		var number uint64
		var hash string
		err = rows.Scan(&number, &hash)
		if err != nil {
			return err
		}

		if rows.Next() {
			return errors.New("number of rows should be at most 1 since LIMIT is 1")
		}

		highestBlock = &L2BlockLocator{
			Number: number,
			Hash:   l2common.HexToHash(hash),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return highestBlock, nil
}

func (d *Database) GetIndexedL1BlockByHash(hash common.Hash) (*IndexedL1Block, error) {
	const selectBlockByHashStatement = `
	SELECT
		hash, parent_hash, number, timestamp
	FROM l1_blocks
	WHERE hash = $1
	`

	var block *IndexedL1Block
	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectBlockByHashStatement)
		if err != nil {
			return err
		}

		rows, err := queryStmt.Query(hash.String())
		if err != nil {
			return err
		}

		if !rows.Next() {
			return nil
		}

		var hash string
		var parentHash string
		var number uint64
		var timestamp uint64
		err = rows.Scan(&hash, &parentHash, &number, &timestamp)
		if err != nil {
			return err
		}

		block = &IndexedL1Block{
			Hash:       common.HexToHash(hash),
			ParentHash: common.HexToHash(parentHash),
			Number:     number,
			Timestamp:  timestamp,
			Deposits:   nil,
		}

		if rows.Next() {
			return errors.New("number of rows should be at most 1 since hash is pk")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return block, nil

}
func (d *Database) GetEventsByBlockHash(hash common.Hash) ([]TxnEnqueuedEvent, error) {
	const selectEventsByBlockHashStatement = `
	SELECT
		b.number, b.timestamp,
		d.tx_hash, d.data
	FROM
		blocks AS b,
		deposits AS d
	WHERE b.hash = d.block_hash AND b.hash = $1
	`

	var events []TxnEnqueuedEvent
	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectEventsByBlockHashStatement)
		if err != nil {
			return err
		}

		rows, err := queryStmt.Query(hash.String())
		if err != nil {
			return err
		}

		for rows.Next() {
			event, err := scanTxnEnqueuedEvent(rows)
			if err != nil {
				return err
			}

			events = append(events, event)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return events, nil
}

func scanTxnEnqueuedEvent(rows *sql.Rows) (TxnEnqueuedEvent, error) {
	var number uint64
	var timestamp uint64
	var txHash string
	var data []byte
	err := rows.Scan(
		&number,
		&timestamp,
		&txHash,
		&data,
	)
	if err != nil {
		return TxnEnqueuedEvent{}, err
	}

	return TxnEnqueuedEvent{
		BlockNumber: number,
		Timestamp:   timestamp,
		TxHash:      common.HexToHash(txHash),
		Data:        data,
	}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) Config() string {
	return d.config
}

func NewDatabase(config string) (*Database, error) {
	db, err := sql.Open("postgres", config)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	for _, migration := range schema {
		_, err = db.Exec(migration)
		if err != nil {
			return nil, err
		}
	}

	return &Database{
		db:     db,
		config: config,
	}, nil
}

func txn(db *sql.DB, apply func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
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
