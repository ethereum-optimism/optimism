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
	_ = abi.ConvertType
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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AccessDenied\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyRevoked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyRevokedOffchain\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyTimestamped\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DeadlineExpired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InsufficientValue\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAttestation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAttestations\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidExpirationTime\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidLength\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidNonce\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidOffset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRegistry\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRevocation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRevocations\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSchema\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignature\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidVerifier\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Irrevocable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotFound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotPayable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WrongSchema\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"schemaUID\",\"type\":\"bytes32\"}],\"name\":\"Attested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newNonce\",\"type\":\"uint256\"}],\"name\":\"NonceIncreased\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"schemaUID\",\"type\":\"bytes32\"}],\"name\":\"Revoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"RevokedOffchain\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"Timestamped\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData\",\"name\":\"data\",\"type\":\"tuple\"}],\"internalType\":\"structAttestationRequest\",\"name\":\"request\",\"type\":\"tuple\"}],\"name\":\"attest\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData\",\"name\":\"data\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature\",\"name\":\"signature\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structDelegatedAttestationRequest\",\"name\":\"delegatedRequest\",\"type\":\"tuple\"}],\"name\":\"attestByDelegation\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getAttestTypeHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"}],\"name\":\"getAttestation\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"time\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"revocationTime\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structAttestation\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getDomainSeparator\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getName\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"getNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"getRevokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getRevokeTypeHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getSchemaRegistry\",\"outputs\":[{\"internalType\":\"contractISchemaRegistry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"getTimestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newNonce\",\"type\":\"uint256\"}],\"name\":\"increaseNonce\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"}],\"name\":\"isAttestationValid\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"}],\"internalType\":\"structMultiAttestationRequest[]\",\"name\":\"multiRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiAttest\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structMultiDelegatedAttestationRequest[]\",\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiAttestByDelegation\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"}],\"internalType\":\"structMultiRevocationRequest[]\",\"name\":\"multiRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiRevoke\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structMultiDelegatedRevocationRequest[]\",\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiRevokeByDelegation\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"data\",\"type\":\"bytes32[]\"}],\"name\":\"multiRevokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"data\",\"type\":\"bytes32[]\"}],\"name\":\"multiTimestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData\",\"name\":\"data\",\"type\":\"tuple\"}],\"internalType\":\"structRevocationRequest\",\"name\":\"request\",\"type\":\"tuple\"}],\"name\":\"revoke\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData\",\"name\":\"data\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature\",\"name\":\"signature\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structDelegatedRevocationRequest\",\"name\":\"delegatedRequest\",\"type\":\"tuple\"}],\"name\":\"revokeByDelegation\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"revokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"timestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x61016060405234801561001157600080fd5b50604080518082018252600381526245415360e81b60208083019182528351808501855260058152640312e322e360dc1b908201529151812060e08190527fe374587661e69268352d25204d81b23ce801573f4b09f3545e69536dc085a37a6101008190524660a081815286517f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f818801819052818901959095526060810193909352608080840192909252308382018190528751808503909201825260c093840190975280519501949094209093529290915261012091909152516101405260805160a05160c05160e05161010051610120516101405161454d61014b600039600061073701526000612753015260006127a20152600061277d015260006126d6015260006127000152600061272a015261454d6000f3fe60806040526004361061018b5760003560e01c806395411525116100d6578063d45c44351161007f578063ed24911d11610059578063ed24911d146104fd578063f10b5cc814610512578063f17325e71461054157600080fd5b8063d45c443514610467578063e30bb5631461049e578063e71ff365146104dd57600080fd5b8063b469318d116100b0578063b469318d146103ba578063b83010d314610414578063cf190f341461044757600080fd5b80639541152514610367578063a3112a641461037a578063a6d4dbc7146103a757600080fd5b806344adc90e116101385780634d003070116101125780634d003070146102de57806354fd4d50146102fe57806379f7573a1461034757600080fd5b806344adc90e1461029857806346926267146102b85780634cb7e9e5146102cb57600080fd5b806317d7de7c1161016957806317d7de7c146102205780632d0335ab146102425780633c0427151461028557600080fd5b80630eabf6601461019057806312b11a17146101a557806313893f61146101e7575b600080fd5b6101a361019e366004613497565b610554565b005b3480156101b157600080fd5b507ff83bb2b0ede93a840239f7e701a54d9bc35f03701f51ae153d601c6947ff3d3f5b6040519081526020015b60405180910390f35b3480156101f357600080fd5b50610207610202366004613497565b6106eb565b60405167ffffffffffffffff90911681526020016101de565b34801561022c57600080fd5b50610235610730565b6040516101de9190613547565b34801561024e57600080fd5b506101d461025d36600461358c565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b6101d46102933660046135a9565b610760565b6102ab6102a6366004613497565b610863565b6040516101de91906135e4565b6101a36102c6366004613628565b6109e4565b6101a36102d9366004613497565b610a68565b3480156102ea57600080fd5b506102076102f9366004613640565b610b4b565b34801561030a57600080fd5b506102356040518060400160405280600581526020017f312e332e3000000000000000000000000000000000000000000000000000000081525081565b34801561035357600080fd5b506101a3610362366004613640565b610b58565b6102ab610375366004613497565b610bef565b34801561038657600080fd5b5061039a610395366004613640565b610e62565b6040516101de9190613740565b6101a36103b5366004613753565b611025565b3480156103c657600080fd5b506102076103d5366004613766565b73ffffffffffffffffffffffffffffffffffffffff919091166000908152603460209081526040808320938352929052205467ffffffffffffffff1690565b34801561042057600080fd5b507f2d4116d8c9824e4c316453e5c2843a1885580374159ce8768603c49085ef424c6101d4565b34801561045357600080fd5b50610207610462366004613640565b6110ca565b34801561047357600080fd5b50610207610482366004613640565b60009081526033602052604090205467ffffffffffffffff1690565b3480156104aa57600080fd5b506104cd6104b9366004613640565b600090815260326020526040902054151590565b60405190151581526020016101de565b3480156104e957600080fd5b506102076104f8366004613497565b6110d8565b34801561050957600080fd5b506101d4611110565b34801561051e57600080fd5b5060405173420000000000000000000000000000000000002081526020016101de565b6101d461054f366004613792565b61111a565b348160005b818110156106e4577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82018114600086868481811061059a5761059a6137cd565b90506020028101906105ac91906137fc565b6105b590613a92565b60208101518051919250908015806105d257508260400151518114155b15610609576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b818110156106ad576106a56040518060a001604052808660000151815260200185848151811061063e5761063e6137cd565b6020026020010151815260200186604001518481518110610661576106616137cd565b60200260200101518152602001866060015173ffffffffffffffffffffffffffffffffffffffff168152602001866080015167ffffffffffffffff168152506111d8565b60010161060c565b506106c383600001518385606001518a886113c5565b6106cd9088613bbc565b9650505050506106dd8160010190565b9050610559565b5050505050565b60004282825b818110156107245761071c3387878481811061070f5761070f6137cd565b90506020020135856119f4565b6001016106f1565b50909150505b92915050565b606061075b7f0000000000000000000000000000000000000000000000000000000000000000611af3565b905090565b600061077361076e83613cf1565b611c81565b604080516001808252818301909252600091816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90920191018161078a5790505090506107f86020840184613d6c565b61080190613da0565b81600081518110610814576108146137cd565b602090810291909101015261083d83358261083560c0870160a0880161358c565b346001611dfe565b60200151600081518110610853576108536137cd565b6020026020010151915050919050565b60608160008167ffffffffffffffff8111156108815761088161383a565b6040519080825280602002602001820160405280156108b457816020015b606081526020019060019003908161089f5790505b509050600034815b848110156109ce577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff85018114368989848181106108fc576108fc6137cd565b905060200281019061090e9190613dac565b905061091d6020820182613de0565b9050600003610958576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600061097d823561096c6020850185613de0565b61097591613e48565b338887611dfe565b805190915061098c9086613bbc565b945080602001518785815181106109a5576109a56137cd565b6020026020010181905250806020015151860195505050506109c78160010190565b90506108bc565b506109d98383612510565b979650505050505050565b604080516001808252818301909252600091816020015b60408051808201909152600080825260208201528152602001906001900390816109fb579050509050610a3636839003830160208401613ebc565b81600081518110610a4957610a496137cd565b6020908102919091010152610a63823582333460016113c5565b505050565b348160005b818110156106e4577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8201811436868684818110610aad57610aad6137cd565b9050602002810190610abf9190613dac565b9050610b2c8135610ad36020840184613ed8565b808060200260200160405190810160405280939291908181526020016000905b82821015610b1f57610b1060408302860136819003810190613ebc565b81526020019060010190610af3565b50505050503388866113c5565b610b369086613bbc565b94505050610b448160010190565b9050610a6d565b60004261072a83826125fa565b33600090815260208190526040902054808211610ba1576040517f756688fe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b336000908152602081815260409182902084905581518381529081018490527f57b09af877df9068fd60a69d7b21f5576b8b38955812d6ae4ac52942f1e38fb7910160405180910390a15050565b60608160008167ffffffffffffffff811115610c0d57610c0d61383a565b604051908082528060200260200182016040528015610c4057816020015b6060815260200190600190039081610c2b5790505b509050600034815b848110156109ce577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8501811436898984818110610c8857610c886137cd565b9050602002810190610c9a91906137fc565b9050366000610cac6020840184613de0565b909250905080801580610ccd5750610cc76040850185613f40565b90508114155b15610d04576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b81811015610de557610ddd6040518060a0016040528087600001358152602001868685818110610d3957610d396137cd565b9050602002810190610d4b9190613d6c565b610d5490613da0565b8152602001610d666040890189613f40565b85818110610d7657610d766137cd565b905060600201803603810190610d8c9190613fa7565b8152602001610da16080890160608a0161358c565b73ffffffffffffffffffffffffffffffffffffffff168152602001610dcc60a0890160808a01613fc3565b67ffffffffffffffff169052611c81565b600101610d07565b506000610e0e8535610df78587613e48565b610e076080890160608a0161358c565b8b8a611dfe565b8051909150610e1d9089613bbc565b975080602001518a8881518110610e3657610e366137cd565b602002602001018190525080602001515189019850505050505050610e5b8160010190565b9050610c48565b604080516101408101825260008082526020820181905291810182905260608082018390526080820183905260a0820183905260c0820183905260e0820183905261010082019290925261012081019190915260008281526032602090815260409182902082516101408101845281548152600182015492810192909252600281015467ffffffffffffffff808216948401949094526801000000000000000081048416606084015270010000000000000000000000000000000090049092166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff16151561010082015260068201805491929161012084019190610f9c90613fde565b80601f0160208091040260200160405190810160405280929190818152602001828054610fc890613fde565b80156110155780601f10610fea57610100808354040283529160200191611015565b820191906000526020600020905b815481529060010190602001808311610ff857829003601f168201915b5050505050815250509050919050565b61103c6110373683900383018361402b565b6111d8565b604080516001808252818301909252600091816020015b604080518082019091526000808252602082015281526020019060019003908161105357905050905061108e36839003830160208401613ebc565b816000815181106110a1576110a16137cd565b6020908102919091010152610a638235826110c260e0860160c0870161358c565b3460016113c5565b60004261072a3384836119f4565b60004282825b81811015610724576111088686838181106110fb576110fb6137cd565b90506020020135846125fa565b6001016110de565b600061075b6126bc565b604080516001808252818301909252600091829190816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816111345790505090506111a26020840184613d6c565b6111ab90613da0565b816000815181106111be576111be6137cd565b602090810291909101015261083d83358233346001611dfe565b608081015167ffffffffffffffff161580159061120c57504267ffffffffffffffff16816080015167ffffffffffffffff16105b15611243576040517f1ab7da6b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6020808201516040808401518451835184860151606088015173ffffffffffffffffffffffffffffffffffffffff166000908152968790529386208054959693959394611313947f2d4116d8c9824e4c316453e5c2843a1885580374159ce8768603c49085ef424c949392876112b883614099565b909155506080808b015160408051602081019890985287019590955260608601939093529184015260a083015267ffffffffffffffff1660c082015260e0015b604051602081830303815290604052805190602001206127f0565b905061138984606001518284602001518560400151866000015160405160200161137593929190928352602083019190915260f81b7fff0000000000000000000000000000000000000000000000000000000000000016604082015260410190565b604051602081830303815290604052612803565b6113bf576040517f8baa579f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b50505050565b6040517fa2ea7c6e0000000000000000000000000000000000000000000000000000000081526004810186905260009081907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa158015611433573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016820160405261147991908101906140d1565b80519091506114b4576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b855160008167ffffffffffffffff8111156114d1576114d161383a565b60405190808252806020026020018201604052801561157057816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816114ef5790505b50905060008267ffffffffffffffff81111561158e5761158e61383a565b6040519080825280602002602001820160405280156115b7578160200160208202803683370190505b50905060005b838110156119d65760008a82815181106115d9576115d96137cd565b6020908102919091018101518051600090815260329092526040909120805491925090611632576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8c81600101541461166f576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015473ffffffffffffffffffffffffffffffffffffffff8c81169116146116c5576040517f4ca8886700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015474010000000000000000000000000000000000000000900460ff1661171b576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002810154700100000000000000000000000000000000900467ffffffffffffffff1615611775576040517f905e710700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b426002820180547fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff811670010000000000000000000000000000000067ffffffffffffffff948516810291821793849055604080516101408101825287548152600188015460208201529386169286169290921791830191909152680100000000000000008304841660608301529091049091166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff161515610100820152600682018054839161012084019161188190613fde565b80601f01602080910402602001604051908101604052809291908181526020018280546118ad90613fde565b80156118fa5780601f106118cf576101008083540402835291602001916118fa565b820191906000526020600020905b8154815290600101906020018083116118dd57829003601f168201915b505050505081525050858481518110611915576119156137cd565b60200260200101819052508160200151848481518110611937576119376137cd565b6020026020010181815250508c8b73ffffffffffffffffffffffffffffffffffffffff1686858151811061196d5761196d6137cd565b602002602001015160c0015173ffffffffffffffffffffffffffffffffffffffff167ff930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f61585600001516040516119c491815260200190565b60405180910390a450506001016115bd565b506119e684838360018b8b6129d2565b9a9950505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff83166000908152603460209081526040808320858452918290529091205467ffffffffffffffff1615611a68576040517fec9d6eeb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008381526020829052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff861690811790915590519091859173ffffffffffffffffffffffffffffffffffffffff8816917f92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a222991a450505050565b604080516020808252818301909252606091600091906020820181803683370190505090506000805b6020811015611bbe576000858260208110611b3957611b396137cd565b1a60f81b90507fff000000000000000000000000000000000000000000000000000000000000008116600003611b6f5750611bbe565b80848481518110611b8257611b826137cd565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053505060019182019101611b1c565b5060008167ffffffffffffffff811115611bda57611bda61383a565b6040519080825280601f01601f191660200182016040528015611c04576020820181803683370190505b50905060005b82811015611c7857838181518110611c2457611c246137cd565b602001015160f81c60f81b828281518110611c4157611c416137cd565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350600101611c0a565b50949350505050565b608081015167ffffffffffffffff1615801590611cb557504267ffffffffffffffff16816080015167ffffffffffffffff16105b15611cec576040517f1ab7da6b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60208082015160408084015184518351848601518486015160608088015160808901518051908b012060a08a0151928c015173ffffffffffffffffffffffffffffffffffffffff1660009081529a8b9052978a208054999a97999798611313987ff83bb2b0ede93a840239f7e701a54d9bc35f03701f51ae153d601c6947ff3d3f989796959491928b611d7e83614099565b909155506080808f015160408051602081019c909c528b019990995273ffffffffffffffffffffffffffffffffffffffff90971660608a015267ffffffffffffffff9586169689019690965292151560a088015260c087019190915260e086015261010085015261012084019190915216610140820152610160016112f8565b60408051808201909152600081526060602082015284516040805180820190915260008152606060208201528167ffffffffffffffff811115611e4357611e4361383a565b604051908082528060200260200182016040528015611e6c578160200160208202803683370190505b5060208201526040517fa2ea7c6e000000000000000000000000000000000000000000000000000000008152600481018990526000907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa158015611ede573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0168201604052611f2491908101906140d1565b8051909150611f5f576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008367ffffffffffffffff811115611f7a57611f7a61383a565b60405190808252806020026020018201604052801561201957816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff909201910181611f985790505b50905060008467ffffffffffffffff8111156120375761203761383a565b604051908082528060200260200182016040528015612060578160200160208202803683370190505b50905060005b858110156124ef5760008b8281518110612082576120826137cd565b60200260200101519050600067ffffffffffffffff16816020015167ffffffffffffffff16141580156120cd57504267ffffffffffffffff16816020015167ffffffffffffffff1611155b15612104576040517f08e8b93700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8460400151158015612117575080604001515b1561214e576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006040518061014001604052806000801b81526020018f81526020016121724290565b67ffffffffffffffff168152602001836020015167ffffffffffffffff168152602001600067ffffffffffffffff16815260200183606001518152602001836000015173ffffffffffffffffffffffffffffffffffffffff1681526020018d73ffffffffffffffffffffffffffffffffffffffff16815260200183604001511515815260200183608001518152509050600080600090505b6122148382612dc3565b600081815260326020526040902054909250156122335760010161220a565b81835260008281526032602090815260409182902085518155908501516001820155908401516002820180546060870151608088015167ffffffffffffffff908116700100000000000000000000000000000000027fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff92821668010000000000000000027fffffffffffffffffffffffffffffffff000000000000000000000000000000009094169190951617919091171691909117905560a0840151600382015560c084015160048201805473ffffffffffffffffffffffffffffffffffffffff9283167fffffffffffffffffffffffff000000000000000000000000000000000000000090911617905560e0850151600583018054610100880151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff000000000000000000000000000000000000000000909116929093169190911791909117905561012084015184919060068201906123b390826141f7565b50505060608401511561240a57606084015160009081526032602052604090205461240a576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8287868151811061241d5761241d6137cd565b60200260200101819052508360a0015186868151811061243f5761243f6137cd565b6020026020010181815250508189602001518681518110612462576124626137cd565b6020026020010181815250508f8e73ffffffffffffffffffffffffffffffffffffffff16856000015173ffffffffffffffffffffffffffffffffffffffff167f8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b35856040516124d291815260200190565b60405180910390a4505050506124e88160010190565b9050612066565b506124ff83838360008c8c6129d2565b845250919998505050505050505050565b606060008267ffffffffffffffff81111561252d5761252d61383a565b604051908082528060200260200182016040528015612556578160200160208202803683370190505b508451909150600090815b818110156125ef57600087828151811061257d5761257d6137cd565b6020026020010151905060008151905060005b818110156125db578281815181106125aa576125aa6137cd565b60200260200101518787815181106125c4576125c46137cd565b602090810291909101015260019586019501612590565b5050506125e88160010190565b9050612561565b509195945050505050565b60008281526033602052604090205467ffffffffffffffff161561264a576040517f2e26794600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008281526033602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff85169081179091559051909184917f5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f9190a35050565b60003073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561272257507f000000000000000000000000000000000000000000000000000000000000000046145b1561274c57507f000000000000000000000000000000000000000000000000000000000000000090565b50604080517f00000000000000000000000000000000000000000000000000000000000000006020808301919091527f0000000000000000000000000000000000000000000000000000000000000000828401527f000000000000000000000000000000000000000000000000000000000000000060608301524660808301523060a0808401919091528351808403909101815260c0909201909252805191012090565b600061072a6127fd6126bc565b83612e22565b60008060006128128585612e64565b9092509050600081600481111561282b5761282b614311565b14801561286357508573ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16145b15612873576001925050506129cb565b6000808773ffffffffffffffffffffffffffffffffffffffff16631626ba7e60e01b88886040516024016128a8929190614340565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009094169390931790925290516129319190614361565b600060405180830381855afa9150503d806000811461296c576040519150601f19603f3d011682016040523d82523d6000602084013e612971565b606091505b5091509150818015612984575080516020145b80156129c4575080517f1626ba7e00000000000000000000000000000000000000000000000000000000906129c29083016020908101908401614373565b145b9450505050505b9392505050565b84516000906001819003612a2a57612a2288886000815181106129f7576129f76137cd565b602002602001015188600081518110612a1257612a126137cd565b6020026020010151888888612ea9565b915050612db9565b602088015173ffffffffffffffffffffffffffffffffffffffff8116612acb5760005b82811015612ab057878181518110612a6757612a676137cd565b6020026020010151600014612aa8576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600101612a4d565b508315612ac057612ac0856131c8565b600092505050612db9565b6000808273ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612b19573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612b3d919061438c565b905060005b84811015612bfa5760008a8281518110612b5e57612b5e6137cd565b6020026020010151905080600003612b765750612bf2565b82612bad576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b88811115612be7576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b978890039792909201915b600101612b42565b508715612cd5576040517f88e5b2d900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8416906388e5b2d9908490612c57908e908e906004016143a9565b60206040518083038185885af1158015612c75573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612c9a919061438c565b612cd0576040517fbf2f3a8b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b612da4565b6040517f91db0b7e00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8416906391db0b7e908490612d2b908e908e906004016143a9565b60206040518083038185885af1158015612d49573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612d6e919061438c565b612da4576040517fe8bee83900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8515612db357612db3876131c8565b50925050505b9695505050505050565b60208083015160c084015160e0850151604080870151606088015161010089015160a08a01516101208b01519451600099612e0499989796918c9101614462565b60405160208183030381529060405280519060200120905092915050565b6040517f190100000000000000000000000000000000000000000000000000000000000060208201526022810183905260428101829052600090606201612e04565b6000808251604103612e9a5760208301516040840151606085015160001a612e8e878285856131db565b94509450505050612ea2565b506000905060025b9250929050565b602086015160009073ffffffffffffffffffffffffffffffffffffffff8116612f1d578515612f04576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8215612f1357612f13846131c8565b6000915050612db9565b8515613008578073ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612f6e573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612f92919061438c565b612fc8576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b83861115613002576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b85840393505b84156130e0576040517fe49617e100000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e49617e1908890613062908b90600401613740565b60206040518083038185885af1158015613080573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906130a5919061438c565b6130db576040517fccf3bb2700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6131ad565b6040517fe60c350500000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e60c3505908890613134908b90600401613740565b60206040518083038185885af1158015613152573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190613177919061438c565b6131ad576040517fbd8ba84d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82156131bc576131bc846131c8565b50939695505050505050565b80156131d8576131d833826132f3565b50565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a083111561321257506000905060036132ea565b8460ff16601b1415801561322a57508460ff16601c14155b1561323b57506000905060046132ea565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa15801561328f573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff81166132e3576000600192509250506132ea565b9150600090505b94509492505050565b80471015613362576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e636500000060448201526064015b60405180910390fd5b60008273ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d80600081146133bc576040519150601f19603f3d011682016040523d82523d6000602084013e6133c1565b606091505b5050905080610a63576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d617920686176652072657665727465640000000000006064820152608401613359565b60008083601f84011261346457600080fd5b50813567ffffffffffffffff81111561347c57600080fd5b6020830191508360208260051b8501011115612ea257600080fd5b600080602083850312156134aa57600080fd5b823567ffffffffffffffff8111156134c157600080fd5b6134cd85828601613452565b90969095509350505050565b60005b838110156134f45781810151838201526020016134dc565b50506000910152565b600081518084526135158160208601602086016134d9565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006129cb60208301846134fd565b73ffffffffffffffffffffffffffffffffffffffff811681146131d857600080fd5b80356135878161355a565b919050565b60006020828403121561359e57600080fd5b81356129cb8161355a565b6000602082840312156135bb57600080fd5b813567ffffffffffffffff8111156135d257600080fd5b820160e081850312156129cb57600080fd5b6020808252825182820181905260009190848201906040850190845b8181101561361c57835183529284019291840191600101613600565b50909695505050505050565b60006060828403121561363a57600080fd5b50919050565b60006020828403121561365257600080fd5b5035919050565b600061014082518452602083015160208501526040830151613687604086018267ffffffffffffffff169052565b5060608301516136a3606086018267ffffffffffffffff169052565b5060808301516136bf608086018267ffffffffffffffff169052565b5060a083015160a085015260c08301516136f160c086018273ffffffffffffffffffffffffffffffffffffffff169052565b5060e083015161371960e086018273ffffffffffffffffffffffffffffffffffffffff169052565b506101008381015115159085015261012080840151818601839052612db9838701826134fd565b6020815260006129cb6020830184613659565b6000610100828403121561363a57600080fd5b6000806040838503121561377957600080fd5b82356137848161355a565b946020939093013593505050565b6000602082840312156137a457600080fd5b813567ffffffffffffffff8111156137bb57600080fd5b8201604081850312156129cb57600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff6183360301811261383057600080fd5b9190910192915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405160a0810167ffffffffffffffff8111828210171561388c5761388c61383a565b60405290565b60405160c0810167ffffffffffffffff8111828210171561388c5761388c61383a565b6040516080810167ffffffffffffffff8111828210171561388c5761388c61383a565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff8111828210171561391f5761391f61383a565b604052919050565b600067ffffffffffffffff8211156139415761394161383a565b5060051b60200190565b60006040828403121561395d57600080fd5b6040516040810181811067ffffffffffffffff821117156139805761398061383a565b604052823581526020928301359281019290925250919050565b6000606082840312156139ac57600080fd5b6040516060810181811067ffffffffffffffff821117156139cf576139cf61383a565b604052905080823560ff811681146139e657600080fd5b8082525060208301356020820152604083013560408201525092915050565b600082601f830112613a1657600080fd5b81356020613a2b613a2683613927565b6138d8565b82815260609283028501820192828201919087851115613a4a57600080fd5b8387015b85811015613a6d57613a60898261399a565b8452928401928101613a4e565b5090979650505050505050565b803567ffffffffffffffff8116811461358757600080fd5b600060a08236031215613aa457600080fd5b613aac613869565b8235815260208084013567ffffffffffffffff80821115613acc57600080fd5b9085019036601f830112613adf57600080fd5b8135613aed613a2682613927565b81815260069190911b83018401908481019036831115613b0c57600080fd5b938501935b82851015613b3557613b23368661394b565b82528582019150604085019450613b11565b80868801525050506040860135925080831115613b5157600080fd5b5050613b5f36828601613a05565b604083015250613b716060840161357c565b6060820152613b8260808401613a7a565b608082015292915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8181038181111561072a5761072a613b8d565b80151581146131d857600080fd5b600067ffffffffffffffff821115613bf757613bf761383a565b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01660200190565b600060c08284031215613c3557600080fd5b613c3d613892565b90508135613c4a8161355a565b81526020613c59838201613a7a565b818301526040830135613c6b81613bcf565b604083015260608381013590830152608083013567ffffffffffffffff811115613c9457600080fd5b8301601f81018513613ca557600080fd5b8035613cb3613a2682613bdd565b8181528684838501011115613cc757600080fd5b818484018583013760008483830101528060808601525050505060a082013560a082015292915050565b600060e08236031215613d0357600080fd5b613d0b613869565b82358152602083013567ffffffffffffffff811115613d2957600080fd5b613d3536828601613c23565b602083015250613d48366040850161399a565b604082015260a0830135613d5b8161355a565b6060820152613b8260c08401613a7a565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff4183360301811261383057600080fd5b600061072a3683613c23565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc183360301811261383057600080fd5b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613e1557600080fd5b83018035915067ffffffffffffffff821115613e3057600080fd5b6020019150600581901b3603821315612ea257600080fd5b6000613e56613a2684613927565b80848252602080830192508560051b850136811115613e7457600080fd5b855b81811015613eb057803567ffffffffffffffff811115613e965760008081fd5b613ea236828a01613c23565b865250938201938201613e76565b50919695505050505050565b600060408284031215613ece57600080fd5b6129cb838361394b565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613f0d57600080fd5b83018035915067ffffffffffffffff821115613f2857600080fd5b6020019150600681901b3603821315612ea257600080fd5b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613f7557600080fd5b83018035915067ffffffffffffffff821115613f9057600080fd5b6020019150606081023603821315612ea257600080fd5b600060608284031215613fb957600080fd5b6129cb838361399a565b600060208284031215613fd557600080fd5b6129cb82613a7a565b600181811c90821680613ff257607f821691505b60208210810361363a577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000610100828403121561403e57600080fd5b614046613869565b82358152614057846020850161394b565b6020820152614069846060850161399a565b604082015260c083013561407c8161355a565b606082015261408d60e08401613a7a565b60808201529392505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036140ca576140ca613b8d565b5060010190565b600060208083850312156140e457600080fd5b825167ffffffffffffffff808211156140fc57600080fd5b908401906080828703121561411057600080fd5b6141186138b5565b82518152838301516141298161355a565b81850152604083015161413b81613bcf565b604082015260608301518281111561415257600080fd5b80840193505086601f84011261416757600080fd5b82519150614177613a2683613bdd565b828152878584860101111561418b57600080fd5b61419a838683018787016134d9565b60608201529695505050505050565b601f821115610a6357600081815260208120601f850160051c810160208610156141d05750805b601f850160051c820191505b818110156141ef578281556001016141dc565b505050505050565b815167ffffffffffffffff8111156142115761421161383a565b6142258161421f8454613fde565b846141a9565b602080601f83116001811461427857600084156142425750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b1785556141ef565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b828110156142c5578886015182559484019460019091019084016142a6565b508582101561430157878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b82815260406020820152600061435960408301846134fd565b949350505050565b600082516138308184602087016134d9565b60006020828403121561438557600080fd5b5051919050565b60006020828403121561439e57600080fd5b81516129cb81613bcf565b6000604082016040835280855180835260608501915060608160051b8601019250602080880160005b8381101561441e577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa088870301855261440c868351613659565b955093820193908201906001016143d2565b50508584038187015286518085528782019482019350915060005b8281101561445557845184529381019392810192600101614439565b5091979650505050505050565b89815260007fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808b60601b166020840152808a60601b166034840152507fffffffffffffffff000000000000000000000000000000000000000000000000808960c01b166048840152808860c01b1660508401525085151560f81b605883015284605983015283516144fb8160798501602088016134d9565b80830190507fffffffff000000000000000000000000000000000000000000000000000000008460e01b166079820152607d81019150509a995050505050505050505056fea164736f6c6343000813000a",
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
	parsed, err := EASMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
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
