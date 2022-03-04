package db

import (
	"database/sql"
	"errors"
	"math/big"

	"github.com/google/uuid"

	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq"
)

const createL1BlocksTable = `
CREATE TABLE IF NOT EXISTS l1_blocks (
	hash VARCHAR NOT NULL PRIMARY KEY,
	parent_hash VARCHAR NOT NULL,
	number INTEGER NOT NULL,
	timestamp INTEGER NOT NULL
)
`

const createL2BlocksTable = `
CREATE TABLE IF NOT EXISTS l2_blocks (
	hash VARCHAR NOT NULL PRIMARY KEY,
	parent_hash VARCHAR NOT NULL,
	number INTEGER NOT NULL,
	timestamp INTEGER NOT NULL
)
`

const createDepositsTable = `
CREATE TABLE IF NOT EXISTS deposits (
	guid VARCHAR PRIMARY KEY NOT NULL,
	from_address VARCHAR NOT NULL,
	to_address VARCHAR NOT NULL,
	l1_token VARCHAR NOT NULL REFERENCES l1_tokens(address),
	l2_token VARCHAR NOT NULL,
	amount VARCHAR NOT NULL,
	data BYTEA NOT NULL,
	log_index INTEGER NOT NULL,
	block_hash VARCHAR NOT NULL REFERENCES l1_blocks(hash),
	tx_hash VARCHAR NOT NULL
)
`

const createL1TokensTable = `
CREATE TABLE IF NOT EXISTS l1_tokens (
	address VARCHAR NOT NULL PRIMARY KEY,
	name VARCHAR NOT NULL,
	symbol VARCHAR NOT NULL,
	decimals INTEGER NOT NULL
)
`

const createL2TokensTable = `
CREATE TABLE IF NOT EXISTS l2_tokens (
	address TEXT NOT NULL PRIMARY KEY,
	name TEXT NOT NULL,
	symbol TEXT NOT NULL,
	decimals INTEGER NOT NULL
)
`

const createStateBatchesTable = `
CREATE TABLE IF NOT EXISTS state_batches (
	index INTEGER NOT NULL PRIMARY KEY,
	root VARCHAR NOT NULL,
	size INTEGER NOT NULL,
	prev_total INTEGER NOT NULL,
	extra_data BYTEA NOT NULL,
	block_hash VARCHAR NOT NULL REFERENCES l1_blocks(hash)
)
`

const createWithdrawalsTable = `
CREATE TABLE IF NOT EXISTS withdrawals (
	guid VARCHAR PRIMARY KEY NOT NULL,
	from_address VARCHAR NOT NULL,
	to_address VARCHAR NOT NULL,
	l1_token VARCHAR NOT NULL,
	l2_token VARCHAR NOT NULL REFERENCES l2_tokens(address),
	amount VARCHAR NOT NULL,
	data BYTEA NOT NULL,
	log_index INTEGER NOT NULL,
	block_hash VARCHAR NOT NULL REFERENCES l2_blocks(hash),
	tx_hash VARCHAR NOT NULL,
	state_batch INTEGER REFERENCES state_batches(index)
)
`

const insertETHL1Token = `
INSERT INTO l1_tokens
	(address, name, symbol, decimals)
VALUES ('0x0000000000000000000000000000000000000000', 'Ethereum', 'ETH', 18)
ON CONFLICT (address) DO NOTHING;
`

// earlier transactions used 0x0000000000000000000000000000000000000000 as
// address of ETH so insert both that and
// 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000
const insertETHL2Token = `
INSERT INTO l2_tokens
	(address, name, symbol, decimals)
VALUES ('0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000', 'Ethereum', 'ETH', 18)
ON CONFLICT (address) DO NOTHING;
INSERT INTO l2_tokens
	(address, name, symbol, decimals)
VALUES ('0x0000000000000000000000000000000000000000', 'Ethereum', 'ETH', 18)
ON CONFLICT (address) DO NOTHING;
`

const createL1L2NumberIndex = `
CREATE UNIQUE INDEX IF NOT EXISTS l1_blocks_number ON l1_blocks(number);
CREATE UNIQUE INDEX IF NOT EXISTS l2_blocks_number ON l2_blocks(number);
`

type PaginationParam struct {
	Limit  uint64
	Offset uint64
}

var schema = []string{
	createL1BlocksTable,
	createL2BlocksTable,
	createL1TokensTable,
	createL2TokensTable,
	createStateBatchesTable,
	insertETHL1Token,
	insertETHL2Token,
	createDepositsTable,
	createWithdrawalsTable,
	createL1L2NumberIndex,
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
	GUID        string
	TxHash      common.Hash
	L1Token     common.Address
	L2Token     common.Address
	FromAddress common.Address
	ToAddress   common.Address
	Amount      *big.Int
	Data        []byte
	LogIndex    uint
}

func (d Deposit) String() string {
	return d.TxHash.String()
}

type Withdrawal struct {
	GUID        string
	TxHash      l2common.Hash
	L1Token     l2common.Address
	L2Token     l2common.Address
	FromAddress l2common.Address
	ToAddress   l2common.Address
	Amount      *big.Int
	Data        []byte
	LogIndex    uint
}

func (w Withdrawal) String() string {
	return w.TxHash.String()
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

type StateBatch struct {
	Index     *big.Int
	Root      common.Hash
	Size      *big.Int
	PrevTotal *big.Int
	ExtraData []byte
	BlockHash common.Hash
}

type StateBatchJSON struct {
	Index          uint64 `json:"index"`
	Root           string `json:"root"`
	Size           uint64 `json:"size"`
	PrevTotal      uint64 `json:"prevTotal"`
	ExtraData      []byte `json:"extraData"`
	BlockHash      string `json:"blockHash"`
	BlockNumber    uint64 `json:"blockNumber"`
	BlockTimestamp uint64 `json:"blockTimestamp"`
}

type Token struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

var ETHL1Token = &Token{
	Address:  "0x0000000000000000000000000000000000000000",
	Name:     "Ethereum",
	Symbol:   "ETH",
	Decimals: 18,
}

var ETHL2Address = l2common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")

var ETHL2Token = &Token{
	Address:  "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
	Name:     "Ethereum",
	Symbol:   "ETH",
	Decimals: 18,
}

type DepositJSON struct {
	GUID           string `json:"guid"`
	FromAddress    string `json:"from"`
	ToAddress      string `json:"to"`
	L1Token        *Token `json:"l1Token"`
	L2Token        string `json:"l2Token"`
	Amount         string `json:"amount"`
	Data           []byte `json:"data"`
	LogIndex       uint64 `json:"logIndex"`
	BlockNumber    uint64 `json:"blockNumber"`
	BlockTimestamp string `json:"blockTimestamp"`
	TxHash         string `json:"transactionHash"`
}

type WithdrawalJSON struct {
	GUID           string          `json:"guid"`
	FromAddress    string          `json:"from"`
	ToAddress      string          `json:"to"`
	L1Token        string          `json:"l1Token"`
	L2Token        *Token          `json:"l2Token"`
	Amount         string          `json:"amount"`
	Data           []byte          `json:"data"`
	LogIndex       uint64          `json:"logIndex"`
	BlockNumber    uint64          `json:"blockNumber"`
	BlockTimestamp string          `json:"blockTimestamp"`
	TxHash         string          `json:"transactionHash"`
	Batch          *StateBatchJSON `json:"batch"`
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

func (d *Database) GetL2TokenByAddress(address string) (*Token, error) {
	const selectL2TokenStatement = `
	SELECT name, symbol, decimals FROM l2_tokens WHERE address = $1;
	`

	var token *Token
	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectL2TokenStatement)
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

func (d *Database) AddL2Token(address string, token *Token) error {
	const insertTokenStatement = `
	INSERT INTO l2_tokens
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
		(guid, from_address, to_address, l1_token, l2_token, amount, tx_hash, log_index, block_hash, data)
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
				NewGUID(),
				deposit.FromAddress.String(),
				deposit.ToAddress.String(),
				deposit.L1Token.String(),
				deposit.L2Token.String(),
				deposit.Amount.String(),
				deposit.TxHash.String(),
				deposit.LogIndex,
				block.Hash.String(),
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
		(guid, from_address, to_address, l1_token, l2_token, amount, tx_hash, log_index, block_hash, data)
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
				NewGUID(),
				withdrawal.FromAddress.String(),
				withdrawal.ToAddress.String(),
				withdrawal.L1Token.String(),
				withdrawal.L2Token.String(),
				withdrawal.Amount.String(),
				withdrawal.TxHash.String(),
				withdrawal.LogIndex,
				block.Hash.String(),
				withdrawal.Data,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Database) AddStateBatch(batches []StateBatch) error {
	const insertStateBatchStatement = `
	INSERT INTO state_batches
		(index, root, size, prev_total, extra_data, block_hash)
	VALUES
		($1, $2, $3, $4, $5, $6)
	`

	return txn(d.db, func(tx *sql.Tx) error {
		stateBatchStmt, err := tx.Prepare(insertStateBatchStatement)
		if err != nil {
			return err
		}

		for _, sb := range batches {
			_, err = stateBatchStmt.Exec(
				sb.Index.Uint64(),
				sb.Root.String(),
				sb.Size.Uint64(),
				sb.PrevTotal.Uint64(),
				sb.ExtraData,
				sb.BlockHash.String(),
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Database) GetDepositsByAddress(address common.Address, page PaginationParam) ([]DepositJSON, error) {
	const selectDepositsStatement = `
	SELECT
		deposits.guid, deposits.from_address, deposits.to_address,
		deposits.amount, deposits.tx_hash, deposits.data,
		deposits.l1_token, deposits.l2_token,
		l1_tokens.name, l1_tokens.symbol, l1_tokens.decimals,
		l1_blocks.number, l1_blocks.timestamp
	FROM deposits
		INNER JOIN l1_blocks ON deposits.block_hash=l1_blocks.hash
		INNER JOIN l1_tokens ON deposits.l1_token=l1_tokens.address
	WHERE deposits.from_address = $1 ORDER BY l1_blocks.timestamp LIMIT $2 OFFSET $3;
	`
	var deposits []DepositJSON

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
			var deposit DepositJSON
			var l1_token Token
			if err := rows.Scan(
				&deposit.GUID, &deposit.FromAddress, &deposit.ToAddress,
				&deposit.Amount, &deposit.TxHash, &deposit.Data,
				&l1_token.Address, &deposit.L2Token,
				&l1_token.Name, &l1_token.Symbol, &l1_token.Decimals,
				&deposit.BlockNumber, &deposit.BlockTimestamp,
			); err != nil {
				return err
			}
			deposit.L1Token = &l1_token
			deposits = append(deposits, deposit)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return deposits, nil
}

func (d *Database) GetWithdrawalBatch(hash l2common.Hash) (*StateBatchJSON, error) {
	const selectWithdrawalBatchStatement = `
	SELECT
		state_batches.index, state_batches.root, state_batches.size, state_batches.prev_total, state_batches.extra_data, state_batches.block_hash,
		l1_blocks.number, l1_blocks.timestamp
	FROM state_batches
	INNER JOIN l1_blocks ON state_batches.block_hash = l1_blocks.hash
	WHERE size + prev_total >= (
		SELECT
			number
		FROM
		withdrawals
		INNER JOIN l2_blocks ON withdrawals.block_hash = l2_blocks.hash where tx_hash=$1
	) ORDER BY INDEX LIMIT 1;
	`

	var batch *StateBatchJSON
	err := txn(d.db, func(tx *sql.Tx) error {
		queryStmt, err := tx.Prepare(selectWithdrawalBatchStatement)
		if err != nil {
			return err
		}

		row := queryStmt.QueryRow(hash.String())
		if row.Err() != nil {
			return row.Err()
		}

		var index, size, prev_total, block_number, block_timestamp uint64
		var root, block_hash string
		var extra_data []byte
		err = row.Scan(&index, &root, &size, &prev_total, &extra_data, &block_hash,
			&block_number, &block_timestamp)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				batch = nil
				return nil
			}
			return err
		}

		batch = &StateBatchJSON{
			Index:          index,
			Root:           root,
			Size:           size,
			PrevTotal:      prev_total,
			ExtraData:      extra_data,
			BlockHash:      block_hash,
			BlockNumber:    block_number,
			BlockTimestamp: block_timestamp,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return batch, nil
}

func (d *Database) GetWithdrawalsByAddress(address l2common.Address, page PaginationParam) ([]WithdrawalJSON, error) {
	const selectWithdrawalsStatement = `
	SELECT
	    withdrawals.guid, withdrawals.from_address, withdrawals.to_address,
		withdrawals.amount, withdrawals.tx_hash, withdrawals.data,
		withdrawals.l1_token, withdrawals.l2_token,
		l2_tokens.name, l2_tokens.symbol, l2_tokens.decimals,
		l2_blocks.number, l2_blocks.timestamp
	FROM withdrawals
		INNER JOIN l2_blocks ON withdrawals.block_hash=l2_blocks.hash
		INNER JOIN l2_tokens ON withdrawals.l2_token=l2_tokens.address
	WHERE withdrawals.from_address = $1 ORDER BY l2_blocks.timestamp LIMIT $2 OFFSET $3;
	`
	var withdrawals []WithdrawalJSON

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
			var withdrawal WithdrawalJSON
			var l2_token Token
			if err := rows.Scan(
				&withdrawal.GUID, &withdrawal.FromAddress, &withdrawal.ToAddress,
				&withdrawal.Amount, &withdrawal.TxHash, &withdrawal.Data,
				&withdrawal.L1Token, &l2_token.Address,
				&l2_token.Name, &l2_token.Symbol, &l2_token.Decimals,
				&withdrawal.BlockNumber, &withdrawal.BlockTimestamp,
			); err != nil {
				return err
			}
			withdrawal.L2Token = &l2_token
			withdrawals = append(withdrawals, withdrawal)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	for i := range withdrawals {
		batch, _ := d.GetWithdrawalBatch(l2common.HexToHash(withdrawals[i].TxHash))
		withdrawals[i].Batch = batch
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

		row := queryStmt.QueryRow()
		if row.Err() != nil {
			return row.Err()
		}

		var number uint64
		var hash string
		err = row.Scan(&number, &hash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				highestBlock = nil
				return nil
			}
			return err
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

		row := queryStmt.QueryRow()
		if row.Err() != nil {
			return row.Err()
		}

		var number uint64
		var hash string
		err = row.Scan(&number, &hash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				highestBlock = nil
				return nil
			}
			return err
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

		row := queryStmt.QueryRow(hash.String())
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil
		}
		if row.Err() != nil {
			return err
		}

		var hash string
		var parentHash string
		var number uint64
		var timestamp uint64
		err = row.Scan(&hash, &parentHash, &number, &timestamp)
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

func NewGUID() string {
	return uuid.New().String()
}
