// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package abi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// The ABI holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABI struct {
	Constructor Method
	Methods     map[string]Method
	Events      map[string]Event
	MethodsById map[[4]byte]*Method
}

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := dec.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	// Fetch the ABI of the requested method
	if name == "" {
		// constructor
		arguments, err := abi.Constructor.Inputs.Pack(args...)
		if err != nil {
			return nil, err
		}
		return arguments, nil
	}
	method, exist := abi.Methods[name]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", name)
	}
	arguments, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	// Pack up the method ID too if not a constructor and return
	return append(method.ID(), arguments...), nil
}

// Unpack output in v according to the abi specification
func (abi ABI) Unpack(v interface{}, name string, data []byte) (err error) {
	// since there can't be naming collisions with contracts and events,
	// we need to decide whether we're calling a method or an event
	if method, ok := abi.Methods[name]; ok {
		if len(data)%32 != 0 {
			return fmt.Errorf("abi: improperly formatted output: %s - Bytes: [%+v]", string(data), data)
		}
		return method.Outputs.Unpack(v, data)
	}
	if event, ok := abi.Events[name]; ok {
		return event.Inputs.Unpack(v, data)
	}
	return fmt.Errorf("abi: could not locate named method or event")
}

// UnpackIntoMap unpacks a log into the provided map[string]interface{}
func (abi ABI) UnpackIntoMap(v map[string]interface{}, name string, data []byte) (err error) {
	// since there can't be naming collisions with contracts and events,
	// we need to decide whether we're calling a method or an event
	if method, ok := abi.Methods[name]; ok {
		if len(data)%32 != 0 {
			return fmt.Errorf("abi: improperly formatted output")
		}
		return method.Outputs.UnpackIntoMap(v, data)
	}
	if event, ok := abi.Events[name]; ok {
		return event.Inputs.UnpackIntoMap(v, data)
	}
	return fmt.Errorf("abi: could not locate named method or event")
}

// UnmarshalJSON implements json.Unmarshaler interface
func (abi *ABI) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type            string
		Name            string
		Constant        bool
		StateMutability string
		Anonymous       bool
		Inputs          []Argument
		Outputs         []Argument
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	abi.Methods = make(map[string]Method)
	// UsingOVM specific changes
	// Create a cache of methods when running under the OVM because
	// looking up methods based on the 4 byte selector is part of the hot
	// code path. Without this cache, it was observed that 50% of the CPU
	// time during syncing was spent hashing the function selectors
	// during the call to `abi.MethodsById`
	abi.MethodsById = make(map[[4]byte]*Method)
	abi.Events = make(map[string]Event)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = Method{
				Inputs: field.Inputs,
			}
		// empty defaults to function according to the abi spec
		case "function", "":
			name := field.Name
			_, ok := abi.Methods[name]
			for idx := 0; ok; idx++ {
				name = fmt.Sprintf("%s%d", field.Name, idx)
				_, ok = abi.Methods[name]
			}
			isConst := field.Constant || field.StateMutability == "pure" || field.StateMutability == "view"
			method := Method{
				Name:    name,
				RawName: field.Name,
				Const:   isConst,
				Inputs:  field.Inputs,
				Outputs: field.Outputs,
			}
			abi.Methods[name] = method
			// UsingOVM specific changes
			// Add method to the id cache
			sigdata := method.ID()
			abi.MethodsById[[4]byte{sigdata[0], sigdata[1], sigdata[2], sigdata[3]}] = &method
		case "event":
			name := field.Name
			_, ok := abi.Events[name]
			for idx := 0; ok; idx++ {
				name = fmt.Sprintf("%s%d", field.Name, idx)
				_, ok = abi.Events[name]
			}
			abi.Events[name] = Event{
				Name:      name,
				RawName:   field.Name,
				Anonymous: field.Anonymous,
				Inputs:    field.Inputs,
			}
		}
	}

	return nil
}

// MethodById looks up a method by the 4-byte id
// returns nil if none found
func (abi *ABI) MethodById(sigdata []byte) (*Method, error) {
	if len(sigdata) < 4 {
		return nil, fmt.Errorf("data too short (%d bytes) for abi method lookup", len(sigdata))
	}

	// UsingOVM specific changes
	// Use the method cache to prevent the need to iterate and hash
	// each method in the ABI
	method, exist := abi.MethodsById[[4]byte{sigdata[0], sigdata[1], sigdata[2], sigdata[3]}]
	if !exist {
		return nil, fmt.Errorf("no method with id: %#x", sigdata[:4])
	}
	return method, nil
}

// EventByID looks an event up by its topic hash in the
// ABI and returns nil if none found.
func (abi *ABI) EventByID(topic common.Hash) (*Event, error) {
	for _, event := range abi.Events {
		if bytes.Equal(event.ID().Bytes(), topic.Bytes()) {
			return &event, nil
		}
	}
	return nil, fmt.Errorf("no event with id: %#x", topic.Hex())
}

// UsingOVM
// Both RevertSelector and UnpackRevert were pulled from upstream
// geth as they were not present in the version of geth that this
// codebase was forked from. These are useful for displaying revert
// messages to users when they use `eth_call`
// RevertSelector is a special function selector for revert reason unpacking.
var RevertSelector = crypto.Keccak256([]byte("Error(string)"))[:4]

// UnpackRevert resolves the abi-encoded revert reason. According to the solidity
// docs https://docs.soliditylang.org/en/v0.8.4/control-structures.html#revert,
// the provided revert reason is abi-encoded as if it were a call to a function
// `Error(string)`. So it's a special tool for it.
func UnpackRevert(data []byte) (string, error) {
	if len(data) < 4 {
		return "", errors.New("invalid data for unpacking")
	}
	if !bytes.Equal(data[:4], RevertSelector) {
		return "", errors.New("invalid data for unpacking")
	}
	typ, _ := NewType("string", "", nil)
	unpacked, err := (Arguments{{Type: typ}}).UnpackValues(data[4:])
	if err != nil {
		return "", err
	}
	return unpacked[0].(string), nil
}
