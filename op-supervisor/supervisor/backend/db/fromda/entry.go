package fromda

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
)

const searchCheckpointFrequency = 256

type EntryObj interface {
	encode() Entry
}

type Entry = entrydb.Entry[EntryType]

type EntryTypeFlag uint8

const (
	FlagSearchCheckpoint EntryTypeFlag = 1 << TypeSearchCheckpoint
	FlagCanonicalHash    EntryTypeFlag = 1 << TypeCanonicalHash
	FlagDerivedLink      EntryTypeFlag = 1 << TypeDerivedLink
	FlagDerivedCheck     EntryTypeFlag = 1 << TypeDerivedCheck
	FlagPadding          EntryTypeFlag = 1 << TypePadding
)

func (x EntryTypeFlag) String() string {
	var out []string
	for i := EntryTypeFlag(1); i != 0; i <<= 1 { // iterate to bitmask
		if x.Any(i) {
			out = append(out, i.String())
		}
	}
	return strings.Join(out, "|")
}

func (x EntryTypeFlag) Any(v EntryTypeFlag) bool {
	return x&v != 0
}

func (x *EntryTypeFlag) Add(v EntryTypeFlag) {
	*x = *x | v
}

func (x *EntryTypeFlag) Remove(v EntryTypeFlag) {
	*x = *x &^ v
}

type EntryType uint8

const (
	TypeSearchCheckpoint EntryType = iota
	TypeCanonicalHash
	TypeDerivedLink
	TypeDerivedCheck
	TypePadding
)

func (x EntryType) String() string {
	switch x {
	case TypeSearchCheckpoint:
		return "searchCheckpoint"
	case TypeCanonicalHash:
		return "canonicalHash"
	case TypeDerivedLink:
		return "derivedLink"
	case TypeDerivedCheck:
		return "derivedCheck"
	case TypePadding:
		return "padding"
	default:
		return fmt.Sprintf("unknown-%d", uint8(x))
	}
}
