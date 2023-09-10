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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AccessDenied\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyRevoked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyRevokedOffchain\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyTimestamped\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DeadlineExpired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InsufficientValue\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAttestation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAttestations\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidExpirationTime\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidLength\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidNonce\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidOffset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRegistry\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRevocation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRevocations\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSchema\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignature\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidVerifier\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Irrevocable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotFound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotPayable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WrongSchema\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"schemaUID\",\"type\":\"bytes32\"}],\"name\":\"Attested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newNonce\",\"type\":\"uint256\"}],\"name\":\"NonceIncreased\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"schemaUID\",\"type\":\"bytes32\"}],\"name\":\"Revoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"RevokedOffchain\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"Timestamped\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData\",\"name\":\"data\",\"type\":\"tuple\"}],\"internalType\":\"structAttestationRequest\",\"name\":\"request\",\"type\":\"tuple\"}],\"name\":\"attest\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData\",\"name\":\"data\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature\",\"name\":\"signature\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structDelegatedAttestationRequest\",\"name\":\"delegatedRequest\",\"type\":\"tuple\"}],\"name\":\"attestByDelegation\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getAttestTypeHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"}],\"name\":\"getAttestation\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"time\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"revocationTime\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structAttestation\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getDomainSeparator\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getName\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"getNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"getRevokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getRevokeTypeHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getSchemaRegistry\",\"outputs\":[{\"internalType\":\"contractISchemaRegistry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"getTimestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newNonce\",\"type\":\"uint256\"}],\"name\":\"increaseNonce\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"}],\"name\":\"isAttestationValid\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"}],\"internalType\":\"structMultiAttestationRequest[]\",\"name\":\"multiRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiAttest\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structMultiDelegatedAttestationRequest[]\",\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiAttestByDelegation\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"}],\"internalType\":\"structMultiRevocationRequest[]\",\"name\":\"multiRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiRevoke\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structMultiDelegatedRevocationRequest[]\",\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiRevokeByDelegation\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"data\",\"type\":\"bytes32[]\"}],\"name\":\"multiRevokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"data\",\"type\":\"bytes32[]\"}],\"name\":\"multiTimestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData\",\"name\":\"data\",\"type\":\"tuple\"}],\"internalType\":\"structRevocationRequest\",\"name\":\"request\",\"type\":\"tuple\"}],\"name\":\"revoke\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData\",\"name\":\"data\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structSignature\",\"name\":\"signature\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"deadline\",\"type\":\"uint64\"}],\"internalType\":\"structDelegatedRevocationRequest\",\"name\":\"delegatedRequest\",\"type\":\"tuple\"}],\"name\":\"revokeByDelegation\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"revokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"timestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101c06040523480156200001257600080fd5b50604080518082018252600381526245415360e81b6020808301918252835180850190945260058452640312e322e360dc1b9084019081526001608052600260a052600060c052825190912083519091206101408290526101608190524661010052919291839183917f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f620000ec8184846040805160208101859052908101839052606081018290524660808201523060a082015260009060c0016040516020818303038152906040528051906020012090509392505050565b60e0523061012052610180525050505060208201516101a052506200010e9050565b60805160a05160c05160e05161010051610120516101405161016051610180516101a0516147d16200018a6000396000610703015260006128ff0152600061294e0152600061292901526000612882015260006128ac015260006128d601526000610b7d01526000610b5401526000610b2b01526147d16000f3fe60806040526004361061018b5760003560e01c806395411525116100d6578063d45c44351161007f578063ed24911d11610059578063ed24911d146104c9578063f10b5cc8146104de578063f17325e71461050d57600080fd5b8063d45c443514610433578063e30bb5631461046a578063e71ff365146104a957600080fd5b8063b469318d116100b0578063b469318d14610386578063b83010d3146103e0578063cf190f341461041357600080fd5b80639541152514610333578063a3112a6414610346578063a6d4dbc71461037357600080fd5b806344adc90e116101385780634d003070116101125780634d003070146102de57806354fd4d50146102fe57806379f7573a1461031357600080fd5b806344adc90e1461029857806346926267146102b85780634cb7e9e5146102cb57600080fd5b806317d7de7c1161016957806317d7de7c146102205780632d0335ab146102425780633c0427151461028557600080fd5b80630eabf6601461019057806312b11a17146101a557806313893f61146101e7575b600080fd5b6101a361019e366004613643565b610520565b005b3480156101b157600080fd5b507ff83bb2b0ede93a840239f7e701a54d9bc35f03701f51ae153d601c6947ff3d3f5b6040519081526020015b60405180910390f35b3480156101f357600080fd5b50610207610202366004613643565b6106b7565b60405167ffffffffffffffff90911681526020016101de565b34801561022c57600080fd5b506102356106fc565b6040516101de91906136f3565b34801561024e57600080fd5b506101d461025d366004613738565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b6101d4610293366004613755565b61072c565b6102ab6102a6366004613643565b61082f565b6040516101de9190613790565b6101a36102c63660046137d4565b6109b0565b6101a36102d9366004613643565b610a34565b3480156102ea57600080fd5b506102076102f93660046137ec565b610b17565b34801561030a57600080fd5b50610235610b24565b34801561031f57600080fd5b506101a361032e3660046137ec565b610bc7565b6102ab610341366004613643565b610c5e565b34801561035257600080fd5b506103666103613660046137ec565b610ed1565b6040516101de91906138ec565b6101a36103813660046138ff565b611094565b34801561039257600080fd5b506102076103a1366004613912565b73ffffffffffffffffffffffffffffffffffffffff919091166000908152603460209081526040808320938352929052205467ffffffffffffffff1690565b3480156103ec57600080fd5b507f2d4116d8c9824e4c316453e5c2843a1885580374159ce8768603c49085ef424c6101d4565b34801561041f57600080fd5b5061020761042e3660046137ec565b611139565b34801561043f57600080fd5b5061020761044e3660046137ec565b60009081526033602052604090205467ffffffffffffffff1690565b34801561047657600080fd5b506104996104853660046137ec565b600090815260326020526040902054151590565b60405190151581526020016101de565b3480156104b557600080fd5b506102076104c4366004613643565b611147565b3480156104d557600080fd5b506101d461117f565b3480156104ea57600080fd5b5060405173420000000000000000000000000000000000002081526020016101de565b6101d461051b36600461393e565b611189565b348160005b818110156106b0577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82018114600086868481811061056657610566613979565b905060200281019061057891906139a8565b61058190613c3e565b602081015180519192509080158061059e57508260400151518114155b156105d5576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b81811015610679576106716040518060a001604052808660000151815260200185848151811061060a5761060a613979565b602002602001015181526020018660400151848151811061062d5761062d613979565b60200260200101518152602001866060015173ffffffffffffffffffffffffffffffffffffffff168152602001866080015167ffffffffffffffff16815250611247565b6001016105d8565b5061068f83600001518385606001518a88611434565b6106999088613d68565b9650505050506106a98160010190565b9050610525565b5050505050565b60004282825b818110156106f0576106e8338787848181106106db576106db613979565b9050602002013585611a63565b6001016106bd565b50909150505b92915050565b60606107277f0000000000000000000000000000000000000000000000000000000000000000611b62565b905090565b600061073f61073a83613e9d565b611cf0565b604080516001808252818301909252600091816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816107565790505090506107c46020840184613f18565b6107cd90613f4c565b816000815181106107e0576107e0613979565b602090810291909101015261080983358261080160c0870160a08801613738565b346001611e6d565b6020015160008151811061081f5761081f613979565b6020026020010151915050919050565b60608160008167ffffffffffffffff81111561084d5761084d6139e6565b60405190808252806020026020018201604052801561088057816020015b606081526020019060019003908161086b5790505b509050600034815b8481101561099a577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff85018114368989848181106108c8576108c8613979565b90506020028101906108da9190613f58565b90506108e96020820182613f8c565b9050600003610924576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600061094982356109386020850185613f8c565b61094191613ff4565b338887611e6d565b80519091506109589086613d68565b9450806020015187858151811061097157610971613979565b6020026020010181905250806020015151860195505050506109938160010190565b9050610888565b506109a5838361257f565b979650505050505050565b604080516001808252818301909252600091816020015b60408051808201909152600080825260208201528152602001906001900390816109c7579050509050610a0236839003830160208401614068565b81600081518110610a1557610a15613979565b6020908102919091010152610a2f82358233346001611434565b505050565b348160005b818110156106b0577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8201811436868684818110610a7957610a79613979565b9050602002810190610a8b9190613f58565b9050610af88135610a9f6020840184614084565b808060200260200160405190810160405280939291908181526020016000905b82821015610aeb57610adc60408302860136819003810190614068565b81526020019060010190610abf565b5050505050338886611434565b610b029086613d68565b94505050610b108160010190565b9050610a39565b6000426106f68382612669565b6060610b4f7f000000000000000000000000000000000000000000000000000000000000000061272b565b610b787f000000000000000000000000000000000000000000000000000000000000000061272b565b610ba17f000000000000000000000000000000000000000000000000000000000000000061272b565b604051602001610bb3939291906140ec565b604051602081830303815290604052905090565b33600090815260208190526040902054808211610c10576040517f756688fe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b336000908152602081815260409182902084905581518381529081018490527f57b09af877df9068fd60a69d7b21f5576b8b38955812d6ae4ac52942f1e38fb7910160405180910390a15050565b60608160008167ffffffffffffffff811115610c7c57610c7c6139e6565b604051908082528060200260200182016040528015610caf57816020015b6060815260200190600190039081610c9a5790505b509050600034815b8481101561099a577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8501811436898984818110610cf757610cf7613979565b9050602002810190610d0991906139a8565b9050366000610d1b6020840184613f8c565b909250905080801580610d3c5750610d366040850185614162565b90508114155b15610d73576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b81811015610e5457610e4c6040518060a0016040528087600001358152602001868685818110610da857610da8613979565b9050602002810190610dba9190613f18565b610dc390613f4c565b8152602001610dd56040890189614162565b85818110610de557610de5613979565b905060600201803603810190610dfb91906141c9565b8152602001610e106080890160608a01613738565b73ffffffffffffffffffffffffffffffffffffffff168152602001610e3b60a0890160808a016141e5565b67ffffffffffffffff169052611cf0565b600101610d76565b506000610e7d8535610e668587613ff4565b610e766080890160608a01613738565b8b8a611e6d565b8051909150610e8c9089613d68565b975080602001518a8881518110610ea557610ea5613979565b602002602001018190525080602001515189019850505050505050610eca8160010190565b9050610cb7565b604080516101408101825260008082526020820181905291810182905260608082018390526080820183905260a0820183905260c0820183905260e0820183905261010082019290925261012081019190915260008281526032602090815260409182902082516101408101845281548152600182015492810192909252600281015467ffffffffffffffff808216948401949094526801000000000000000081048416606084015270010000000000000000000000000000000090049092166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff1615156101008201526006820180549192916101208401919061100b90614200565b80601f016020809104026020016040519081016040528092919081815260200182805461103790614200565b80156110845780601f1061105957610100808354040283529160200191611084565b820191906000526020600020905b81548152906001019060200180831161106757829003601f168201915b5050505050815250509050919050565b6110ab6110a63683900383018361424d565b611247565b604080516001808252818301909252600091816020015b60408051808201909152600080825260208201528152602001906001900390816110c25790505090506110fd36839003830160208401614068565b8160008151811061111057611110613979565b6020908102919091010152610a2f82358261113160e0860160c08701613738565b346001611434565b6000426106f6338483611a63565b60004282825b818110156106f05761117786868381811061116a5761116a613979565b9050602002013584612669565b60010161114d565b6000610727612868565b604080516001808252818301909252600091829190816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816111a35790505090506112116020840184613f18565b61121a90613f4c565b8160008151811061122d5761122d613979565b602090810291909101015261080983358233346001611e6d565b608081015167ffffffffffffffff161580159061127b57504267ffffffffffffffff16816080015167ffffffffffffffff16105b156112b2576040517f1ab7da6b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6020808201516040808401518451835184860151606088015173ffffffffffffffffffffffffffffffffffffffff166000908152968790529386208054959693959394611382947f2d4116d8c9824e4c316453e5c2843a1885580374159ce8768603c49085ef424c94939287611327836142bb565b909155506080808b015160408051602081019890985287019590955260608601939093529184015260a083015267ffffffffffffffff1660c082015260e0015b6040516020818303038152906040528051906020012061299c565b90506113f88460600151828460200151856040015186600001516040516020016113e493929190928352602083019190915260f81b7fff0000000000000000000000000000000000000000000000000000000000000016604082015260410190565b6040516020818303038152906040526129af565b61142e576040517f8baa579f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b50505050565b6040517fa2ea7c6e0000000000000000000000000000000000000000000000000000000081526004810186905260009081907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa1580156114a2573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01682016040526114e891908101906142f3565b8051909150611523576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b855160008167ffffffffffffffff811115611540576115406139e6565b6040519080825280602002602001820160405280156115df57816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90920191018161155e5790505b50905060008267ffffffffffffffff8111156115fd576115fd6139e6565b604051908082528060200260200182016040528015611626578160200160208202803683370190505b50905060005b83811015611a455760008a828151811061164857611648613979565b60209081029190910181015180516000908152603290925260409091208054919250906116a1576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8c8160010154146116de576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015473ffffffffffffffffffffffffffffffffffffffff8c8116911614611734576040517f4ca8886700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015474010000000000000000000000000000000000000000900460ff1661178a576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002810154700100000000000000000000000000000000900467ffffffffffffffff16156117e4576040517f905e710700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b426002820180547fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff811670010000000000000000000000000000000067ffffffffffffffff948516810291821793849055604080516101408101825287548152600188015460208201529386169286169290921791830191909152680100000000000000008304841660608301529091049091166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff16151561010082015260068201805483916101208401916118f090614200565b80601f016020809104026020016040519081016040528092919081815260200182805461191c90614200565b80156119695780601f1061193e57610100808354040283529160200191611969565b820191906000526020600020905b81548152906001019060200180831161194c57829003601f168201915b50505050508152505085848151811061198457611984613979565b602002602001018190525081602001518484815181106119a6576119a6613979565b6020026020010181815250508c8b73ffffffffffffffffffffffffffffffffffffffff168685815181106119dc576119dc613979565b602002602001015160c0015173ffffffffffffffffffffffffffffffffffffffff167ff930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f6158560000151604051611a3391815260200190565b60405180910390a4505060010161162c565b50611a5584838360018b8b612b7e565b9a9950505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff83166000908152603460209081526040808320858452918290529091205467ffffffffffffffff1615611ad7576040517fec9d6eeb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008381526020829052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff861690811790915590519091859173ffffffffffffffffffffffffffffffffffffffff8816917f92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a222991a450505050565b604080516020808252818301909252606091600091906020820181803683370190505090506000805b6020811015611c2d576000858260208110611ba857611ba8613979565b1a60f81b90507fff000000000000000000000000000000000000000000000000000000000000008116600003611bde5750611c2d565b80848481518110611bf157611bf1613979565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053505060019182019101611b8b565b5060008167ffffffffffffffff811115611c4957611c496139e6565b6040519080825280601f01601f191660200182016040528015611c73576020820181803683370190505b50905060005b82811015611ce757838181518110611c9357611c93613979565b602001015160f81c60f81b828281518110611cb057611cb0613979565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350600101611c79565b50949350505050565b608081015167ffffffffffffffff1615801590611d2457504267ffffffffffffffff16816080015167ffffffffffffffff16105b15611d5b576040517f1ab7da6b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60208082015160408084015184518351848601518486015160608088015160808901518051908b012060a08a0151928c015173ffffffffffffffffffffffffffffffffffffffff1660009081529a8b9052978a208054999a97999798611382987ff83bb2b0ede93a840239f7e701a54d9bc35f03701f51ae153d601c6947ff3d3f989796959491928b611ded836142bb565b909155506080808f015160408051602081019c909c528b019990995273ffffffffffffffffffffffffffffffffffffffff90971660608a015267ffffffffffffffff9586169689019690965292151560a088015260c087019190915260e08601526101008501526101208401919091521661014082015261016001611367565b60408051808201909152600081526060602082015284516040805180820190915260008152606060208201528167ffffffffffffffff811115611eb257611eb26139e6565b604051908082528060200260200182016040528015611edb578160200160208202803683370190505b5060208201526040517fa2ea7c6e000000000000000000000000000000000000000000000000000000008152600481018990526000907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa158015611f4d573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0168201604052611f9391908101906142f3565b8051909150611fce576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008367ffffffffffffffff811115611fe957611fe96139e6565b60405190808252806020026020018201604052801561208857816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816120075790505b50905060008467ffffffffffffffff8111156120a6576120a66139e6565b6040519080825280602002602001820160405280156120cf578160200160208202803683370190505b50905060005b8581101561255e5760008b82815181106120f1576120f1613979565b60200260200101519050600067ffffffffffffffff16816020015167ffffffffffffffff161415801561213c57504267ffffffffffffffff16816020015167ffffffffffffffff1611155b15612173576040517f08e8b93700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8460400151158015612186575080604001515b156121bd576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006040518061014001604052806000801b81526020018f81526020016121e14290565b67ffffffffffffffff168152602001836020015167ffffffffffffffff168152602001600067ffffffffffffffff16815260200183606001518152602001836000015173ffffffffffffffffffffffffffffffffffffffff1681526020018d73ffffffffffffffffffffffffffffffffffffffff16815260200183604001511515815260200183608001518152509050600080600090505b6122838382612f6f565b600081815260326020526040902054909250156122a257600101612279565b81835260008281526032602090815260409182902085518155908501516001820155908401516002820180546060870151608088015167ffffffffffffffff908116700100000000000000000000000000000000027fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff92821668010000000000000000027fffffffffffffffffffffffffffffffff000000000000000000000000000000009094169190951617919091171691909117905560a0840151600382015560c084015160048201805473ffffffffffffffffffffffffffffffffffffffff9283167fffffffffffffffffffffffff000000000000000000000000000000000000000090911617905560e0850151600583018054610100880151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff000000000000000000000000000000000000000000909116929093169190911791909117905561012084015184919060068201906124229082614419565b505050606084015115612479576060840151600090815260326020526040902054612479576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8287868151811061248c5761248c613979565b60200260200101819052508360a001518686815181106124ae576124ae613979565b60200260200101818152505081896020015186815181106124d1576124d1613979565b6020026020010181815250508f8e73ffffffffffffffffffffffffffffffffffffffff16856000015173ffffffffffffffffffffffffffffffffffffffff167f8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b358560405161254191815260200190565b60405180910390a4505050506125578160010190565b90506120d5565b5061256e83838360008c8c612b7e565b845250919998505050505050505050565b606060008267ffffffffffffffff81111561259c5761259c6139e6565b6040519080825280602002602001820160405280156125c5578160200160208202803683370190505b508451909150600090815b8181101561265e5760008782815181106125ec576125ec613979565b6020026020010151905060008151905060005b8181101561264a5782818151811061261957612619613979565b602002602001015187878151811061263357612633613979565b6020908102919091010152600195860195016125ff565b5050506126578160010190565b90506125d0565b509195945050505050565b60008281526033602052604090205467ffffffffffffffff16156126b9576040517f2e26794600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008281526033602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff85169081179091559051909184917f5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f9190a35050565b60608160000361276e57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156127985780612782816142bb565b91506127919050600a83614562565b9150612772565b60008167ffffffffffffffff8111156127b3576127b36139e6565b6040519080825280601f01601f1916602001820160405280156127dd576020820181803683370190505b5090505b8415612860576127f2600183613d68565b91506127ff600a86614576565b61280a90603061458a565b60f81b81838151811061281f5761281f613979565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350612859600a86614562565b94506127e1565b949350505050565b60003073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000161480156128ce57507f000000000000000000000000000000000000000000000000000000000000000046145b156128f857507f000000000000000000000000000000000000000000000000000000000000000090565b50604080517f00000000000000000000000000000000000000000000000000000000000000006020808301919091527f0000000000000000000000000000000000000000000000000000000000000000828401527f000000000000000000000000000000000000000000000000000000000000000060608301524660808301523060a0808401919091528351808403909101815260c0909201909252805191012090565b60006106f66129a9612868565b83612fce565b60008060006129be8585613010565b909250905060008160048111156129d7576129d761459d565b148015612a0f57508573ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16145b15612a1f57600192505050612b77565b6000808773ffffffffffffffffffffffffffffffffffffffff16631626ba7e60e01b8888604051602401612a549291906145cc565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff00000000000000000000000000000000000000000000000000000000909416939093179092529051612add91906145e5565b600060405180830381855afa9150503d8060008114612b18576040519150601f19603f3d011682016040523d82523d6000602084013e612b1d565b606091505b5091509150818015612b30575080516020145b8015612b70575080517f1626ba7e0000000000000000000000000000000000000000000000000000000090612b6e90830160209081019084016145f7565b145b9450505050505b9392505050565b84516000906001819003612bd657612bce8888600081518110612ba357612ba3613979565b602002602001015188600081518110612bbe57612bbe613979565b6020026020010151888888613055565b915050612f65565b602088015173ffffffffffffffffffffffffffffffffffffffff8116612c775760005b82811015612c5c57878181518110612c1357612c13613979565b6020026020010151600014612c54576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600101612bf9565b508315612c6c57612c6c85613374565b600092505050612f65565b6000808273ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612cc5573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612ce99190614610565b905060005b84811015612da65760008a8281518110612d0a57612d0a613979565b6020026020010151905080600003612d225750612d9e565b82612d59576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b88811115612d93576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b978890039792909201915b600101612cee565b508715612e81576040517f88e5b2d900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8416906388e5b2d9908490612e03908e908e9060040161462d565b60206040518083038185885af1158015612e21573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612e469190614610565b612e7c576040517fbf2f3a8b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b612f50565b6040517f91db0b7e00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8416906391db0b7e908490612ed7908e908e9060040161462d565b60206040518083038185885af1158015612ef5573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612f1a9190614610565b612f50576040517fe8bee83900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8515612f5f57612f5f87613374565b50925050505b9695505050505050565b60208083015160c084015160e0850151604080870151606088015161010089015160a08a01516101208b01519451600099612fb099989796918c91016146e6565b60405160208183030381529060405280519060200120905092915050565b6040517f190100000000000000000000000000000000000000000000000000000000000060208201526022810183905260428101829052600090606201612fb0565b60008082516041036130465760208301516040840151606085015160001a61303a87828585613387565b9450945050505061304e565b506000905060025b9250929050565b602086015160009073ffffffffffffffffffffffffffffffffffffffff81166130c95785156130b0576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82156130bf576130bf84613374565b6000915050612f65565b85156131b4578073ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa15801561311a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061313e9190614610565b613174576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b838611156131ae576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b85840393505b841561328c576040517fe49617e100000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e49617e190889061320e908b906004016138ec565b60206040518083038185885af115801561322c573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906132519190614610565b613287576040517fccf3bb2700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b613359565b6040517fe60c350500000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e60c35059088906132e0908b906004016138ec565b60206040518083038185885af11580156132fe573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906133239190614610565b613359576040517fbd8ba84d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82156133685761336884613374565b50939695505050505050565b801561338457613384338261349f565b50565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08311156133be5750600090506003613496565b8460ff16601b141580156133d657508460ff16601c14155b156133e75750600090506004613496565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa15801561343b573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff811661348f57600060019250925050613496565b9150600090505b94509492505050565b8047101561350e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e636500000060448201526064015b60405180910390fd5b60008273ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d8060008114613568576040519150601f19603f3d011682016040523d82523d6000602084013e61356d565b606091505b5050905080610a2f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d617920686176652072657665727465640000000000006064820152608401613505565b60008083601f84011261361057600080fd5b50813567ffffffffffffffff81111561362857600080fd5b6020830191508360208260051b850101111561304e57600080fd5b6000806020838503121561365657600080fd5b823567ffffffffffffffff81111561366d57600080fd5b613679858286016135fe565b90969095509350505050565b60005b838110156136a0578181015183820152602001613688565b50506000910152565b600081518084526136c1816020860160208601613685565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000612b7760208301846136a9565b73ffffffffffffffffffffffffffffffffffffffff8116811461338457600080fd5b803561373381613706565b919050565b60006020828403121561374a57600080fd5b8135612b7781613706565b60006020828403121561376757600080fd5b813567ffffffffffffffff81111561377e57600080fd5b820160e08185031215612b7757600080fd5b6020808252825182820181905260009190848201906040850190845b818110156137c8578351835292840192918401916001016137ac565b50909695505050505050565b6000606082840312156137e657600080fd5b50919050565b6000602082840312156137fe57600080fd5b5035919050565b600061014082518452602083015160208501526040830151613833604086018267ffffffffffffffff169052565b50606083015161384f606086018267ffffffffffffffff169052565b50608083015161386b608086018267ffffffffffffffff169052565b5060a083015160a085015260c083015161389d60c086018273ffffffffffffffffffffffffffffffffffffffff169052565b5060e08301516138c560e086018273ffffffffffffffffffffffffffffffffffffffff169052565b506101008381015115159085015261012080840151818601839052612f65838701826136a9565b602081526000612b776020830184613805565b600061010082840312156137e657600080fd5b6000806040838503121561392557600080fd5b823561393081613706565b946020939093013593505050565b60006020828403121561395057600080fd5b813567ffffffffffffffff81111561396757600080fd5b820160408185031215612b7757600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff618336030181126139dc57600080fd5b9190910192915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405160a0810167ffffffffffffffff81118282101715613a3857613a386139e6565b60405290565b60405160c0810167ffffffffffffffff81118282101715613a3857613a386139e6565b6040516080810167ffffffffffffffff81118282101715613a3857613a386139e6565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff81118282101715613acb57613acb6139e6565b604052919050565b600067ffffffffffffffff821115613aed57613aed6139e6565b5060051b60200190565b600060408284031215613b0957600080fd5b6040516040810181811067ffffffffffffffff82111715613b2c57613b2c6139e6565b604052823581526020928301359281019290925250919050565b600060608284031215613b5857600080fd5b6040516060810181811067ffffffffffffffff82111715613b7b57613b7b6139e6565b604052905080823560ff81168114613b9257600080fd5b8082525060208301356020820152604083013560408201525092915050565b600082601f830112613bc257600080fd5b81356020613bd7613bd283613ad3565b613a84565b82815260609283028501820192828201919087851115613bf657600080fd5b8387015b85811015613c1957613c0c8982613b46565b8452928401928101613bfa565b5090979650505050505050565b803567ffffffffffffffff8116811461373357600080fd5b600060a08236031215613c5057600080fd5b613c58613a15565b8235815260208084013567ffffffffffffffff80821115613c7857600080fd5b9085019036601f830112613c8b57600080fd5b8135613c99613bd282613ad3565b81815260069190911b83018401908481019036831115613cb857600080fd5b938501935b82851015613ce157613ccf3686613af7565b82528582019150604085019450613cbd565b80868801525050506040860135925080831115613cfd57600080fd5b5050613d0b36828601613bb1565b604083015250613d1d60608401613728565b6060820152613d2e60808401613c26565b608082015292915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b818103818111156106f6576106f6613d39565b801515811461338457600080fd5b600067ffffffffffffffff821115613da357613da36139e6565b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01660200190565b600060c08284031215613de157600080fd5b613de9613a3e565b90508135613df681613706565b81526020613e05838201613c26565b818301526040830135613e1781613d7b565b604083015260608381013590830152608083013567ffffffffffffffff811115613e4057600080fd5b8301601f81018513613e5157600080fd5b8035613e5f613bd282613d89565b8181528684838501011115613e7357600080fd5b818484018583013760008483830101528060808601525050505060a082013560a082015292915050565b600060e08236031215613eaf57600080fd5b613eb7613a15565b82358152602083013567ffffffffffffffff811115613ed557600080fd5b613ee136828601613dcf565b602083015250613ef43660408501613b46565b604082015260a0830135613f0781613706565b6060820152613d2e60c08401613c26565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff418336030181126139dc57600080fd5b60006106f63683613dcf565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc18336030181126139dc57600080fd5b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613fc157600080fd5b83018035915067ffffffffffffffff821115613fdc57600080fd5b6020019150600581901b360382131561304e57600080fd5b6000614002613bd284613ad3565b80848252602080830192508560051b85013681111561402057600080fd5b855b8181101561405c57803567ffffffffffffffff8111156140425760008081fd5b61404e36828a01613dcf565b865250938201938201614022565b50919695505050505050565b60006040828403121561407a57600080fd5b612b778383613af7565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126140b957600080fd5b83018035915067ffffffffffffffff8211156140d457600080fd5b6020019150600681901b360382131561304e57600080fd5b600084516140fe818460208901613685565b80830190507f2e00000000000000000000000000000000000000000000000000000000000000808252855161413a816001850160208a01613685565b60019201918201528351614155816002840160208801613685565b0160020195945050505050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe184360301811261419757600080fd5b83018035915067ffffffffffffffff8211156141b257600080fd5b602001915060608102360382131561304e57600080fd5b6000606082840312156141db57600080fd5b612b778383613b46565b6000602082840312156141f757600080fd5b612b7782613c26565b600181811c9082168061421457607f821691505b6020821081036137e6577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000610100828403121561426057600080fd5b614268613a15565b823581526142798460208501613af7565b602082015261428b8460608501613b46565b604082015260c083013561429e81613706565b60608201526142af60e08401613c26565b60808201529392505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036142ec576142ec613d39565b5060010190565b6000602080838503121561430657600080fd5b825167ffffffffffffffff8082111561431e57600080fd5b908401906080828703121561433257600080fd5b61433a613a61565b825181528383015161434b81613706565b81850152604083015161435d81613d7b565b604082015260608301518281111561437457600080fd5b80840193505086601f84011261438957600080fd5b82519150614399613bd283613d89565b82815287858486010111156143ad57600080fd5b6143bc83868301878701613685565b60608201529695505050505050565b601f821115610a2f57600081815260208120601f850160051c810160208610156143f25750805b601f850160051c820191505b81811015614411578281556001016143fe565b505050505050565b815167ffffffffffffffff811115614433576144336139e6565b614447816144418454614200565b846143cb565b602080601f83116001811461449a57600084156144645750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555614411565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b828110156144e7578886015182559484019460019091019084016144c8565b508582101561452357878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60008261457157614571614533565b500490565b60008261458557614585614533565b500690565b808201808211156106f6576106f6613d39565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b82815260406020820152600061286060408301846136a9565b600082516139dc818460208701613685565b60006020828403121561460957600080fd5b5051919050565b60006020828403121561462257600080fd5b8151612b7781613d7b565b6000604082016040835280855180835260608501915060608160051b8601019250602080880160005b838110156146a2577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa0888703018552614690868351613805565b95509382019390820190600101614656565b50508584038187015286518085528782019482019350915060005b828110156146d9578451845293810193928101926001016146bd565b5091979650505050505050565b89815260007fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808b60601b166020840152808a60601b166034840152507fffffffffffffffff000000000000000000000000000000000000000000000000808960c01b166048840152808860c01b1660508401525085151560f81b6058830152846059830152835161477f816079850160208801613685565b80830190507fffffffff000000000000000000000000000000000000000000000000000000008460e01b166079820152607d81019150509a995050505050505050505056fea164736f6c6343000813000a",
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
