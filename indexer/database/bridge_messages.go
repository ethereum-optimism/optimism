package database

import (
	"errors"
	"fmt"
	"math/big"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common"

	"github.com/google/uuid"
)

/**
 * Types
 */

type BridgeMessage struct {
	Nonce       U256        `gorm:"primaryKey"`
	MessageHash common.Hash `gorm:"serializer:json"`

	SentMessageEventGUID    uuid.UUID
	RelayedMessageEventGUID *uuid.UUID

	Tx       Transaction `gorm:"embedded"`
	GasLimit U256
}

type L1BridgeMessage struct {
	BridgeMessage         `gorm:"embedded"`
	TransactionSourceHash common.Hash `gorm:"serializer:json"`
}

type L2BridgeMessage struct {
	BridgeMessage             `gorm:"embedded"`
	TransactionWithdrawalHash common.Hash `gorm:"serializer:json"`
}

type BridgeMessagesView interface {
	L1BridgeMessage(*big.Int) (*L1BridgeMessage, error)
	L1BridgeMessageByHash(common.Hash) (*L1BridgeMessage, error)
	LatestL1BridgeMessageNonce() (*big.Int, error)

	L2BridgeMessage(*big.Int) (*L2BridgeMessage, error)
	L2BridgeMessageByHash(common.Hash) (*L2BridgeMessage, error)
	LatestL2BridgeMessageNonce() (*big.Int, error)
}

type BridgeMessagesDB interface {
	BridgeMessagesView

	StoreL1BridgeMessages([]*L1BridgeMessage) error
	MarkRelayedL1BridgeMessage(common.Hash, uuid.UUID) error

	StoreL2BridgeMessages([]*L2BridgeMessage) error
	MarkRelayedL2BridgeMessage(common.Hash, uuid.UUID) error
}

/**
 * Implementation
 */

type bridgeMessagesDB struct {
	gorm *gorm.DB
}

func newBridgeMessagesDB(db *gorm.DB) BridgeMessagesDB {
	return &bridgeMessagesDB{gorm: db}
}

/**
 * Arbitrary Messages Sent from L1
 */

func (db bridgeMessagesDB) StoreL1BridgeMessages(messages []*L1BridgeMessage) error {
	result := db.gorm.Create(&messages)
	return result.Error
}

func (db bridgeMessagesDB) L1BridgeMessage(nonce *big.Int) (*L1BridgeMessage, error) {
	var sentMessage L1BridgeMessage
	result := db.gorm.Where(&BridgeMessage{Nonce: U256{Int: nonce}}).Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &sentMessage, nil
}

func (db bridgeMessagesDB) L1BridgeMessageByHash(messageHash common.Hash) (*L1BridgeMessage, error) {
	var sentMessage L1BridgeMessage
	result := db.gorm.Where(&BridgeMessage{MessageHash: messageHash}).Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &sentMessage, nil
}

func (db bridgeMessagesDB) LatestL1BridgeMessageNonce() (*big.Int, error) {
	var sentMessage L1BridgeMessage
	result := db.gorm.Order("nonce DESC").Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return sentMessage.Nonce.Int, nil
}

/**
 * Arbitrary Messages Sent from L2
 */

func (db bridgeMessagesDB) MarkRelayedL1BridgeMessage(messageHash common.Hash, relayEvent uuid.UUID) error {
	message, err := db.L1BridgeMessageByHash(messageHash)
	if err != nil {
		return err
	} else if message == nil {
		return fmt.Errorf("L1BridgeMessage with message hash %s not found", messageHash)
	}

	message.RelayedMessageEventGUID = &relayEvent
	result := db.gorm.Save(message)
	return result.Error
}

func (db bridgeMessagesDB) StoreL2BridgeMessages(messages []*L2BridgeMessage) error {
	result := db.gorm.Create(&messages)
	return result.Error
}

func (db bridgeMessagesDB) L2BridgeMessage(nonce *big.Int) (*L2BridgeMessage, error) {
	var sentMessage L2BridgeMessage
	result := db.gorm.Where(&BridgeMessage{Nonce: U256{Int: nonce}}).Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &sentMessage, nil
}

func (db bridgeMessagesDB) L2BridgeMessageByHash(messageHash common.Hash) (*L2BridgeMessage, error) {
	var sentMessage L2BridgeMessage
	result := db.gorm.Where(&BridgeMessage{MessageHash: messageHash}).Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &sentMessage, nil
}

func (db bridgeMessagesDB) LatestL2BridgeMessageNonce() (*big.Int, error) {
	var sentMessage L2BridgeMessage
	result := db.gorm.Order("nonce DESC").Take(&sentMessage)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return sentMessage.Nonce.Int, nil
}

func (db bridgeMessagesDB) MarkRelayedL2BridgeMessage(messageHash common.Hash, relayEvent uuid.UUID) error {
	message, err := db.L2BridgeMessageByHash(messageHash)
	if err != nil {
		return err
	} else if message == nil {
		return fmt.Errorf("L2BridgeMessage with message hash %s not found", messageHash)
	}

	message.RelayedMessageEventGUID = &relayEvent
	result := db.gorm.Save(message)
	return result.Error
}
