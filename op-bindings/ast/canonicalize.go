package ast

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

var remapTypeRe = regexp.MustCompile(`^(t_[\w_]+\([\w]+\))([\d]+)(_[\w]+)?$`)

// CanonicalizeASTIDs canonicalizes AST IDs in storage layouts so that they
// don't cause unnecessary conflicts/diffs. The implementation is not
// particularly efficient, but is plenty fast enough for our purposes.
// It works in two passes:
//
//  1. First, it finds all AST IDs in storage and types, and builds a
//     map to replace them in the second pass.
//  2. The second pass performs the replacement.
//
// This function returns a copy of the passed-in storage layout. The
// inefficiency comes from replaceType, which performs a linear
// search of all replacements when performing substring matches of
// composite types.
func CanonicalizeASTIDs(in *solc.StorageLayout) *solc.StorageLayout {
	lastId := uint(1000)
	astIDRemappings := make(map[uint]uint)
	typeRemappings := make(map[string]string)

	for _, slot := range in.Storage {
		astIDRemappings[slot.AstId] = lastId
		lastId++
	}

	// Go map iteration order is random, so we need to sort
	// keys here in order to prevent non-determinism.
	var sortedOldTypes sort.StringSlice
	for oldType := range in.Types {
		sortedOldTypes = append(sortedOldTypes, oldType)
	}
	sortedOldTypes.Sort()

	for _, oldType := range sortedOldTypes {
		matches := remapTypeRe.FindAllStringSubmatch(oldType, -1)
		if len(matches) == 0 {
			continue
		}

		replaceAstID := matches[0][2]
		newType := strings.Replace(oldType, replaceAstID, strconv.Itoa(int(lastId)), 1)
		typeRemappings[oldType] = newType
		lastId++
	}

	outLayout := &solc.StorageLayout{
		Types: make(map[string]solc.StorageLayoutType),
	}
	for _, slot := range in.Storage {
		outLayout.Storage = append(outLayout.Storage, solc.StorageLayoutEntry{
			AstId:    astIDRemappings[slot.AstId],
			Contract: slot.Contract,
			Label:    slot.Label,
			Offset:   slot.Offset,
			Slot:     slot.Slot,
			Type:     replaceType(typeRemappings, slot.Type),
		})
	}

	for _, oldType := range sortedOldTypes {
		value := in.Types[oldType]
		newType := replaceType(typeRemappings, oldType)
		outLayout.Types[newType] = solc.StorageLayoutType{
			Encoding:      value.Encoding,
			Label:         value.Label,
			NumberOfBytes: value.NumberOfBytes,
			Key:           replaceType(typeRemappings, value.Key),
			Value:         replaceType(typeRemappings, value.Value),
		}
	}
	return outLayout
}

func replaceType(typeRemappings map[string]string, in string) string {
	if typeRemappings[in] != "" {
		return typeRemappings[in]
	}

	for oldType, newType := range typeRemappings {
		if strings.Contains(in, oldType) {
			return strings.Replace(in, oldType, newType, 1)
		}
	}

	return in
}
