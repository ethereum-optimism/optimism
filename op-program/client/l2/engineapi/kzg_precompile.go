// This file contains code of the upstream go-ethereum kzgPointEvaluation implementation.
// Modifications have been made, primarily to substitute kzg4844.VerifyProof with a preimage oracle call.
//
// Original copyright disclaimer, applicable only to this file:
// -------------------------------------------------------------------
// Copyright 2014 The go-ethereum Authors
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

package engineapi

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// OracleKZGPointEvaluation implements the EIP-4844 point evaluation precompile,
// using the preimage-oracle to perform the evaluation.
type OracleKZGPointEvaluation struct {
	Oracle KZGPointEvaluationOracle
}

// KZGPointEvaluationOracle defines the high-level API used to retrieve the result of the KZG point evaluation precompile
type KZGPointEvaluationOracle interface {
	KZGPointEvaluation(input []byte) bool
}

// RequiredGas estimates the gas required for running the point evaluation precompile.
func (b *OracleKZGPointEvaluation) RequiredGas(input []byte) uint64 {
	return params.BlobTxPointEvaluationPrecompileGas
}

const (
	blobVerifyInputLength     = 192 // Max input length for the point evaluation precompile.
	blobPrecompileReturnValue = "000000000000000000000000000000000000000000000000000000000000100073eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"
)

var (
	errBlobVerifyInvalidInputLength = errors.New("invalid input length")
	errBlobVerifyMismatchedVersion  = errors.New("mismatched versioned hash")
	errBlobVerifyKZGProof           = errors.New("error verifying kzg proof")
)

// Run executes the point evaluation precompile.
func (b *OracleKZGPointEvaluation) Run(input []byte) ([]byte, error) {
	// Modification note: the L1 precompile behavior may change, but not in incompatible ways.
	// We want to enforce the subset that represents the EVM behavior activated in L2.
	// Below is a copy of the Cancun behavior. L1 might expand on that at a later point.

	if len(input) != blobVerifyInputLength {
		return nil, errBlobVerifyInvalidInputLength
	}
	// versioned hash: first 32 bytes
	var versionedHash common.Hash
	copy(versionedHash[:], input[:])

	var (
		point kzg4844.Point
		claim kzg4844.Claim
	)
	// Evaluation point: next 32 bytes
	copy(point[:], input[32:])
	// Expected output: next 32 bytes
	copy(claim[:], input[64:])

	// input kzg point: next 48 bytes
	var commitment kzg4844.Commitment
	copy(commitment[:], input[96:])
	if eth.KZGToVersionedHash(commitment) != versionedHash {
		return nil, errBlobVerifyMismatchedVersion
	}

	// Proof: next 48 bytes
	var proof kzg4844.Proof
	copy(proof[:], input[144:])

	// Modification note: below replaces the kzg4844.VerifyProof call
	ok := b.Oracle.KZGPointEvaluation(input)
	if !ok {
		return nil, fmt.Errorf("%w: invalid KZG point evaluation", errBlobVerifyKZGProof)
	}
	return common.FromHex(blobPrecompileReturnValue), nil
}
