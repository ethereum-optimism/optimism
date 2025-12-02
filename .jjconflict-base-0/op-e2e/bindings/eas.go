// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// Attestation is an auto generated low-level Go binding around an user-defined struct.
type Attestation struct {
	Uid            [32]byte
	Schema         [32]byte
	Time           uint64
	ExpirationTime uint64
	RevocationTime uint64
	RefUID         [32]byte
	Recipient      common.Address
	Attester       common.Address
	Revocable      bool
	Data           []byte
}

// AttestationRequest is an auto generated low-level Go binding around an user-defined struct.
type AttestationRequest struct {
	Schema [32]byte
	Data   AttestationRequestData
}

// AttestationRequestData is an auto generated low-level Go binding around an user-defined struct.
type AttestationRequestData struct {
	Recipient      common.Address
	ExpirationTime uint64
	Revocable      bool
	RefUID         [32]byte
	Data           []byte
	Value          *big.Int
}

// DelegatedAttestationRequest is an auto generated low-level Go binding around an user-defined struct.
type DelegatedAttestationRequest struct {
	Schema    [32]byte
	Data      AttestationRequestData
	Signature Signature
	Attester  common.Address
	Deadline  uint64
}

// DelegatedRevocationRequest is an auto generated low-level Go binding around an user-defined struct.
type DelegatedRevocationRequest struct {
	Schema    [32]byte
	Data      RevocationRequestData
	Signature Signature
	Revoker   common.Address
	Deadline  uint64
}

// MultiAttestationRequest is an auto generated low-level Go binding around an user-defined struct.
type MultiAttestationRequest struct {
	Schema [32]byte
	Data   []AttestationRequestData
}

// MultiDelegatedAttestationRequest is an auto generated low-level Go binding around an user-defined struct.
type MultiDelegatedAttestationRequest struct {
	Schema     [32]byte
	Data       []AttestationRequestData
	Signatures []Signature
	Attester   common.Address
	Deadline   uint64
}

// MultiDelegatedRevocationRequest is an auto generated low-level Go binding around an user-defined struct.
type MultiDelegatedRevocationRequest struct {
	Schema     [32]byte
	Data       []RevocationRequestData
	Signatures []Signature
	Revoker    common.Address
	Deadline   uint64
}

// MultiRevocationRequest is an auto generated low-level Go binding around an user-defined struct.
type MultiRevocationRequest struct {
	Schema [32]byte
	Data   []RevocationRequestData
}

// RevocationRequest is an auto generated low-level Go binding around an user-defined struct.
type RevocationRequest struct {
	Schema [32]byte
	Data   RevocationRequestData
}

// RevocationRequestData is an auto generated low-level Go binding around an user-defined struct.
type RevocationRequestData struct {
	Uid   [32]byte
	Value *big.Int
}

// Signature is an auto generated low-level Go binding around an user-defined struct.
type Signature struct {
	V uint8
	R [32]byte
	S [32]byte
}

// EASMetaData contains all meta data concerning the EAS contract.
var EASMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"attest\",\"inputs\":[{\"name\":\"request\",\"type\":\"tuple\",\"internalType\":\"structAttestationRequest\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple\",\"internalType\":\"structAttestationRequestData\",\"components\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"expirationTime\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"refUID\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"attestByDelegation\",\"inputs\":[{\"name\":\"delegatedRequest\",\"type\":\"tuple\",\"internalType\":\"structDelegatedAttestationRequest\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple\",\"internalType\":\"structAttestationRequestData\",\"components\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"expirationTime\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"refUID\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"signature\",\"type\":\"tuple\",\"internalType\":\"structSignature\",\"components\":[{\"name\":\"v\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"attester\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"deadline\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"getAttestTypeHash\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getAttestation\",\"inputs\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structAttestation\",\"components\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"time\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"expirationTime\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"revocationTime\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"refUID\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"attester\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getDomainSeparator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getName\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNonce\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRevokeOffchain\",\"inputs\":[{\"name\":\"revoker\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRevokeTypeHash\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getSchemaRegistry\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractISchemaRegistry\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getTimestamp\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"increaseNonce\",\"inputs\":[{\"name\":\"newNonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isAttestationValid\",\"inputs\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"multiAttest\",\"inputs\":[{\"name\":\"multiRequests\",\"type\":\"tuple[]\",\"internalType\":\"structMultiAttestationRequest[]\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple[]\",\"internalType\":\"structAttestationRequestData[]\",\"components\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"expirationTime\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"refUID\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"multiAttestByDelegation\",\"inputs\":[{\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\",\"internalType\":\"structMultiDelegatedAttestationRequest[]\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple[]\",\"internalType\":\"structAttestationRequestData[]\",\"components\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"expirationTime\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"refUID\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"signatures\",\"type\":\"tuple[]\",\"internalType\":\"structSignature[]\",\"components\":[{\"name\":\"v\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"attester\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"deadline\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"multiRevoke\",\"inputs\":[{\"name\":\"multiRequests\",\"type\":\"tuple[]\",\"internalType\":\"structMultiRevocationRequest[]\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple[]\",\"internalType\":\"structRevocationRequestData[]\",\"components\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"multiRevokeByDelegation\",\"inputs\":[{\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\",\"internalType\":\"structMultiDelegatedRevocationRequest[]\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple[]\",\"internalType\":\"structRevocationRequestData[]\",\"components\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"signatures\",\"type\":\"tuple[]\",\"internalType\":\"structSignature[]\",\"components\":[{\"name\":\"v\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"revoker\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"deadline\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"multiRevokeOffchain\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"multiTimestamp\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revoke\",\"inputs\":[{\"name\":\"request\",\"type\":\"tuple\",\"internalType\":\"structRevocationRequest\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple\",\"internalType\":\"structRevocationRequestData\",\"components\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"revokeByDelegation\",\"inputs\":[{\"name\":\"delegatedRequest\",\"type\":\"tuple\",\"internalType\":\"structDelegatedRevocationRequest\",\"components\":[{\"name\":\"schema\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"data\",\"type\":\"tuple\",\"internalType\":\"structRevocationRequestData\",\"components\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"signature\",\"type\":\"tuple\",\"internalType\":\"structSignature\",\"components\":[{\"name\":\"v\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"revoker\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"deadline\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"revokeOffchain\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"timestamp\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"Attested\",\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"attester\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"uid\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"schemaUID\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"NonceIncreased\",\"inputs\":[{\"name\":\"oldNonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newNonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Revoked\",\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"attester\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"uid\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"schemaUID\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RevokedOffchain\",\"inputs\":[{\"name\":\"revoker\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"timestamp\",\"type\":\"uint64\",\"indexed\":true,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Timestamped\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"timestamp\",\"type\":\"uint64\",\"indexed\":true,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AccessDenied\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AlreadyRevoked\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AlreadyRevokedOffchain\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"AlreadyTimestamped\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DeadlineExpired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InsufficientValue\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidAttestation\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidAttestations\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidExpirationTime\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidLength\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidNonce\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidOffset\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRegistry\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRevocation\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidRevocations\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidSchema\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidVerifier\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"Irrevocable\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotFound\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotPayable\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"WrongSchema\",\"inputs\":[]}]",
	Bin: "0x61016060405234801561001157600080fd5b50604080518082018252600381526245415360e81b60208083019182528351808501855260058152640312e332e360dc1b908201529151812060e08190527f6a08c3e203132c561752255a4d52ffae85bb9c5d33cb3291520dea1b843563896101008190524660a081815286517f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f818801819052818901959095526060810193909352608080840192909252308382018190528751808503909201825260c093840190975280519501949094209093529290915261012091909152516101405260805160a05160c05160e05161010051610120516101405161457e61014b600039600061073701526000612784015260006127d3015260006127ae01526000612707015260006127310152600061275b015261457e6000f3fe60806040526004361061018b5760003560e01c806395411525116100d6578063d45c44351161007f578063ed24911d11610059578063ed24911d146104fd578063f10b5cc814610512578063f17325e71461054157600080fd5b8063d45c443514610467578063e30bb5631461049e578063e71ff365146104dd57600080fd5b8063b469318d116100b0578063b469318d146103ba578063b83010d314610414578063cf190f341461044757600080fd5b80639541152514610367578063a3112a641461037a578063a6d4dbc7146103a757600080fd5b806344adc90e116101385780634d003070116101125780634d003070146102de57806354fd4d50146102fe57806379f7573a1461034757600080fd5b806344adc90e1461029857806346926267146102b85780634cb7e9e5146102cb57600080fd5b806317d7de7c1161016957806317d7de7c146102205780632d0335ab146102425780633c0427151461028557600080fd5b80630eabf6601461019057806312b11a17146101a557806313893f61146101e7575b600080fd5b6101a361019e3660046134c8565b610554565b005b3480156101b157600080fd5b507ffeb2925a02bae3dae48d424a0437a2b6ac939aa9230ddc55a1a76f065d9880765b6040519081526020015b60405180910390f35b3480156101f357600080fd5b506102076102023660046134c8565b6106eb565b60405167ffffffffffffffff90911681526020016101de565b34801561022c57600080fd5b50610235610730565b6040516101de9190613578565b34801561024e57600080fd5b506101d461025d3660046135bd565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b6101d46102933660046135da565b610760565b6102ab6102a63660046134c8565b610863565b6040516101de9190613615565b6101a36102c6366004613659565b6109e4565b6101a36102d93660046134c8565b610a68565b3480156102ea57600080fd5b506102076102f9366004613671565b610b4b565b34801561030a57600080fd5b506102356040518060400160405280600581526020017f312e342e3000000000000000000000000000000000000000000000000000000081525081565b34801561035357600080fd5b506101a3610362366004613671565b610b58565b6102ab6103753660046134c8565b610bef565b34801561038657600080fd5b5061039a610395366004613671565b610e62565b6040516101de9190613771565b6101a36103b5366004613784565b611025565b3480156103c657600080fd5b506102076103d5366004613797565b73ffffffffffffffffffffffffffffffffffffffff919091166000908152603460209081526040808320938352929052205467ffffffffffffffff1690565b34801561042057600080fd5b507fb5d556f07587ec0f08cf386545cc4362c702a001650c2058002615ee5c9d1e756101d4565b34801561045357600080fd5b50610207610462366004613671565b6110ca565b34801561047357600080fd5b50610207610482366004613671565b60009081526033602052604090205467ffffffffffffffff1690565b3480156104aa57600080fd5b506104cd6104b9366004613671565b600090815260326020526040902054151590565b60405190151581526020016101de565b3480156104e957600080fd5b506102076104f83660046134c8565b6110d8565b34801561050957600080fd5b506101d4611110565b34801561051e57600080fd5b5060405173420000000000000000000000000000000000002081526020016101de565b6101d461054f3660046137c3565b61111a565b348160005b818110156106e4577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82018114600086868481811061059a5761059a6137fe565b90506020028101906105ac919061382d565b6105b590613ac3565b60208101518051919250908015806105d257508260400151518114155b15610609576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b818110156106ad576106a56040518060a001604052808660000151815260200185848151811061063e5761063e6137fe565b6020026020010151815260200186604001518481518110610661576106616137fe565b60200260200101518152602001866060015173ffffffffffffffffffffffffffffffffffffffff168152602001866080015167ffffffffffffffff168152506111d8565b60010161060c565b506106c383600001518385606001518a886113e9565b6106cd9088613bed565b9650505050506106dd8160010190565b9050610559565b5050505050565b60004282825b818110156107245761071c3387878481811061070f5761070f6137fe565b9050602002013585611a18565b6001016106f1565b50909150505b92915050565b606061075b7f0000000000000000000000000000000000000000000000000000000000000000611b17565b905090565b600061077361076e83613d22565b611ca5565b604080516001808252818301909252600091816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90920191018161078a5790505090506107f86020840184613d9d565b61080190613dd1565b81600081518110610814576108146137fe565b602090810291909101015261083d83358261083560c0870160a088016135bd565b346001611e2f565b60200151600081518110610853576108536137fe565b6020026020010151915050919050565b60608160008167ffffffffffffffff8111156108815761088161386b565b6040519080825280602002602001820160405280156108b457816020015b606081526020019060019003908161089f5790505b509050600034815b848110156109ce577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff85018114368989848181106108fc576108fc6137fe565b905060200281019061090e9190613ddd565b905061091d6020820182613e11565b9050600003610958576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600061097d823561096c6020850185613e11565b61097591613e79565b338887611e2f565b805190915061098c9086613bed565b945080602001518785815181106109a5576109a56137fe565b6020026020010181905250806020015151860195505050506109c78160010190565b90506108bc565b506109d98383612541565b979650505050505050565b604080516001808252818301909252600091816020015b60408051808201909152600080825260208201528152602001906001900390816109fb579050509050610a3636839003830160208401613eed565b81600081518110610a4957610a496137fe565b6020908102919091010152610a63823582333460016113e9565b505050565b348160005b818110156106e4577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8201811436868684818110610aad57610aad6137fe565b9050602002810190610abf9190613ddd565b9050610b2c8135610ad36020840184613f09565b808060200260200160405190810160405280939291908181526020016000905b82821015610b1f57610b1060408302860136819003810190613eed565b81526020019060010190610af3565b50505050503388866113e9565b610b369086613bed565b94505050610b448160010190565b9050610a6d565b60004261072a838261262b565b33600090815260208190526040902054808211610ba1576040517f756688fe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b336000908152602081815260409182902084905581518381529081018490527f57b09af877df9068fd60a69d7b21f5576b8b38955812d6ae4ac52942f1e38fb7910160405180910390a15050565b60608160008167ffffffffffffffff811115610c0d57610c0d61386b565b604051908082528060200260200182016040528015610c4057816020015b6060815260200190600190039081610c2b5790505b509050600034815b848110156109ce577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8501811436898984818110610c8857610c886137fe565b9050602002810190610c9a919061382d565b9050366000610cac6020840184613e11565b909250905080801580610ccd5750610cc76040850185613f71565b90508114155b15610d04576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b81811015610de557610ddd6040518060a0016040528087600001358152602001868685818110610d3957610d396137fe565b9050602002810190610d4b9190613d9d565b610d5490613dd1565b8152602001610d666040890189613f71565b85818110610d7657610d766137fe565b905060600201803603810190610d8c9190613fd8565b8152602001610da16080890160608a016135bd565b73ffffffffffffffffffffffffffffffffffffffff168152602001610dcc60a0890160808a01613ff4565b67ffffffffffffffff169052611ca5565b600101610d07565b506000610e0e8535610df78587613e79565b610e076080890160608a016135bd565b8b8a611e2f565b8051909150610e1d9089613bed565b975080602001518a8881518110610e3657610e366137fe565b602002602001018190525080602001515189019850505050505050610e5b8160010190565b9050610c48565b604080516101408101825260008082526020820181905291810182905260608082018390526080820183905260a0820183905260c0820183905260e0820183905261010082019290925261012081019190915260008281526032602090815260409182902082516101408101845281548152600182015492810192909252600281015467ffffffffffffffff808216948401949094526801000000000000000081048416606084015270010000000000000000000000000000000090049092166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff16151561010082015260068201805491929161012084019190610f9c9061400f565b80601f0160208091040260200160405190810160405280929190818152602001828054610fc89061400f565b80156110155780601f10610fea57610100808354040283529160200191611015565b820191906000526020600020905b815481529060010190602001808311610ff857829003601f168201915b5050505050815250509050919050565b61103c6110373683900383018361405c565b6111d8565b604080516001808252818301909252600091816020015b604080518082019091526000808252602082015281526020019060019003908161105357905050905061108e36839003830160208401613eed565b816000815181106110a1576110a16137fe565b6020908102919091010152610a638235826110c260e0860160c087016135bd565b3460016113e9565b60004261072a338483611a18565b60004282825b81811015610724576111088686838181106110fb576110fb6137fe565b905060200201358461262b565b6001016110de565b600061075b6126ed565b604080516001808252818301909252600091829190816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816111345790505090506111a26020840184613d9d565b6111ab90613dd1565b816000815181106111be576111be6137fe565b602090810291909101015261083d83358233346001611e2f565b608081015167ffffffffffffffff161580159061120c57504267ffffffffffffffff16816080015167ffffffffffffffff16105b15611243576040517f1ab7da6b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6020808201516040808401516060850151855184518587015173ffffffffffffffffffffffffffffffffffffffff84166000908152978890529487208054969794969495611337957fb5d556f07587ec0f08cf386545cc4362c702a001650c2058002615ee5c9d1e7595949392886112ba836140ca565b909155506080808c015160408051602081019990995273ffffffffffffffffffffffffffffffffffffffff9097169688019690965260608701949094529285019190915260a084015260c083015267ffffffffffffffff1660e0820152610100015b60405160208183030381529060405280519060200120612821565b90506113ad84606001518284602001518560400151866000015160405160200161139993929190928352602083019190915260f81b7fff0000000000000000000000000000000000000000000000000000000000000016604082015260410190565b604051602081830303815290604052612834565b6113e3576040517f8baa579f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b50505050565b6040517fa2ea7c6e0000000000000000000000000000000000000000000000000000000081526004810186905260009081907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa158015611457573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016820160405261149d9190810190614102565b80519091506114d8576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b855160008167ffffffffffffffff8111156114f5576114f561386b565b60405190808252806020026020018201604052801561159457816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816115135790505b50905060008267ffffffffffffffff8111156115b2576115b261386b565b6040519080825280602002602001820160405280156115db578160200160208202803683370190505b50905060005b838110156119fa5760008a82815181106115fd576115fd6137fe565b6020908102919091018101518051600090815260329092526040909120805491925090611656576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8c816001015414611693576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015473ffffffffffffffffffffffffffffffffffffffff8c81169116146116e9576040517f4ca8886700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015474010000000000000000000000000000000000000000900460ff1661173f576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002810154700100000000000000000000000000000000900467ffffffffffffffff1615611799576040517f905e710700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b426002820180547fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff811670010000000000000000000000000000000067ffffffffffffffff948516810291821793849055604080516101408101825287548152600188015460208201529386169286169290921791830191909152680100000000000000008304841660608301529091049091166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff16151561010082015260068201805483916101208401916118a59061400f565b80601f01602080910402602001604051908101604052809291908181526020018280546118d19061400f565b801561191e5780601f106118f35761010080835404028352916020019161191e565b820191906000526020600020905b81548152906001019060200180831161190157829003601f168201915b505050505081525050858481518110611939576119396137fe565b6020026020010181905250816020015184848151811061195b5761195b6137fe565b6020026020010181815250508c8b73ffffffffffffffffffffffffffffffffffffffff16868581518110611991576119916137fe565b602002602001015160c0015173ffffffffffffffffffffffffffffffffffffffff167ff930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f61585600001516040516119e891815260200190565b60405180910390a450506001016115e1565b50611a0a84838360018b8b612a03565b9a9950505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff83166000908152603460209081526040808320858452918290529091205467ffffffffffffffff1615611a8c576040517fec9d6eeb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008381526020829052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff861690811790915590519091859173ffffffffffffffffffffffffffffffffffffffff8816917f92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a222991a450505050565b604080516020808252818301909252606091600091906020820181803683370190505090506000805b6020811015611be2576000858260208110611b5d57611b5d6137fe565b1a60f81b90507fff000000000000000000000000000000000000000000000000000000000000008116600003611b935750611be2565b80848481518110611ba657611ba66137fe565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053505060019182019101611b40565b5060008167ffffffffffffffff811115611bfe57611bfe61386b565b6040519080825280601f01601f191660200182016040528015611c28576020820181803683370190505b50905060005b82811015611c9c57838181518110611c4857611c486137fe565b602001015160f81c60f81b828281518110611c6557611c656137fe565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350600101611c2e565b50949350505050565b608081015167ffffffffffffffff1615801590611cd957504267ffffffffffffffff16816080015167ffffffffffffffff16105b15611d10576040517f1ab7da6b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6020808201516040808401516060808601518651855186880151868801519488015160808901518051908b012060a08a015173ffffffffffffffffffffffffffffffffffffffff871660009081529b8c9052988b2080549a9b989a9899611337997ffeb2925a02bae3dae48d424a0437a2b6ac939aa9230ddc55a1a76f065d988076999493928c611da0836140ca565b919050558e6080015160405160200161131c9b9a999897969594939291909a8b5273ffffffffffffffffffffffffffffffffffffffff998a1660208c015260408b019890985295909716606089015267ffffffffffffffff938416608089015291151560a088015260c087015260e0860152610100850193909352610120840152166101408201526101600190565b60408051808201909152600081526060602082015284516040805180820190915260008152606060208201528167ffffffffffffffff811115611e7457611e7461386b565b604051908082528060200260200182016040528015611e9d578160200160208202803683370190505b5060208201526040517fa2ea7c6e000000000000000000000000000000000000000000000000000000008152600481018990526000907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa158015611f0f573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0168201604052611f559190810190614102565b8051909150611f90576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008367ffffffffffffffff811115611fab57611fab61386b565b60405190808252806020026020018201604052801561204a57816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff909201910181611fc95790505b50905060008467ffffffffffffffff8111156120685761206861386b565b604051908082528060200260200182016040528015612091578160200160208202803683370190505b50905060005b858110156125205760008b82815181106120b3576120b36137fe565b60200260200101519050600067ffffffffffffffff16816020015167ffffffffffffffff16141580156120fe57504267ffffffffffffffff16816020015167ffffffffffffffff1611155b15612135576040517f08e8b93700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8460400151158015612148575080604001515b1561217f576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006040518061014001604052806000801b81526020018f81526020016121a34290565b67ffffffffffffffff168152602001836020015167ffffffffffffffff168152602001600067ffffffffffffffff16815260200183606001518152602001836000015173ffffffffffffffffffffffffffffffffffffffff1681526020018d73ffffffffffffffffffffffffffffffffffffffff16815260200183604001511515815260200183608001518152509050600080600090505b6122458382612df4565b600081815260326020526040902054909250156122645760010161223b565b81835260008281526032602090815260409182902085518155908501516001820155908401516002820180546060870151608088015167ffffffffffffffff908116700100000000000000000000000000000000027fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff92821668010000000000000000027fffffffffffffffffffffffffffffffff000000000000000000000000000000009094169190951617919091171691909117905560a0840151600382015560c084015160048201805473ffffffffffffffffffffffffffffffffffffffff9283167fffffffffffffffffffffffff000000000000000000000000000000000000000090911617905560e0850151600583018054610100880151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff000000000000000000000000000000000000000000909116929093169190911791909117905561012084015184919060068201906123e49082614228565b50505060608401511561243b57606084015160009081526032602052604090205461243b576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8287868151811061244e5761244e6137fe565b60200260200101819052508360a00151868681518110612470576124706137fe565b6020026020010181815250508189602001518681518110612493576124936137fe565b6020026020010181815250508f8e73ffffffffffffffffffffffffffffffffffffffff16856000015173ffffffffffffffffffffffffffffffffffffffff167f8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b358560405161250391815260200190565b60405180910390a4505050506125198160010190565b9050612097565b5061253083838360008c8c612a03565b845250919998505050505050505050565b606060008267ffffffffffffffff81111561255e5761255e61386b565b604051908082528060200260200182016040528015612587578160200160208202803683370190505b508451909150600090815b818110156126205760008782815181106125ae576125ae6137fe565b6020026020010151905060008151905060005b8181101561260c578281815181106125db576125db6137fe565b60200260200101518787815181106125f5576125f56137fe565b6020908102919091010152600195860195016125c1565b5050506126198160010190565b9050612592565b509195945050505050565b60008281526033602052604090205467ffffffffffffffff161561267b576040517f2e26794600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008281526033602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff85169081179091559051909184917f5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f9190a35050565b60003073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561275357507f000000000000000000000000000000000000000000000000000000000000000046145b1561277d57507f000000000000000000000000000000000000000000000000000000000000000090565b50604080517f00000000000000000000000000000000000000000000000000000000000000006020808301919091527f0000000000000000000000000000000000000000000000000000000000000000828401527f000000000000000000000000000000000000000000000000000000000000000060608301524660808301523060a0808401919091528351808403909101815260c0909201909252805191012090565b600061072a61282e6126ed565b83612e53565b60008060006128438585612e95565b9092509050600081600481111561285c5761285c614342565b14801561289457508573ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16145b156128a4576001925050506129fc565b6000808773ffffffffffffffffffffffffffffffffffffffff16631626ba7e60e01b88886040516024016128d9929190614371565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009094169390931790925290516129629190614392565b600060405180830381855afa9150503d806000811461299d576040519150601f19603f3d011682016040523d82523d6000602084013e6129a2565b606091505b50915091508180156129b5575080516020145b80156129f5575080517f1626ba7e00000000000000000000000000000000000000000000000000000000906129f390830160209081019084016143a4565b145b9450505050505b9392505050565b84516000906001819003612a5b57612a538888600081518110612a2857612a286137fe565b602002602001015188600081518110612a4357612a436137fe565b6020026020010151888888612eda565b915050612dea565b602088015173ffffffffffffffffffffffffffffffffffffffff8116612afc5760005b82811015612ae157878181518110612a9857612a986137fe565b6020026020010151600014612ad9576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600101612a7e565b508315612af157612af1856131f9565b600092505050612dea565b6000808273ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612b4a573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612b6e91906143bd565b905060005b84811015612c2b5760008a8281518110612b8f57612b8f6137fe565b6020026020010151905080600003612ba75750612c23565b82612bde576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b88811115612c18576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b978890039792909201915b600101612b73565b508715612d06576040517f88e5b2d900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8416906388e5b2d9908490612c88908e908e906004016143da565b60206040518083038185885af1158015612ca6573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612ccb91906143bd565b612d01576040517fbf2f3a8b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b612dd5565b6040517f91db0b7e00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8416906391db0b7e908490612d5c908e908e906004016143da565b60206040518083038185885af1158015612d7a573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612d9f91906143bd565b612dd5576040517fe8bee83900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8515612de457612de4876131f9565b50925050505b9695505050505050565b60208083015160c084015160e0850151604080870151606088015161010089015160a08a01516101208b01519451600099612e3599989796918c9101614493565b60405160208183030381529060405280519060200120905092915050565b6040517f190100000000000000000000000000000000000000000000000000000000000060208201526022810183905260428101829052600090606201612e35565b6000808251604103612ecb5760208301516040840151606085015160001a612ebf8782858561320c565b94509450505050612ed3565b506000905060025b9250929050565b602086015160009073ffffffffffffffffffffffffffffffffffffffff8116612f4e578515612f35576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8215612f4457612f44846131f9565b6000915050612dea565b8515613039578073ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612f9f573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612fc391906143bd565b612ff9576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b83861115613033576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b85840393505b8415613111576040517fe49617e100000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e49617e1908890613093908b90600401613771565b60206040518083038185885af11580156130b1573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906130d691906143bd565b61310c576040517fccf3bb2700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6131de565b6040517fe60c350500000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e60c3505908890613165908b90600401613771565b60206040518083038185885af1158015613183573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906131a891906143bd565b6131de576040517fbd8ba84d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82156131ed576131ed846131f9565b50939695505050505050565b8015613209576132093382613324565b50565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0831115613243575060009050600361331b565b8460ff16601b1415801561325b57508460ff16601c14155b1561326c575060009050600461331b565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa1580156132c0573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff81166133145760006001925092505061331b565b9150600090505b94509492505050565b80471015613393576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e636500000060448201526064015b60405180910390fd5b60008273ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d80600081146133ed576040519150601f19603f3d011682016040523d82523d6000602084013e6133f2565b606091505b5050905080610a63576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d61792068617665207265766572746564000000000000606482015260840161338a565b60008083601f84011261349557600080fd5b50813567ffffffffffffffff8111156134ad57600080fd5b6020830191508360208260051b8501011115612ed357600080fd5b600080602083850312156134db57600080fd5b823567ffffffffffffffff8111156134f257600080fd5b6134fe85828601613483565b90969095509350505050565b60005b8381101561352557818101518382015260200161350d565b50506000910152565b6000815180845261354681602086016020860161350a565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006129fc602083018461352e565b73ffffffffffffffffffffffffffffffffffffffff8116811461320957600080fd5b80356135b88161358b565b919050565b6000602082840312156135cf57600080fd5b81356129fc8161358b565b6000602082840312156135ec57600080fd5b813567ffffffffffffffff81111561360357600080fd5b820160e081850312156129fc57600080fd5b6020808252825182820181905260009190848201906040850190845b8181101561364d57835183529284019291840191600101613631565b50909695505050505050565b60006060828403121561366b57600080fd5b50919050565b60006020828403121561368357600080fd5b5035919050565b6000610140825184526020830151602085015260408301516136b8604086018267ffffffffffffffff169052565b5060608301516136d4606086018267ffffffffffffffff169052565b5060808301516136f0608086018267ffffffffffffffff169052565b5060a083015160a085015260c083015161372260c086018273ffffffffffffffffffffffffffffffffffffffff169052565b5060e083015161374a60e086018273ffffffffffffffffffffffffffffffffffffffff169052565b506101008381015115159085015261012080840151818601839052612dea8387018261352e565b6020815260006129fc602083018461368a565b6000610100828403121561366b57600080fd5b600080604083850312156137aa57600080fd5b82356137b58161358b565b946020939093013593505050565b6000602082840312156137d557600080fd5b813567ffffffffffffffff8111156137ec57600080fd5b8201604081850312156129fc57600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff6183360301811261386157600080fd5b9190910192915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405160a0810167ffffffffffffffff811182821017156138bd576138bd61386b565b60405290565b60405160c0810167ffffffffffffffff811182821017156138bd576138bd61386b565b6040516080810167ffffffffffffffff811182821017156138bd576138bd61386b565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff811182821017156139505761395061386b565b604052919050565b600067ffffffffffffffff8211156139725761397261386b565b5060051b60200190565b60006040828403121561398e57600080fd5b6040516040810181811067ffffffffffffffff821117156139b1576139b161386b565b604052823581526020928301359281019290925250919050565b6000606082840312156139dd57600080fd5b6040516060810181811067ffffffffffffffff82111715613a0057613a0061386b565b604052905080823560ff81168114613a1757600080fd5b8082525060208301356020820152604083013560408201525092915050565b600082601f830112613a4757600080fd5b81356020613a5c613a5783613958565b613909565b82815260609283028501820192828201919087851115613a7b57600080fd5b8387015b85811015613a9e57613a9189826139cb565b8452928401928101613a7f565b5090979650505050505050565b803567ffffffffffffffff811681146135b857600080fd5b600060a08236031215613ad557600080fd5b613add61389a565b8235815260208084013567ffffffffffffffff80821115613afd57600080fd5b9085019036601f830112613b1057600080fd5b8135613b1e613a5782613958565b81815260069190911b83018401908481019036831115613b3d57600080fd5b938501935b82851015613b6657613b54368661397c565b82528582019150604085019450613b42565b80868801525050506040860135925080831115613b8257600080fd5b5050613b9036828601613a36565b604083015250613ba2606084016135ad565b6060820152613bb360808401613aab565b608082015292915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8181038181111561072a5761072a613bbe565b801515811461320957600080fd5b600067ffffffffffffffff821115613c2857613c2861386b565b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01660200190565b600060c08284031215613c6657600080fd5b613c6e6138c3565b90508135613c7b8161358b565b81526020613c8a838201613aab565b818301526040830135613c9c81613c00565b604083015260608381013590830152608083013567ffffffffffffffff811115613cc557600080fd5b8301601f81018513613cd657600080fd5b8035613ce4613a5782613c0e565b8181528684838501011115613cf857600080fd5b818484018583013760008483830101528060808601525050505060a082013560a082015292915050565b600060e08236031215613d3457600080fd5b613d3c61389a565b82358152602083013567ffffffffffffffff811115613d5a57600080fd5b613d6636828601613c54565b602083015250613d7936604085016139cb565b604082015260a0830135613d8c8161358b565b6060820152613bb360c08401613aab565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff4183360301811261386157600080fd5b600061072a3683613c54565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc183360301811261386157600080fd5b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613e4657600080fd5b83018035915067ffffffffffffffff821115613e6157600080fd5b6020019150600581901b3603821315612ed357600080fd5b6000613e87613a5784613958565b80848252602080830192508560051b850136811115613ea557600080fd5b855b81811015613ee157803567ffffffffffffffff811115613ec75760008081fd5b613ed336828a01613c54565b865250938201938201613ea7565b50919695505050505050565b600060408284031215613eff57600080fd5b6129fc838361397c565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613f3e57600080fd5b83018035915067ffffffffffffffff821115613f5957600080fd5b6020019150600681901b3603821315612ed357600080fd5b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613fa657600080fd5b83018035915067ffffffffffffffff821115613fc157600080fd5b6020019150606081023603821315612ed357600080fd5b600060608284031215613fea57600080fd5b6129fc83836139cb565b60006020828403121561400657600080fd5b6129fc82613aab565b600181811c9082168061402357607f821691505b60208210810361366b577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000610100828403121561406f57600080fd5b61407761389a565b82358152614088846020850161397c565b602082015261409a84606085016139cb565b604082015260c08301356140ad8161358b565b60608201526140be60e08401613aab565b60808201529392505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036140fb576140fb613bbe565b5060010190565b6000602080838503121561411557600080fd5b825167ffffffffffffffff8082111561412d57600080fd5b908401906080828703121561414157600080fd5b6141496138e6565b825181528383015161415a8161358b565b81850152604083015161416c81613c00565b604082015260608301518281111561418357600080fd5b80840193505086601f84011261419857600080fd5b825191506141a8613a5783613c0e565b82815287858486010111156141bc57600080fd5b6141cb8386830187870161350a565b60608201529695505050505050565b601f821115610a6357600081815260208120601f850160051c810160208610156142015750805b601f850160051c820191505b818110156142205782815560010161420d565b505050505050565b815167ffffffffffffffff8111156142425761424261386b565b61425681614250845461400f565b846141da565b602080601f8311600181146142a957600084156142735750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555614220565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b828110156142f6578886015182559484019460019091019084016142d7565b508582101561433257878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b82815260406020820152600061438a604083018461352e565b949350505050565b6000825161386181846020870161350a565b6000602082840312156143b657600080fd5b5051919050565b6000602082840312156143cf57600080fd5b81516129fc81613c00565b6000604082016040835280855180835260608501915060608160051b8601019250602080880160005b8381101561444f577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa088870301855261443d86835161368a565b95509382019390820190600101614403565b50508584038187015286518085528782019482019350915060005b828110156144865784518452938101939281019260010161446a565b5091979650505050505050565b89815260007fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808b60601b166020840152808a60601b166034840152507fffffffffffffffff000000000000000000000000000000000000000000000000808960c01b166048840152808860c01b1660508401525085151560f81b6058830152846059830152835161452c81607985016020880161350a565b80830190507fffffffff000000000000000000000000000000000000000000000000000000008460e01b166079820152607d81019150509a995050505050505050505056fea164736f6c6343000813000a",
}

// EASABI is the input ABI used to generate the binding from.
// Deprecated: Use EASMetaData.ABI instead.
var EASABI = EASMetaData.ABI

// EASBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EASMetaData.Bin instead.
var EASBin = EASMetaData.Bin

// DeployEAS deploys a new Ethereum contract, binding an instance of EAS to it.
func DeployEAS(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *EAS, error) {
	parsed, err := EASMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EASBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &EAS{EASCaller: EASCaller{contract: contract}, EASTransactor: EASTransactor{contract: contract}, EASFilterer: EASFilterer{contract: contract}}, nil
}

// EAS is an auto generated Go binding around an Ethereum contract.
type EAS struct {
	EASCaller     // Read-only binding to the contract
	EASTransactor // Write-only binding to the contract
	EASFilterer   // Log filterer for contract events
}

// EASCaller is an auto generated read-only Go binding around an Ethereum contract.
type EASCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EASTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EASTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EASFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EASFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EASSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EASSession struct {
	Contract     *EAS              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// EASCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EASCallerSession struct {
	Contract *EASCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// EASTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EASTransactorSession struct {
	Contract     *EASTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// EASRaw is an auto generated low-level Go binding around an Ethereum contract.
type EASRaw struct {
	Contract *EAS // Generic contract binding to access the raw methods on
}

// EASCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EASCallerRaw struct {
	Contract *EASCaller // Generic read-only contract binding to access the raw methods on
}

// EASTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EASTransactorRaw struct {
	Contract *EASTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEAS creates a new instance of EAS, bound to a specific deployed contract.
func NewEAS(address common.Address, backend bind.ContractBackend) (*EAS, error) {
	contract, err := bindEAS(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &EAS{EASCaller: EASCaller{contract: contract}, EASTransactor: EASTransactor{contract: contract}, EASFilterer: EASFilterer{contract: contract}}, nil
}

// NewEASCaller creates a new read-only instance of EAS, bound to a specific deployed contract.
func NewEASCaller(address common.Address, caller bind.ContractCaller) (*EASCaller, error) {
	contract, err := bindEAS(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EASCaller{contract: contract}, nil
}

// NewEASTransactor creates a new write-only instance of EAS, bound to a specific deployed contract.
func NewEASTransactor(address common.Address, transactor bind.ContractTransactor) (*EASTransactor, error) {
	contract, err := bindEAS(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EASTransactor{contract: contract}, nil
}

// NewEASFilterer creates a new log filterer instance of EAS, bound to a specific deployed contract.
func NewEASFilterer(address common.Address, filterer bind.ContractFilterer) (*EASFilterer, error) {
	contract, err := bindEAS(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EASFilterer{contract: contract}, nil
}

// bindEAS binds a generic wrapper to an already deployed contract.
func bindEAS(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(EASABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EAS *EASRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EAS.Contract.EASCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EAS *EASRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EAS.Contract.EASTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EAS *EASRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EAS.Contract.EASTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EAS *EASCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EAS.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EAS *EASTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EAS.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EAS *EASTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EAS.Contract.contract.Transact(opts, method, params...)
}

// GetAttestTypeHash is a free data retrieval call binding the contract method 0x12b11a17.
//
// Solidity: function getAttestTypeHash() pure returns(bytes32)
func (_EAS *EASCaller) GetAttestTypeHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getAttestTypeHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetAttestTypeHash is a free data retrieval call binding the contract method 0x12b11a17.
//
// Solidity: function getAttestTypeHash() pure returns(bytes32)
func (_EAS *EASSession) GetAttestTypeHash() ([32]byte, error) {
	return _EAS.Contract.GetAttestTypeHash(&_EAS.CallOpts)
}

// GetAttestTypeHash is a free data retrieval call binding the contract method 0x12b11a17.
//
// Solidity: function getAttestTypeHash() pure returns(bytes32)
func (_EAS *EASCallerSession) GetAttestTypeHash() ([32]byte, error) {
	return _EAS.Contract.GetAttestTypeHash(&_EAS.CallOpts)
}

// GetAttestation is a free data retrieval call binding the contract method 0xa3112a64.
//
// Solidity: function getAttestation(bytes32 uid) view returns((bytes32,bytes32,uint64,uint64,uint64,bytes32,address,address,bool,bytes))
func (_EAS *EASCaller) GetAttestation(opts *bind.CallOpts, uid [32]byte) (Attestation, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getAttestation", uid)

	if err != nil {
		return *new(Attestation), err
	}

	out0 := *abi.ConvertType(out[0], new(Attestation)).(*Attestation)

	return out0, err

}

// GetAttestation is a free data retrieval call binding the contract method 0xa3112a64.
//
// Solidity: function getAttestation(bytes32 uid) view returns((bytes32,bytes32,uint64,uint64,uint64,bytes32,address,address,bool,bytes))
func (_EAS *EASSession) GetAttestation(uid [32]byte) (Attestation, error) {
	return _EAS.Contract.GetAttestation(&_EAS.CallOpts, uid)
}

// GetAttestation is a free data retrieval call binding the contract method 0xa3112a64.
//
// Solidity: function getAttestation(bytes32 uid) view returns((bytes32,bytes32,uint64,uint64,uint64,bytes32,address,address,bool,bytes))
func (_EAS *EASCallerSession) GetAttestation(uid [32]byte) (Attestation, error) {
	return _EAS.Contract.GetAttestation(&_EAS.CallOpts, uid)
}

// GetDomainSeparator is a free data retrieval call binding the contract method 0xed24911d.
//
// Solidity: function getDomainSeparator() view returns(bytes32)
func (_EAS *EASCaller) GetDomainSeparator(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getDomainSeparator")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetDomainSeparator is a free data retrieval call binding the contract method 0xed24911d.
//
// Solidity: function getDomainSeparator() view returns(bytes32)
func (_EAS *EASSession) GetDomainSeparator() ([32]byte, error) {
	return _EAS.Contract.GetDomainSeparator(&_EAS.CallOpts)
}

// GetDomainSeparator is a free data retrieval call binding the contract method 0xed24911d.
//
// Solidity: function getDomainSeparator() view returns(bytes32)
func (_EAS *EASCallerSession) GetDomainSeparator() ([32]byte, error) {
	return _EAS.Contract.GetDomainSeparator(&_EAS.CallOpts)
}

// GetName is a free data retrieval call binding the contract method 0x17d7de7c.
//
// Solidity: function getName() view returns(string)
func (_EAS *EASCaller) GetName(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getName")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GetName is a free data retrieval call binding the contract method 0x17d7de7c.
//
// Solidity: function getName() view returns(string)
func (_EAS *EASSession) GetName() (string, error) {
	return _EAS.Contract.GetName(&_EAS.CallOpts)
}

// GetName is a free data retrieval call binding the contract method 0x17d7de7c.
//
// Solidity: function getName() view returns(string)
func (_EAS *EASCallerSession) GetName() (string, error) {
	return _EAS.Contract.GetName(&_EAS.CallOpts)
}

// GetNonce is a free data retrieval call binding the contract method 0x2d0335ab.
//
// Solidity: function getNonce(address account) view returns(uint256)
func (_EAS *EASCaller) GetNonce(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getNonce", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNonce is a free data retrieval call binding the contract method 0x2d0335ab.
//
// Solidity: function getNonce(address account) view returns(uint256)
func (_EAS *EASSession) GetNonce(account common.Address) (*big.Int, error) {
	return _EAS.Contract.GetNonce(&_EAS.CallOpts, account)
}

// GetNonce is a free data retrieval call binding the contract method 0x2d0335ab.
//
// Solidity: function getNonce(address account) view returns(uint256)
func (_EAS *EASCallerSession) GetNonce(account common.Address) (*big.Int, error) {
	return _EAS.Contract.GetNonce(&_EAS.CallOpts, account)
}

// GetRevokeOffchain is a free data retrieval call binding the contract method 0xb469318d.
//
// Solidity: function getRevokeOffchain(address revoker, bytes32 data) view returns(uint64)
func (_EAS *EASCaller) GetRevokeOffchain(opts *bind.CallOpts, revoker common.Address, data [32]byte) (uint64, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getRevokeOffchain", revoker, data)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetRevokeOffchain is a free data retrieval call binding the contract method 0xb469318d.
//
// Solidity: function getRevokeOffchain(address revoker, bytes32 data) view returns(uint64)
func (_EAS *EASSession) GetRevokeOffchain(revoker common.Address, data [32]byte) (uint64, error) {
	return _EAS.Contract.GetRevokeOffchain(&_EAS.CallOpts, revoker, data)
}

// GetRevokeOffchain is a free data retrieval call binding the contract method 0xb469318d.
//
// Solidity: function getRevokeOffchain(address revoker, bytes32 data) view returns(uint64)
func (_EAS *EASCallerSession) GetRevokeOffchain(revoker common.Address, data [32]byte) (uint64, error) {
	return _EAS.Contract.GetRevokeOffchain(&_EAS.CallOpts, revoker, data)
}

// GetRevokeTypeHash is a free data retrieval call binding the contract method 0xb83010d3.
//
// Solidity: function getRevokeTypeHash() pure returns(bytes32)
func (_EAS *EASCaller) GetRevokeTypeHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getRevokeTypeHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRevokeTypeHash is a free data retrieval call binding the contract method 0xb83010d3.
//
// Solidity: function getRevokeTypeHash() pure returns(bytes32)
func (_EAS *EASSession) GetRevokeTypeHash() ([32]byte, error) {
	return _EAS.Contract.GetRevokeTypeHash(&_EAS.CallOpts)
}

// GetRevokeTypeHash is a free data retrieval call binding the contract method 0xb83010d3.
//
// Solidity: function getRevokeTypeHash() pure returns(bytes32)
func (_EAS *EASCallerSession) GetRevokeTypeHash() ([32]byte, error) {
	return _EAS.Contract.GetRevokeTypeHash(&_EAS.CallOpts)
}

// GetSchemaRegistry is a free data retrieval call binding the contract method 0xf10b5cc8.
//
// Solidity: function getSchemaRegistry() pure returns(address)
func (_EAS *EASCaller) GetSchemaRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getSchemaRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetSchemaRegistry is a free data retrieval call binding the contract method 0xf10b5cc8.
//
// Solidity: function getSchemaRegistry() pure returns(address)
func (_EAS *EASSession) GetSchemaRegistry() (common.Address, error) {
	return _EAS.Contract.GetSchemaRegistry(&_EAS.CallOpts)
}

// GetSchemaRegistry is a free data retrieval call binding the contract method 0xf10b5cc8.
//
// Solidity: function getSchemaRegistry() pure returns(address)
func (_EAS *EASCallerSession) GetSchemaRegistry() (common.Address, error) {
	return _EAS.Contract.GetSchemaRegistry(&_EAS.CallOpts)
}

// GetTimestamp is a free data retrieval call binding the contract method 0xd45c4435.
//
// Solidity: function getTimestamp(bytes32 data) view returns(uint64)
func (_EAS *EASCaller) GetTimestamp(opts *bind.CallOpts, data [32]byte) (uint64, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "getTimestamp", data)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetTimestamp is a free data retrieval call binding the contract method 0xd45c4435.
//
// Solidity: function getTimestamp(bytes32 data) view returns(uint64)
func (_EAS *EASSession) GetTimestamp(data [32]byte) (uint64, error) {
	return _EAS.Contract.GetTimestamp(&_EAS.CallOpts, data)
}

// GetTimestamp is a free data retrieval call binding the contract method 0xd45c4435.
//
// Solidity: function getTimestamp(bytes32 data) view returns(uint64)
func (_EAS *EASCallerSession) GetTimestamp(data [32]byte) (uint64, error) {
	return _EAS.Contract.GetTimestamp(&_EAS.CallOpts, data)
}

// IsAttestationValid is a free data retrieval call binding the contract method 0xe30bb563.
//
// Solidity: function isAttestationValid(bytes32 uid) view returns(bool)
func (_EAS *EASCaller) IsAttestationValid(opts *bind.CallOpts, uid [32]byte) (bool, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "isAttestationValid", uid)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAttestationValid is a free data retrieval call binding the contract method 0xe30bb563.
//
// Solidity: function isAttestationValid(bytes32 uid) view returns(bool)
func (_EAS *EASSession) IsAttestationValid(uid [32]byte) (bool, error) {
	return _EAS.Contract.IsAttestationValid(&_EAS.CallOpts, uid)
}

// IsAttestationValid is a free data retrieval call binding the contract method 0xe30bb563.
//
// Solidity: function isAttestationValid(bytes32 uid) view returns(bool)
func (_EAS *EASCallerSession) IsAttestationValid(uid [32]byte) (bool, error) {
	return _EAS.Contract.IsAttestationValid(&_EAS.CallOpts, uid)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_EAS *EASCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _EAS.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_EAS *EASSession) Version() (string, error) {
	return _EAS.Contract.Version(&_EAS.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_EAS *EASCallerSession) Version() (string, error) {
	return _EAS.Contract.Version(&_EAS.CallOpts)
}

// Attest is a paid mutator transaction binding the contract method 0xf17325e7.
//
// Solidity: function attest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)) request) payable returns(bytes32)
func (_EAS *EASTransactor) Attest(opts *bind.TransactOpts, request AttestationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "attest", request)
}

// Attest is a paid mutator transaction binding the contract method 0xf17325e7.
//
// Solidity: function attest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)) request) payable returns(bytes32)
func (_EAS *EASSession) Attest(request AttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.Attest(&_EAS.TransactOpts, request)
}

// Attest is a paid mutator transaction binding the contract method 0xf17325e7.
//
// Solidity: function attest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)) request) payable returns(bytes32)
func (_EAS *EASTransactorSession) Attest(request AttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.Attest(&_EAS.TransactOpts, request)
}

// AttestByDelegation is a paid mutator transaction binding the contract method 0x3c042715.
//
// Solidity: function attestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256),(uint8,bytes32,bytes32),address,uint64) delegatedRequest) payable returns(bytes32)
func (_EAS *EASTransactor) AttestByDelegation(opts *bind.TransactOpts, delegatedRequest DelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "attestByDelegation", delegatedRequest)
}

// AttestByDelegation is a paid mutator transaction binding the contract method 0x3c042715.
//
// Solidity: function attestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256),(uint8,bytes32,bytes32),address,uint64) delegatedRequest) payable returns(bytes32)
func (_EAS *EASSession) AttestByDelegation(delegatedRequest DelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.AttestByDelegation(&_EAS.TransactOpts, delegatedRequest)
}

// AttestByDelegation is a paid mutator transaction binding the contract method 0x3c042715.
//
// Solidity: function attestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256),(uint8,bytes32,bytes32),address,uint64) delegatedRequest) payable returns(bytes32)
func (_EAS *EASTransactorSession) AttestByDelegation(delegatedRequest DelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.AttestByDelegation(&_EAS.TransactOpts, delegatedRequest)
}

// IncreaseNonce is a paid mutator transaction binding the contract method 0x79f7573a.
//
// Solidity: function increaseNonce(uint256 newNonce) returns()
func (_EAS *EASTransactor) IncreaseNonce(opts *bind.TransactOpts, newNonce *big.Int) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "increaseNonce", newNonce)
}

// IncreaseNonce is a paid mutator transaction binding the contract method 0x79f7573a.
//
// Solidity: function increaseNonce(uint256 newNonce) returns()
func (_EAS *EASSession) IncreaseNonce(newNonce *big.Int) (*types.Transaction, error) {
	return _EAS.Contract.IncreaseNonce(&_EAS.TransactOpts, newNonce)
}

// IncreaseNonce is a paid mutator transaction binding the contract method 0x79f7573a.
//
// Solidity: function increaseNonce(uint256 newNonce) returns()
func (_EAS *EASTransactorSession) IncreaseNonce(newNonce *big.Int) (*types.Transaction, error) {
	return _EAS.Contract.IncreaseNonce(&_EAS.TransactOpts, newNonce)
}

// MultiAttest is a paid mutator transaction binding the contract method 0x44adc90e.
//
// Solidity: function multiAttest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[])[] multiRequests) payable returns(bytes32[])
func (_EAS *EASTransactor) MultiAttest(opts *bind.TransactOpts, multiRequests []MultiAttestationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiAttest", multiRequests)
}

// MultiAttest is a paid mutator transaction binding the contract method 0x44adc90e.
//
// Solidity: function multiAttest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[])[] multiRequests) payable returns(bytes32[])
func (_EAS *EASSession) MultiAttest(multiRequests []MultiAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiAttest(&_EAS.TransactOpts, multiRequests)
}

// MultiAttest is a paid mutator transaction binding the contract method 0x44adc90e.
//
// Solidity: function multiAttest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[])[] multiRequests) payable returns(bytes32[])
func (_EAS *EASTransactorSession) MultiAttest(multiRequests []MultiAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiAttest(&_EAS.TransactOpts, multiRequests)
}

// MultiAttestByDelegation is a paid mutator transaction binding the contract method 0x95411525.
//
// Solidity: function multiAttestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[],(uint8,bytes32,bytes32)[],address,uint64)[] multiDelegatedRequests) payable returns(bytes32[])
func (_EAS *EASTransactor) MultiAttestByDelegation(opts *bind.TransactOpts, multiDelegatedRequests []MultiDelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiAttestByDelegation", multiDelegatedRequests)
}

// MultiAttestByDelegation is a paid mutator transaction binding the contract method 0x95411525.
//
// Solidity: function multiAttestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[],(uint8,bytes32,bytes32)[],address,uint64)[] multiDelegatedRequests) payable returns(bytes32[])
func (_EAS *EASSession) MultiAttestByDelegation(multiDelegatedRequests []MultiDelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiAttestByDelegation(&_EAS.TransactOpts, multiDelegatedRequests)
}

// MultiAttestByDelegation is a paid mutator transaction binding the contract method 0x95411525.
//
// Solidity: function multiAttestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[],(uint8,bytes32,bytes32)[],address,uint64)[] multiDelegatedRequests) payable returns(bytes32[])
func (_EAS *EASTransactorSession) MultiAttestByDelegation(multiDelegatedRequests []MultiDelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiAttestByDelegation(&_EAS.TransactOpts, multiDelegatedRequests)
}

// MultiRevoke is a paid mutator transaction binding the contract method 0x4cb7e9e5.
//
// Solidity: function multiRevoke((bytes32,(bytes32,uint256)[])[] multiRequests) payable returns()
func (_EAS *EASTransactor) MultiRevoke(opts *bind.TransactOpts, multiRequests []MultiRevocationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiRevoke", multiRequests)
}

// MultiRevoke is a paid mutator transaction binding the contract method 0x4cb7e9e5.
//
// Solidity: function multiRevoke((bytes32,(bytes32,uint256)[])[] multiRequests) payable returns()
func (_EAS *EASSession) MultiRevoke(multiRequests []MultiRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiRevoke(&_EAS.TransactOpts, multiRequests)
}

// MultiRevoke is a paid mutator transaction binding the contract method 0x4cb7e9e5.
//
// Solidity: function multiRevoke((bytes32,(bytes32,uint256)[])[] multiRequests) payable returns()
func (_EAS *EASTransactorSession) MultiRevoke(multiRequests []MultiRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiRevoke(&_EAS.TransactOpts, multiRequests)
}

// MultiRevokeByDelegation is a paid mutator transaction binding the contract method 0x0eabf660.
//
// Solidity: function multiRevokeByDelegation((bytes32,(bytes32,uint256)[],(uint8,bytes32,bytes32)[],address,uint64)[] multiDelegatedRequests) payable returns()
func (_EAS *EASTransactor) MultiRevokeByDelegation(opts *bind.TransactOpts, multiDelegatedRequests []MultiDelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiRevokeByDelegation", multiDelegatedRequests)
}

// MultiRevokeByDelegation is a paid mutator transaction binding the contract method 0x0eabf660.
//
// Solidity: function multiRevokeByDelegation((bytes32,(bytes32,uint256)[],(uint8,bytes32,bytes32)[],address,uint64)[] multiDelegatedRequests) payable returns()
func (_EAS *EASSession) MultiRevokeByDelegation(multiDelegatedRequests []MultiDelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiRevokeByDelegation(&_EAS.TransactOpts, multiDelegatedRequests)
}

// MultiRevokeByDelegation is a paid mutator transaction binding the contract method 0x0eabf660.
//
// Solidity: function multiRevokeByDelegation((bytes32,(bytes32,uint256)[],(uint8,bytes32,bytes32)[],address,uint64)[] multiDelegatedRequests) payable returns()
func (_EAS *EASTransactorSession) MultiRevokeByDelegation(multiDelegatedRequests []MultiDelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiRevokeByDelegation(&_EAS.TransactOpts, multiDelegatedRequests)
}

// MultiRevokeOffchain is a paid mutator transaction binding the contract method 0x13893f61.
//
// Solidity: function multiRevokeOffchain(bytes32[] data) returns(uint64)
func (_EAS *EASTransactor) MultiRevokeOffchain(opts *bind.TransactOpts, data [][32]byte) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiRevokeOffchain", data)
}

// MultiRevokeOffchain is a paid mutator transaction binding the contract method 0x13893f61.
//
// Solidity: function multiRevokeOffchain(bytes32[] data) returns(uint64)
func (_EAS *EASSession) MultiRevokeOffchain(data [][32]byte) (*types.Transaction, error) {
	return _EAS.Contract.MultiRevokeOffchain(&_EAS.TransactOpts, data)
}

// MultiRevokeOffchain is a paid mutator transaction binding the contract method 0x13893f61.
//
// Solidity: function multiRevokeOffchain(bytes32[] data) returns(uint64)
func (_EAS *EASTransactorSession) MultiRevokeOffchain(data [][32]byte) (*types.Transaction, error) {
	return _EAS.Contract.MultiRevokeOffchain(&_EAS.TransactOpts, data)
}

// MultiTimestamp is a paid mutator transaction binding the contract method 0xe71ff365.
//
// Solidity: function multiTimestamp(bytes32[] data) returns(uint64)
func (_EAS *EASTransactor) MultiTimestamp(opts *bind.TransactOpts, data [][32]byte) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiTimestamp", data)
}

// MultiTimestamp is a paid mutator transaction binding the contract method 0xe71ff365.
//
// Solidity: function multiTimestamp(bytes32[] data) returns(uint64)
func (_EAS *EASSession) MultiTimestamp(data [][32]byte) (*types.Transaction, error) {
	return _EAS.Contract.MultiTimestamp(&_EAS.TransactOpts, data)
}

// MultiTimestamp is a paid mutator transaction binding the contract method 0xe71ff365.
//
// Solidity: function multiTimestamp(bytes32[] data) returns(uint64)
func (_EAS *EASTransactorSession) MultiTimestamp(data [][32]byte) (*types.Transaction, error) {
	return _EAS.Contract.MultiTimestamp(&_EAS.TransactOpts, data)
}

// Revoke is a paid mutator transaction binding the contract method 0x46926267.
//
// Solidity: function revoke((bytes32,(bytes32,uint256)) request) payable returns()
func (_EAS *EASTransactor) Revoke(opts *bind.TransactOpts, request RevocationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "revoke", request)
}

// Revoke is a paid mutator transaction binding the contract method 0x46926267.
//
// Solidity: function revoke((bytes32,(bytes32,uint256)) request) payable returns()
func (_EAS *EASSession) Revoke(request RevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.Revoke(&_EAS.TransactOpts, request)
}

// Revoke is a paid mutator transaction binding the contract method 0x46926267.
//
// Solidity: function revoke((bytes32,(bytes32,uint256)) request) payable returns()
func (_EAS *EASTransactorSession) Revoke(request RevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.Revoke(&_EAS.TransactOpts, request)
}

// RevokeByDelegation is a paid mutator transaction binding the contract method 0xa6d4dbc7.
//
// Solidity: function revokeByDelegation((bytes32,(bytes32,uint256),(uint8,bytes32,bytes32),address,uint64) delegatedRequest) payable returns()
func (_EAS *EASTransactor) RevokeByDelegation(opts *bind.TransactOpts, delegatedRequest DelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "revokeByDelegation", delegatedRequest)
}

// RevokeByDelegation is a paid mutator transaction binding the contract method 0xa6d4dbc7.
//
// Solidity: function revokeByDelegation((bytes32,(bytes32,uint256),(uint8,bytes32,bytes32),address,uint64) delegatedRequest) payable returns()
func (_EAS *EASSession) RevokeByDelegation(delegatedRequest DelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.RevokeByDelegation(&_EAS.TransactOpts, delegatedRequest)
}

// RevokeByDelegation is a paid mutator transaction binding the contract method 0xa6d4dbc7.
//
// Solidity: function revokeByDelegation((bytes32,(bytes32,uint256),(uint8,bytes32,bytes32),address,uint64) delegatedRequest) payable returns()
func (_EAS *EASTransactorSession) RevokeByDelegation(delegatedRequest DelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.RevokeByDelegation(&_EAS.TransactOpts, delegatedRequest)
}

// RevokeOffchain is a paid mutator transaction binding the contract method 0xcf190f34.
//
// Solidity: function revokeOffchain(bytes32 data) returns(uint64)
func (_EAS *EASTransactor) RevokeOffchain(opts *bind.TransactOpts, data [32]byte) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "revokeOffchain", data)
}

// RevokeOffchain is a paid mutator transaction binding the contract method 0xcf190f34.
//
// Solidity: function revokeOffchain(bytes32 data) returns(uint64)
func (_EAS *EASSession) RevokeOffchain(data [32]byte) (*types.Transaction, error) {
	return _EAS.Contract.RevokeOffchain(&_EAS.TransactOpts, data)
}

// RevokeOffchain is a paid mutator transaction binding the contract method 0xcf190f34.
//
// Solidity: function revokeOffchain(bytes32 data) returns(uint64)
func (_EAS *EASTransactorSession) RevokeOffchain(data [32]byte) (*types.Transaction, error) {
	return _EAS.Contract.RevokeOffchain(&_EAS.TransactOpts, data)
}

// Timestamp is a paid mutator transaction binding the contract method 0x4d003070.
//
// Solidity: function timestamp(bytes32 data) returns(uint64)
func (_EAS *EASTransactor) Timestamp(opts *bind.TransactOpts, data [32]byte) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "timestamp", data)
}

// Timestamp is a paid mutator transaction binding the contract method 0x4d003070.
//
// Solidity: function timestamp(bytes32 data) returns(uint64)
func (_EAS *EASSession) Timestamp(data [32]byte) (*types.Transaction, error) {
	return _EAS.Contract.Timestamp(&_EAS.TransactOpts, data)
}

// Timestamp is a paid mutator transaction binding the contract method 0x4d003070.
//
// Solidity: function timestamp(bytes32 data) returns(uint64)
func (_EAS *EASTransactorSession) Timestamp(data [32]byte) (*types.Transaction, error) {
	return _EAS.Contract.Timestamp(&_EAS.TransactOpts, data)
}

// EASAttestedIterator is returned from FilterAttested and is used to iterate over the raw logs and unpacked data for Attested events raised by the EAS contract.
type EASAttestedIterator struct {
	Event *EASAttested // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EASAttestedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EASAttested)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EASAttested)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EASAttestedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EASAttestedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EASAttested represents a Attested event raised by the EAS contract.
type EASAttested struct {
	Recipient common.Address
	Attester  common.Address
	Uid       [32]byte
	SchemaUID [32]byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAttested is a free log retrieval operation binding the contract event 0x8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b35.
//
// Solidity: event Attested(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schemaUID)
func (_EAS *EASFilterer) FilterAttested(opts *bind.FilterOpts, recipient []common.Address, attester []common.Address, schemaUID [][32]byte) (*EASAttestedIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaUIDRule []interface{}
	for _, schemaUIDItem := range schemaUID {
		schemaUIDRule = append(schemaUIDRule, schemaUIDItem)
	}

	logs, sub, err := _EAS.contract.FilterLogs(opts, "Attested", recipientRule, attesterRule, schemaUIDRule)
	if err != nil {
		return nil, err
	}
	return &EASAttestedIterator{contract: _EAS.contract, event: "Attested", logs: logs, sub: sub}, nil
}

// WatchAttested is a free log subscription operation binding the contract event 0x8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b35.
//
// Solidity: event Attested(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schemaUID)
func (_EAS *EASFilterer) WatchAttested(opts *bind.WatchOpts, sink chan<- *EASAttested, recipient []common.Address, attester []common.Address, schemaUID [][32]byte) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaUIDRule []interface{}
	for _, schemaUIDItem := range schemaUID {
		schemaUIDRule = append(schemaUIDRule, schemaUIDItem)
	}

	logs, sub, err := _EAS.contract.WatchLogs(opts, "Attested", recipientRule, attesterRule, schemaUIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EASAttested)
				if err := _EAS.contract.UnpackLog(event, "Attested", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAttested is a log parse operation binding the contract event 0x8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b35.
//
// Solidity: event Attested(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schemaUID)
func (_EAS *EASFilterer) ParseAttested(log types.Log) (*EASAttested, error) {
	event := new(EASAttested)
	if err := _EAS.contract.UnpackLog(event, "Attested", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EASNonceIncreasedIterator is returned from FilterNonceIncreased and is used to iterate over the raw logs and unpacked data for NonceIncreased events raised by the EAS contract.
type EASNonceIncreasedIterator struct {
	Event *EASNonceIncreased // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EASNonceIncreasedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EASNonceIncreased)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EASNonceIncreased)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EASNonceIncreasedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EASNonceIncreasedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EASNonceIncreased represents a NonceIncreased event raised by the EAS contract.
type EASNonceIncreased struct {
	OldNonce *big.Int
	NewNonce *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterNonceIncreased is a free log retrieval operation binding the contract event 0x57b09af877df9068fd60a69d7b21f5576b8b38955812d6ae4ac52942f1e38fb7.
//
// Solidity: event NonceIncreased(uint256 oldNonce, uint256 newNonce)
func (_EAS *EASFilterer) FilterNonceIncreased(opts *bind.FilterOpts) (*EASNonceIncreasedIterator, error) {

	logs, sub, err := _EAS.contract.FilterLogs(opts, "NonceIncreased")
	if err != nil {
		return nil, err
	}
	return &EASNonceIncreasedIterator{contract: _EAS.contract, event: "NonceIncreased", logs: logs, sub: sub}, nil
}

// WatchNonceIncreased is a free log subscription operation binding the contract event 0x57b09af877df9068fd60a69d7b21f5576b8b38955812d6ae4ac52942f1e38fb7.
//
// Solidity: event NonceIncreased(uint256 oldNonce, uint256 newNonce)
func (_EAS *EASFilterer) WatchNonceIncreased(opts *bind.WatchOpts, sink chan<- *EASNonceIncreased) (event.Subscription, error) {

	logs, sub, err := _EAS.contract.WatchLogs(opts, "NonceIncreased")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EASNonceIncreased)
				if err := _EAS.contract.UnpackLog(event, "NonceIncreased", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseNonceIncreased is a log parse operation binding the contract event 0x57b09af877df9068fd60a69d7b21f5576b8b38955812d6ae4ac52942f1e38fb7.
//
// Solidity: event NonceIncreased(uint256 oldNonce, uint256 newNonce)
func (_EAS *EASFilterer) ParseNonceIncreased(log types.Log) (*EASNonceIncreased, error) {
	event := new(EASNonceIncreased)
	if err := _EAS.contract.UnpackLog(event, "NonceIncreased", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EASRevokedIterator is returned from FilterRevoked and is used to iterate over the raw logs and unpacked data for Revoked events raised by the EAS contract.
type EASRevokedIterator struct {
	Event *EASRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EASRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EASRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EASRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EASRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EASRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EASRevoked represents a Revoked event raised by the EAS contract.
type EASRevoked struct {
	Recipient common.Address
	Attester  common.Address
	Uid       [32]byte
	SchemaUID [32]byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRevoked is a free log retrieval operation binding the contract event 0xf930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f615.
//
// Solidity: event Revoked(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schemaUID)
func (_EAS *EASFilterer) FilterRevoked(opts *bind.FilterOpts, recipient []common.Address, attester []common.Address, schemaUID [][32]byte) (*EASRevokedIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaUIDRule []interface{}
	for _, schemaUIDItem := range schemaUID {
		schemaUIDRule = append(schemaUIDRule, schemaUIDItem)
	}

	logs, sub, err := _EAS.contract.FilterLogs(opts, "Revoked", recipientRule, attesterRule, schemaUIDRule)
	if err != nil {
		return nil, err
	}
	return &EASRevokedIterator{contract: _EAS.contract, event: "Revoked", logs: logs, sub: sub}, nil
}

// WatchRevoked is a free log subscription operation binding the contract event 0xf930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f615.
//
// Solidity: event Revoked(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schemaUID)
func (_EAS *EASFilterer) WatchRevoked(opts *bind.WatchOpts, sink chan<- *EASRevoked, recipient []common.Address, attester []common.Address, schemaUID [][32]byte) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaUIDRule []interface{}
	for _, schemaUIDItem := range schemaUID {
		schemaUIDRule = append(schemaUIDRule, schemaUIDItem)
	}

	logs, sub, err := _EAS.contract.WatchLogs(opts, "Revoked", recipientRule, attesterRule, schemaUIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EASRevoked)
				if err := _EAS.contract.UnpackLog(event, "Revoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRevoked is a log parse operation binding the contract event 0xf930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f615.
//
// Solidity: event Revoked(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schemaUID)
func (_EAS *EASFilterer) ParseRevoked(log types.Log) (*EASRevoked, error) {
	event := new(EASRevoked)
	if err := _EAS.contract.UnpackLog(event, "Revoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EASRevokedOffchainIterator is returned from FilterRevokedOffchain and is used to iterate over the raw logs and unpacked data for RevokedOffchain events raised by the EAS contract.
type EASRevokedOffchainIterator struct {
	Event *EASRevokedOffchain // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EASRevokedOffchainIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EASRevokedOffchain)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EASRevokedOffchain)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EASRevokedOffchainIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EASRevokedOffchainIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EASRevokedOffchain represents a RevokedOffchain event raised by the EAS contract.
type EASRevokedOffchain struct {
	Revoker   common.Address
	Data      [32]byte
	Timestamp uint64
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRevokedOffchain is a free log retrieval operation binding the contract event 0x92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a2229.
//
// Solidity: event RevokedOffchain(address indexed revoker, bytes32 indexed data, uint64 indexed timestamp)
func (_EAS *EASFilterer) FilterRevokedOffchain(opts *bind.FilterOpts, revoker []common.Address, data [][32]byte, timestamp []uint64) (*EASRevokedOffchainIterator, error) {

	var revokerRule []interface{}
	for _, revokerItem := range revoker {
		revokerRule = append(revokerRule, revokerItem)
	}
	var dataRule []interface{}
	for _, dataItem := range data {
		dataRule = append(dataRule, dataItem)
	}
	var timestampRule []interface{}
	for _, timestampItem := range timestamp {
		timestampRule = append(timestampRule, timestampItem)
	}

	logs, sub, err := _EAS.contract.FilterLogs(opts, "RevokedOffchain", revokerRule, dataRule, timestampRule)
	if err != nil {
		return nil, err
	}
	return &EASRevokedOffchainIterator{contract: _EAS.contract, event: "RevokedOffchain", logs: logs, sub: sub}, nil
}

// WatchRevokedOffchain is a free log subscription operation binding the contract event 0x92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a2229.
//
// Solidity: event RevokedOffchain(address indexed revoker, bytes32 indexed data, uint64 indexed timestamp)
func (_EAS *EASFilterer) WatchRevokedOffchain(opts *bind.WatchOpts, sink chan<- *EASRevokedOffchain, revoker []common.Address, data [][32]byte, timestamp []uint64) (event.Subscription, error) {

	var revokerRule []interface{}
	for _, revokerItem := range revoker {
		revokerRule = append(revokerRule, revokerItem)
	}
	var dataRule []interface{}
	for _, dataItem := range data {
		dataRule = append(dataRule, dataItem)
	}
	var timestampRule []interface{}
	for _, timestampItem := range timestamp {
		timestampRule = append(timestampRule, timestampItem)
	}

	logs, sub, err := _EAS.contract.WatchLogs(opts, "RevokedOffchain", revokerRule, dataRule, timestampRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EASRevokedOffchain)
				if err := _EAS.contract.UnpackLog(event, "RevokedOffchain", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRevokedOffchain is a log parse operation binding the contract event 0x92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a2229.
//
// Solidity: event RevokedOffchain(address indexed revoker, bytes32 indexed data, uint64 indexed timestamp)
func (_EAS *EASFilterer) ParseRevokedOffchain(log types.Log) (*EASRevokedOffchain, error) {
	event := new(EASRevokedOffchain)
	if err := _EAS.contract.UnpackLog(event, "RevokedOffchain", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EASTimestampedIterator is returned from FilterTimestamped and is used to iterate over the raw logs and unpacked data for Timestamped events raised by the EAS contract.
type EASTimestampedIterator struct {
	Event *EASTimestamped // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EASTimestampedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EASTimestamped)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(EASTimestamped)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EASTimestampedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EASTimestampedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EASTimestamped represents a Timestamped event raised by the EAS contract.
type EASTimestamped struct {
	Data      [32]byte
	Timestamp uint64
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterTimestamped is a free log retrieval operation binding the contract event 0x5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f.
//
// Solidity: event Timestamped(bytes32 indexed data, uint64 indexed timestamp)
func (_EAS *EASFilterer) FilterTimestamped(opts *bind.FilterOpts, data [][32]byte, timestamp []uint64) (*EASTimestampedIterator, error) {

	var dataRule []interface{}
	for _, dataItem := range data {
		dataRule = append(dataRule, dataItem)
	}
	var timestampRule []interface{}
	for _, timestampItem := range timestamp {
		timestampRule = append(timestampRule, timestampItem)
	}

	logs, sub, err := _EAS.contract.FilterLogs(opts, "Timestamped", dataRule, timestampRule)
	if err != nil {
		return nil, err
	}
	return &EASTimestampedIterator{contract: _EAS.contract, event: "Timestamped", logs: logs, sub: sub}, nil
}

// WatchTimestamped is a free log subscription operation binding the contract event 0x5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f.
//
// Solidity: event Timestamped(bytes32 indexed data, uint64 indexed timestamp)
func (_EAS *EASFilterer) WatchTimestamped(opts *bind.WatchOpts, sink chan<- *EASTimestamped, data [][32]byte, timestamp []uint64) (event.Subscription, error) {

	var dataRule []interface{}
	for _, dataItem := range data {
		dataRule = append(dataRule, dataItem)
	}
	var timestampRule []interface{}
	for _, timestampItem := range timestamp {
		timestampRule = append(timestampRule, timestampItem)
	}

	logs, sub, err := _EAS.contract.WatchLogs(opts, "Timestamped", dataRule, timestampRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EASTimestamped)
				if err := _EAS.contract.UnpackLog(event, "Timestamped", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTimestamped is a log parse operation binding the contract event 0x5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f.
//
// Solidity: event Timestamped(bytes32 indexed data, uint64 indexed timestamp)
func (_EAS *EASFilterer) ParseTimestamped(log types.Log) (*EASTimestamped, error) {
	event := new(EASTimestamped)
	if err := _EAS.contract.UnpackLog(event, "Timestamped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
