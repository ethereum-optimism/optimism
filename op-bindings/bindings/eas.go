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
	Signature EIP712Signature
	Attester  common.Address
}

// DelegatedRevocationRequest is an auto generated low-level Go binding around an user-defined struct.
type DelegatedRevocationRequest struct {
	Schema    [32]byte
	Data      RevocationRequestData
	Signature EIP712Signature
	Revoker   common.Address
}

// EIP712Signature is an auto generated low-level Go binding around an user-defined struct.
type EIP712Signature struct {
	V uint8
	R [32]byte
	S [32]byte
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
	Signatures []EIP712Signature
	Attester   common.Address
}

// MultiDelegatedRevocationRequest is an auto generated low-level Go binding around an user-defined struct.
type MultiDelegatedRevocationRequest struct {
	Schema     [32]byte
	Data       []RevocationRequestData
	Signatures []EIP712Signature
	Revoker    common.Address
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

// EASMetaData contains all meta data concerning the EAS contract.
var EASMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractISchemaRegistry\",\"name\":\"registry\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AccessDenied\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyRevoked\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyRevokedOffchain\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"AlreadyTimestamped\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InsufficientValue\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAttestation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAttestations\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidExpirationTime\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidLength\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidOffset\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRegistry\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRevocation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidRevocations\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSchema\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignature\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidVerifier\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Irrevocable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotFound\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotPayable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WrongSchema\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"}],\"name\":\"Attested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"}],\"name\":\"Revoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"RevokedOffchain\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"name\":\"Timestamped\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData\",\"name\":\"data\",\"type\":\"tuple\"}],\"internalType\":\"structAttestationRequest\",\"name\":\"request\",\"type\":\"tuple\"}],\"name\":\"attest\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData\",\"name\":\"data\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structEIP712Signature\",\"name\":\"signature\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"}],\"internalType\":\"structDelegatedAttestationRequest\",\"name\":\"delegatedRequest\",\"type\":\"tuple\"}],\"name\":\"attestByDelegation\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getAttestTypeHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"}],\"name\":\"getAttestation\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"time\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"revocationTime\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structAttestation\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getDomainSeparator\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getName\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"getNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"getRevokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getRevokeTypeHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getSchemaRegistry\",\"outputs\":[{\"internalType\":\"contractISchemaRegistry\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"getTimestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"}],\"name\":\"isAttestationValid\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"}],\"internalType\":\"structMultiAttestationRequest[]\",\"name\":\"multiRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiAttest\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"expirationTime\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"refUID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structAttestationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structEIP712Signature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"attester\",\"type\":\"address\"}],\"internalType\":\"structMultiDelegatedAttestationRequest[]\",\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiAttestByDelegation\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"}],\"internalType\":\"structMultiRevocationRequest[]\",\"name\":\"multiRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiRevoke\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData[]\",\"name\":\"data\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structEIP712Signature[]\",\"name\":\"signatures\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"}],\"internalType\":\"structMultiDelegatedRevocationRequest[]\",\"name\":\"multiDelegatedRequests\",\"type\":\"tuple[]\"}],\"name\":\"multiRevokeByDelegation\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"data\",\"type\":\"bytes32[]\"}],\"name\":\"multiRevokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"data\",\"type\":\"bytes32[]\"}],\"name\":\"multiTimestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData\",\"name\":\"data\",\"type\":\"tuple\"}],\"internalType\":\"structRevocationRequest\",\"name\":\"request\",\"type\":\"tuple\"}],\"name\":\"revoke\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"schema\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structRevocationRequestData\",\"name\":\"data\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structEIP712Signature\",\"name\":\"signature\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"revoker\",\"type\":\"address\"}],\"internalType\":\"structDelegatedRevocationRequest\",\"name\":\"delegatedRequest\",\"type\":\"tuple\"}],\"name\":\"revokeByDelegation\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"revokeOffchain\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"data\",\"type\":\"bytes32\"}],\"name\":\"timestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101e06040523480156200001257600080fd5b5060405162004a5738038062004a57833981016040819052620000359162000164565b604080518082018252600381526245415360e81b6020808301918252835180850190945260058452640312e302e360dc1b9084019081526001608052600060a081905260c052825190912083519091206101408290526101608190524661010052919291839183917f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f6200010e8184846040805160208101859052908101839052606081018290524660808201523060a082015260009060c0016040516020818303038152906040528051906020012090509392505050565b60e0523061012052610180525050505060208201516101a05250506001600160a01b03811662000151576040516311a1e69760e01b815260040160405180910390fd5b6001600160a01b03166101c05262000196565b6000602082840312156200017757600080fd5b81516001600160a01b03811681146200018f57600080fd5b9392505050565b60805160a05160c05160e05161010051610120516101405161016051610180516101a0516101c05161482c6200022b600039600081816104fa0152818161160e0152611db2015260006105830152600061291901526000612968015260006129430152600061289c015260006128c6015260006128f0015260006108b50152600061088c01526000610863015261482c6000f3fe60806040526004361061018b5760003560e01c8063b469318d116100d6578063e45d03f91161007f578063ed24911d11610059578063ed24911d146104be578063f10b5cc8146104d3578063f17325e71461052457600080fd5b8063e45d03f914610478578063e57a6b1b1461048b578063e71ff3651461049e57600080fd5b8063d45c4435116100b0578063d45c4435146103ef578063e13458fc14610426578063e30bb5631461043957600080fd5b8063b469318d14610342578063b83010d31461039c578063cf190f34146103cf57600080fd5b80634cb7e9e5116101385780638129fc1c116101125780638129fc1c146102ed578063831e05a114610302578063a3112a641461031557600080fd5b80634cb7e9e5146102a55780634d003070146102b857806354fd4d50146102d857600080fd5b80632d0335ab116101695780632d0335ab1461022d57806344adc90e14610270578063469262671461029057600080fd5b806312b11a171461019057806313893f61146101d257806317d7de7c1461020b575b600080fd5b34801561019c57600080fd5b507fdbfdf8dc2b135c26253e00d5b6cbe6f20457e003fd526d97cea183883570de615b6040519081526020015b60405180910390f35b3480156101de57600080fd5b506101f26101ed36600461373b565b610537565b60405167ffffffffffffffff90911681526020016101c9565b34801561021757600080fd5b5061022061057c565b6040516101c991906137eb565b34801561023957600080fd5b506101bf610248366004613837565b73ffffffffffffffffffffffffffffffffffffffff1660009081526001602052604090205490565b61028361027e36600461373b565b6105ac565b6040516101c99190613854565b6102a361029e366004613898565b6106e3565b005b6102a36102b336600461373b565b610767565b3480156102c457600080fd5b506101f26102d33660046138b0565b61084f565b3480156102e457600080fd5b5061022061085c565b3480156102f957600080fd5b506102a36108ff565b61028361031036600461373b565b610a96565b34801561032157600080fd5b506103356103303660046138b0565b610ce7565b6040516101c991906139b0565b34801561034e57600080fd5b506101f261035d3660046139c3565b73ffffffffffffffffffffffffffffffffffffffff919091166000908152603560209081526040808320938352929052205467ffffffffffffffff1690565b3480156103a857600080fd5b507fa98d02348410c9c76735e0d0bb1396f4015ac2bb9615f9c2611d19d7a8a996506101bf565b3480156103db57600080fd5b506101f26103ea3660046138b0565b610eaa565b3480156103fb57600080fd5b506101f261040a3660046138b0565b60009081526034602052604090205467ffffffffffffffff1690565b6101bf6104343660046139ef565b610eb8565b34801561044557600080fd5b506104686104543660046138b0565b600090815260336020526040902054151590565b60405190151581526020016101c9565b6102a361048636600461373b565b610fbb565b6102a3610499366004613a2a565b611136565b3480156104aa57600080fd5b506101f26104b936600461373b565b6111db565b3480156104ca57600080fd5b506101bf611213565b3480156104df57600080fd5b5060405173ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001681526020016101c9565b6101bf610532366004613a3c565b61121d565b60004282825b81811015610570576105683387878481811061055b5761055b613a77565b90506020020135856112db565b60010161053d565b50909150505b92915050565b60606105a77f00000000000000000000000000000000000000000000000000000000000000006113da565b905090565b606060008267ffffffffffffffff8111156105c9576105c9613aa6565b6040519080825280602002602001820160405280156105fc57816020015b60608152602001906001900390816105e75790505b509050600034815b858110156106ce577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff860181143688888481811061064457610644613a77565b90506020028101906106569190613ad5565b9050600061067d823561066c6020850185613b13565b61067591613d8c565b338887611568565b805190915061068c9086613e2f565b945080602001518785815181106106a5576106a5613a77565b6020026020010181905250806020015151860195505050506106c78160010190565b9050610604565b506106d98383611c9c565b9695505050505050565b604080516001808252818301909252600091816020015b60408051808201909152600080825260208201528152602001906001900390816106fa57905050905061073536839003830160208401613e91565b8160008151811061074857610748613a77565b602090810291909101015261076282358233346001611d69565b505050565b3460005b82811015610849577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff83018114368585848181106107ab576107ab613a77565b90506020028101906107bd9190613ad5565b905061082a81356107d16020840184613ead565b808060200260200160405190810160405280939291908181526020016000905b8282101561081d5761080e60408302860136819003810190613e91565b815260200190600101906107f1565b5050505050338786611d69565b6108349085613e2f565b935050506108428160010190565b905061076b565b50505050565b60004261057683826123c5565b60606108877f0000000000000000000000000000000000000000000000000000000000000000612487565b6108b07f0000000000000000000000000000000000000000000000000000000000000000612487565b6108d97f0000000000000000000000000000000000000000000000000000000000000000612487565b6040516020016108eb93929190613f15565b604051602081830303815290604052905090565b600054610100900460ff161580801561091f5750600054600160ff909116105b806109395750303b158015610939575060005460ff166001145b6109ca576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790558015610a2857600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b610a306125c4565b8015610a9357600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50565b606060008267ffffffffffffffff811115610ab357610ab3613aa6565b604051908082528060200260200182016040528015610ae657816020015b6060815260200190600190039081610ad15790505b509050600034815b858110156106ce577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8601811436888884818110610b2e57610b2e613a77565b9050602002810190610b409190613f8b565b9050366000610b526020840184613b13565b9092509050801580610b725750610b6c6040840184613fbf565b82141590505b15610ba9576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b81811015610c6b57610c63604051806080016040528086600001358152602001858585818110610bde57610bde613a77565b9050602002810190610bf09190614026565b610bf99061405a565b8152602001610c0b6040880188613fbf565b85818110610c1b57610c1b613a77565b905060600201803603810190610c3191906140d1565b8152602001610c466080880160608901613837565b73ffffffffffffffffffffffffffffffffffffffff169052612665565b600101610bac565b506000610c948435610c7d8486613d8c565b610c8d6080880160608901613837565b8a89611568565b8051909150610ca39088613e2f565b96508060200151898781518110610cbc57610cbc613a77565b6020026020010181905250806020015151880197505050505050610ce08160010190565b9050610aee565b604080516101408101825260008082526020820181905291810182905260608082018390526080820183905260a0820183905260c0820183905260e0820183905261010082019290925261012081019190915260008281526033602090815260409182902082516101408101845281548152600182015492810192909252600281015467ffffffffffffffff808216948401949094526801000000000000000081048416606084015270010000000000000000000000000000000090049092166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff16151561010082015260068201805491929161012084019190610e21906140ed565b80601f0160208091040260200160405190810160405280929190818152602001828054610e4d906140ed565b8015610e9a5780601f10610e6f57610100808354040283529160200191610e9a565b820191906000526020600020905b815481529060010190602001808311610e7d57829003601f168201915b5050505050815250509050919050565b6000426105763384836112db565b6000610ecb610ec68361413a565b612665565b604080516001808252818301909252600091816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff909201910181610ee2579050509050610f506020840184614026565b610f599061405a565b81600081518110610f6c57610f6c613a77565b6020908102919091010152610f95833582610f8d60c0870160a08801613837565b346001611568565b60200151600081518110610fab57610fab613a77565b6020026020010151915050919050565b3460005b82811015610849577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff83018114600085858481811061100057611000613a77565b90506020028101906110129190613f8b565b61101b9061421f565b602081015180519192509015806110385750816040015151815114155b1561106f576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b8151811015611100576110f86040518060800160405280856000015181526020018484815181106110a5576110a5613a77565b60200260200101518152602001856040015184815181106110c8576110c8613a77565b60200260200101518152602001856060015173ffffffffffffffffffffffffffffffffffffffff168152506127f4565b600101611072565b5061111682600001518284606001518887611d69565b6111209086613e2f565b945050505061112f8160010190565b9050610fbf565b61114d611148368390038301836142fe565b6127f4565b604080516001808252818301909252600091816020015b604080518082019091526000808252602082015281526020019060019003908161116457905050905061119f36839003830160208401613e91565b816000815181106111b2576111b2613a77565b60209081029190910101526107628235826111d360e0860160c08701613837565b346001611d69565b60004282825b818110156105705761120b8686838181106111fe576111fe613a77565b90506020020135846123c5565b6001016111e1565b60006105a7612882565b604080516001808252818301909252600091829190816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816112375790505090506112a56020840184614026565b6112ae9061405a565b816000815181106112c1576112c1613a77565b6020908102919091010152610f9583358233346001611568565b73ffffffffffffffffffffffffffffffffffffffff83166000908152603560209081526040808320858452918290529091205467ffffffffffffffff161561134f576040517fec9d6eeb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008381526020829052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff861690811790915590519091859173ffffffffffffffffffffffffffffffffffffffff8816917f92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a222991a450505050565b604080516020808252818301909252606091600091906020820181803683370190505090506000805b60208110156114a557600085826020811061142057611420613a77565b1a60f81b90507fff00000000000000000000000000000000000000000000000000000000000000811660000361145657506114a5565b8084848151811061146957611469613a77565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053505060019182019101611403565b5060008167ffffffffffffffff8111156114c1576114c1613aa6565b6040519080825280601f01601f1916602001820160405280156114eb576020820181803683370190505b50905060005b8281101561155f5783818151811061150b5761150b613a77565b602001015160f81c60f81b82828151811061152857611528613a77565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506001016114f1565b50949350505050565b60408051808201909152600081526060602082015284516040805180820190915260008152606060208201528167ffffffffffffffff8111156115ad576115ad613aa6565b6040519080825280602002602001820160405280156115d6578160200160208202803683370190505b5060208201526040517fa2ea7c6e000000000000000000000000000000000000000000000000000000008152600481018990526000907f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff169063a2ea7c6e90602401600060405180830381865afa15801561166a573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01682016040526116b0919081019061435a565b80519091506116eb576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008367ffffffffffffffff81111561170657611706613aa6565b6040519080825280602002602001820160405280156117a557816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816117245790505b50905060008467ffffffffffffffff8111156117c3576117c3613aa6565b6040519080825280602002602001820160405280156117ec578160200160208202803683370190505b50905060005b85811015611c7b5760008b828151811061180e5761180e613a77565b60200260200101519050600067ffffffffffffffff16816020015167ffffffffffffffff161415801561185957504267ffffffffffffffff16816020015167ffffffffffffffff1611155b15611890576040517f08e8b93700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b84604001511580156118a3575080604001515b156118da576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006040518061014001604052806000801b81526020018f81526020016118fe4290565b67ffffffffffffffff168152602001836020015167ffffffffffffffff168152602001600067ffffffffffffffff16815260200183606001518152602001836000015173ffffffffffffffffffffffffffffffffffffffff1681526020018d73ffffffffffffffffffffffffffffffffffffffff16815260200183604001511515815260200183608001518152509050600080600090505b6119a083826129b6565b600081815260336020526040902054909250156119bf57600101611996565b81835260008281526033602090815260409182902085518155908501516001820155908401516002820180546060870151608088015167ffffffffffffffff908116700100000000000000000000000000000000027fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff92821668010000000000000000027fffffffffffffffffffffffffffffffff000000000000000000000000000000009094169190951617919091171691909117905560a0840151600382015560c084015160048201805473ffffffffffffffffffffffffffffffffffffffff9283167fffffffffffffffffffffffff000000000000000000000000000000000000000090911617905560e0850151600583018054610100880151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff00000000000000000000000000000000000000000090911692909316919091179190911790556101208401518491906006820190611b3f9082614480565b505050606084015115611b96576060840151600090815260336020526040902054611b96576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82878681518110611ba957611ba9613a77565b60200260200101819052508360a00151868681518110611bcb57611bcb613a77565b6020026020010181815250508189602001518681518110611bee57611bee613a77565b6020026020010181815250508f8e73ffffffffffffffffffffffffffffffffffffffff16856000015173ffffffffffffffffffffffffffffffffffffffff167f8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b3585604051611c5e91815260200190565b60405180910390a450505050611c748160010190565b90506117f2565b50611c8b83838360008c8c612a15565b845250919998505050505050505050565b606060008267ffffffffffffffff811115611cb957611cb9613aa6565b604051908082528060200260200182016040528015611ce2578160200160208202803683370190505b5090506000805b8551811015610570576000868281518110611d0657611d06613a77565b6020026020010151905060005b8151811015611d5f57818181518110611d2e57611d2e613a77565b6020026020010151858581518110611d4857611d48613a77565b602090810291909101015260019384019301611d13565b5050600101611ce9565b6040517fa2ea7c6e00000000000000000000000000000000000000000000000000000000815260048101869052600090819073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063a2ea7c6e90602401600060405180830381865afa158015611df9573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0168201604052611e3f919081019061435a565b8051909150611e7a576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b855160008167ffffffffffffffff811115611e9757611e97613aa6565b604051908082528060200260200182016040528015611f3657816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff909201910181611eb55790505b50905060008267ffffffffffffffff811115611f5457611f54613aa6565b604051908082528060200260200182016040528015611f7d578160200160208202803683370190505b50905060005b838110156123a75760008a8281518110611f9f57611f9f613a77565b6020908102919091018101518051600090815260339092526040909120805491925090611ff8576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8c816001015414612035576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015473ffffffffffffffffffffffffffffffffffffffff8c811691161461208b576040517f4ca8886700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015474010000000000000000000000000000000000000000900460ff166120e1576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002810154700100000000000000000000000000000000900467ffffffffffffffff161561213b576040517f905e710700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b426002820180547fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff811670010000000000000000000000000000000067ffffffffffffffff948516810291821793849055604080516101408101825287548152600188015460208201529386169286169290921791830191909152680100000000000000008304841660608301529091049091166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff1615156101008201526006820180548391610120840191612247906140ed565b80601f0160208091040260200160405190810160405280929190818152602001828054612273906140ed565b80156122c05780601f10612295576101008083540402835291602001916122c0565b820191906000526020600020905b8154815290600101906020018083116122a357829003601f168201915b5050505050815250508584815181106122db576122db613a77565b602002602001018190525081602001518484815181106122fd576122fd613a77565b60200260200101818152505080600101548b73ffffffffffffffffffffffffffffffffffffffff168260040160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167ff930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f615856000015160405161239591815260200190565b60405180910390a45050600101611f83565b506123b784838360018b8b612a15565b9a9950505050505050505050565b60008281526034602052604090205467ffffffffffffffff1615612415576040517f2e26794600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008281526034602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff85169081179091559051909184917f5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f9190a35050565b6060816000036124ca57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156124f457806124de8161459a565b91506124ed9050600a83614601565b91506124ce565b60008167ffffffffffffffff81111561250f5761250f613aa6565b6040519080825280601f01601f191660200182016040528015612539576020820181803683370190505b5090505b84156125bc5761254e600183613e2f565b915061255b600a86614615565b612566906030614629565b60f81b81838151811061257b5761257b613a77565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506125b5600a86614601565b945061253d565b949350505050565b600054610100900460ff1661265b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016109c1565b612663612def565b565b60208082015160408084015160608086015173ffffffffffffffffffffffffffffffffffffffff1660009081526001808752848220805491820190558751865187890151878901519589015160808a01518051908c01209851999a97999498959761276b97612750977fdbfdf8dc2b135c26253e00d5b6cbe6f20457e003fd526d97cea183883570de619791939290918c9101978852602088019690965273ffffffffffffffffffffffffffffffffffffffff94909416604087015267ffffffffffffffff9290921660608601521515608085015260a084015260c083015260e08201526101000190565b60405160208183030381529060405280519060200120612e86565b9050846060015173ffffffffffffffffffffffffffffffffffffffff166127a082856000015186602001518760400151612e99565b73ffffffffffffffffffffffffffffffffffffffff16146127ed576040517f8baa579f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5050505050565b60208181015160408084015160608086015173ffffffffffffffffffffffffffffffffffffffff1660009081526001808752848220805491820190558751865186517fa98d02348410c9c76735e0d0bb1396f4015ac2bb9615f9c2611d19d7a8a996509981019990995295880152918601939093526080850181905292939092919061276b9060a001612750565b60003073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000161480156128e857507f000000000000000000000000000000000000000000000000000000000000000046145b1561291257507f000000000000000000000000000000000000000000000000000000000000000090565b50604080517f00000000000000000000000000000000000000000000000000000000000000006020808301919091527f0000000000000000000000000000000000000000000000000000000000000000828401527f000000000000000000000000000000000000000000000000000000000000000060608301524660808301523060a0808401919091528351808403909101815260c0909201909252805191012090565b60208083015160c084015160e0850151604080870151606088015161010089015160a08a01516101208b015194516000996129f799989796918c910161463c565b60405160208183030381529060405280519060200120905092915050565b84516000906001819003612a6d57612a658888600081518110612a3a57612a3a613a77565b602002602001015188600081518110612a5557612a55613a77565b6020026020010151888888612ec1565b9150506106d9565b602088015173ffffffffffffffffffffffffffffffffffffffff8116612aff5760005b82811015612af357878181518110612aaa57612aaa613a77565b6020026020010151600014612aeb576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600101612a90565b506000925050506106d9565b6000805b83811015612c29576000898281518110612b1f57612b1f613a77565b6020026020010151905080600014158015612ba657508373ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612b80573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612ba4919061471a565b155b15612bdd576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b87811115612c17576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b96879003969190910190600101612b03565b508615612d04576040517f88e5b2d900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8316906388e5b2d9908390612c86908d908d90600401614737565b60206040518083038185885af1158015612ca4573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612cc9919061471a565b612cff576040517fbf2f3a8b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b612dd3565b6040517f91db0b7e00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8316906391db0b7e908390612d5a908d908d90600401614737565b60206040518083038185885af1158015612d78573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612d9d919061471a565b612dd3576040517fe8bee83900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8415612de257612de2866131d7565b9998505050505050505050565b600054610100900460ff16612663576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016109c1565b6000610576612e93612882565b836131e7565b6000806000612eaa87878787613229565b91509150612eb781613341565b5095945050505050565b602086015160009073ffffffffffffffffffffffffffffffffffffffff8116612f26578515612f1c576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60009150506106d9565b8515801590612fa157508073ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612f7b573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612f9f919061471a565b155b15612fd8576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b83861115613012576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b858403935084156130ef576040517fe49617e100000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e49617e1908890613071908b906004016139b0565b60206040518083038185885af115801561308f573d6000803e3d6000fd5b50505050506040513d601f19601f820116820180604052508101906130b4919061471a565b6130ea576040517fccf3bb2700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6131bc565b6040517fe60c350500000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e60c3505908890613143908b906004016139b0565b60206040518083038185885af1158015613161573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190613186919061471a565b6131bc576040517fbd8ba84d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82156131cb576131cb846131d7565b50939695505050505050565b8015610a9357610a933382613595565b6040517f1901000000000000000000000000000000000000000000000000000000000000602082015260228101839052604281018290526000906062016129f7565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08311156132605750600090506003613338565b8460ff16601b1415801561327857508460ff16601c14155b156132895750600090506004613338565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa1580156132dd573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff811661333157600060019250925050613338565b9150600090505b94509492505050565b6000816004811115613355576133556147f0565b0361335d5750565b6001816004811115613371576133716147f0565b036133d8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f45434453413a20696e76616c6964207369676e6174757265000000000000000060448201526064016109c1565b60028160048111156133ec576133ec6147f0565b03613453576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45434453413a20696e76616c6964207369676e6174757265206c656e6774680060448201526064016109c1565b6003816004811115613467576134676147f0565b036134f4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202773272076616c60448201527f756500000000000000000000000000000000000000000000000000000000000060648201526084016109c1565b6004816004811115613508576135086147f0565b03610a93576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202776272076616c60448201527f756500000000000000000000000000000000000000000000000000000000000060648201526084016109c1565b804710156135ff576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e636500000060448201526064016109c1565b60008273ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d8060008114613659576040519150601f19603f3d011682016040523d82523d6000602084013e61365e565b606091505b5050905080610762576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d6179206861766520726576657274656400000000000060648201526084016109c1565b60008083601f84011261370157600080fd5b50813567ffffffffffffffff81111561371957600080fd5b6020830191508360208260051b850101111561373457600080fd5b9250929050565b6000806020838503121561374e57600080fd5b823567ffffffffffffffff81111561376557600080fd5b613771858286016136ef565b90969095509350505050565b60005b83811015613798578181015183820152602001613780565b50506000910152565b600081518084526137b981602086016020860161377d565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006137fe60208301846137a1565b9392505050565b73ffffffffffffffffffffffffffffffffffffffff81168114610a9357600080fd5b803561383281613805565b919050565b60006020828403121561384957600080fd5b81356137fe81613805565b6020808252825182820181905260009190848201906040850190845b8181101561388c57835183529284019291840191600101613870565b50909695505050505050565b6000606082840312156138aa57600080fd5b50919050565b6000602082840312156138c257600080fd5b5035919050565b6000610140825184526020830151602085015260408301516138f7604086018267ffffffffffffffff169052565b506060830151613913606086018267ffffffffffffffff169052565b50608083015161392f608086018267ffffffffffffffff169052565b5060a083015160a085015260c083015161396160c086018273ffffffffffffffffffffffffffffffffffffffff169052565b5060e083015161398960e086018273ffffffffffffffffffffffffffffffffffffffff169052565b5061010083810151151590850152610120808401518186018390526106d9838701826137a1565b6020815260006137fe60208301846138c9565b600080604083850312156139d657600080fd5b82356139e181613805565b946020939093013593505050565b600060208284031215613a0157600080fd5b813567ffffffffffffffff811115613a1857600080fd5b820160c081850312156137fe57600080fd5b600060e082840312156138aa57600080fd5b600060208284031215613a4e57600080fd5b813567ffffffffffffffff811115613a6557600080fd5b8201604081850312156137fe57600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc1833603018112613b0957600080fd5b9190910192915050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613b4857600080fd5b83018035915067ffffffffffffffff821115613b6357600080fd5b6020019150600581901b360382131561373457600080fd5b60405160c0810167ffffffffffffffff81118282101715613b9e57613b9e613aa6565b60405290565b6040516080810167ffffffffffffffff81118282101715613b9e57613b9e613aa6565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff81118282101715613c0e57613c0e613aa6565b604052919050565b600067ffffffffffffffff821115613c3057613c30613aa6565b5060051b60200190565b8015158114610a9357600080fd5b803561383281613c3a565b600067ffffffffffffffff821115613c6d57613c6d613aa6565b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01660200190565b600082601f830112613caa57600080fd5b8135613cbd613cb882613c53565b613bc7565b818152846020838601011115613cd257600080fd5b816020850160208301376000918101602001919091529392505050565b600060c08284031215613d0157600080fd5b613d09613b7b565b90508135613d1681613805565b8152602082013567ffffffffffffffff8082168214613d3457600080fd5b816020840152613d4660408501613c48565b6040840152606084013560608401526080840135915080821115613d6957600080fd5b50613d7684828501613c99565b60808301525060a082013560a082015292915050565b6000613d9a613cb884613c16565b80848252602080830192508560051b850136811115613db857600080fd5b855b81811015613df457803567ffffffffffffffff811115613dda5760008081fd5b613de636828a01613cef565b865250938201938201613dba565b50919695505050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8181038181111561057657610576613e00565b600060408284031215613e5457600080fd5b6040516040810181811067ffffffffffffffff82111715613e7757613e77613aa6565b604052823581526020928301359281019290925250919050565b600060408284031215613ea357600080fd5b6137fe8383613e42565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613ee257600080fd5b83018035915067ffffffffffffffff821115613efd57600080fd5b6020019150600681901b360382131561373457600080fd5b60008451613f2781846020890161377d565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551613f63816001850160208a0161377d565b60019201918201528351613f7e81600284016020880161377d565b0160020195945050505050565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81833603018112613b0957600080fd5b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613ff457600080fd5b83018035915067ffffffffffffffff82111561400f57600080fd5b602001915060608102360382131561373457600080fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff41833603018112613b0957600080fd5b60006105763683613cef565b60006060828403121561407857600080fd5b6040516060810181811067ffffffffffffffff8211171561409b5761409b613aa6565b604052905080823560ff811681146140b257600080fd5b8082525060208301356020820152604083013560408201525092915050565b6000606082840312156140e357600080fd5b6137fe8383614066565b600181811c9082168061410157607f821691505b6020821081036138aa577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060c0823603121561414c57600080fd5b614154613ba4565b82358152602083013567ffffffffffffffff81111561417257600080fd5b61417e36828601613cef565b6020830152506141913660408501614066565b604082015260a08301356141a481613805565b606082015292915050565b600082601f8301126141c057600080fd5b813560206141d0613cb883613c16565b828152606092830285018201928282019190878511156141ef57600080fd5b8387015b85811015614212576142058982614066565b84529284019281016141f3565b5090979650505050505050565b60006080823603121561423157600080fd5b614239613ba4565b8235815260208084013567ffffffffffffffff8082111561425957600080fd5b9085019036601f83011261426c57600080fd5b813561427a613cb882613c16565b81815260069190911b8301840190848101903683111561429957600080fd5b938501935b828510156142c2576142b03686613e42565b8252858201915060408501945061429e565b808688015250505060408601359250808311156142de57600080fd5b50506142ec368286016141af565b6040830152506141a460608401613827565b600060e0828403121561431057600080fd5b614318613ba4565b823581526143298460208501613e42565b602082015261433b8460608501614066565b604082015260c083013561434e81613805565b60608201529392505050565b6000602080838503121561436d57600080fd5b825167ffffffffffffffff8082111561438557600080fd5b908401906080828703121561439957600080fd5b6143a1613ba4565b82518152838301516143b281613805565b8185015260408301516143c481613c3a565b60408201526060830151828111156143db57600080fd5b80840193505086601f8401126143f057600080fd5b82519150614400613cb883613c53565b828152878584860101111561441457600080fd5b6144238386830187870161377d565b60608201529695505050505050565b601f82111561076257600081815260208120601f850160051c810160208610156144595750805b601f850160051c820191505b8181101561447857828155600101614465565b505050505050565b815167ffffffffffffffff81111561449a5761449a613aa6565b6144ae816144a884546140ed565b84614432565b602080601f83116001811461450157600084156144cb5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555614478565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b8281101561454e5788860151825594840194600190910190840161452f565b508582101561458a57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036145cb576145cb613e00565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082614610576146106145d2565b500490565b600082614624576146246145d2565b500690565b8082018082111561057657610576613e00565b89815260007fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808b60601b166020840152808a60601b166034840152507fffffffffffffffff000000000000000000000000000000000000000000000000808960c01b166048840152808860c01b1660508401525085151560f81b605883015284605983015283516146d581607985016020880161377d565b80830190507fffffffff000000000000000000000000000000000000000000000000000000008460e01b166079820152607d81019150509a9950505050505050505050565b60006020828403121561472c57600080fd5b81516137fe81613c3a565b6000604082016040835280855180835260608501915060608160051b8601019250602080880160005b838110156147ac577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa088870301855261479a8683516138c9565b95509382019390820190600101614760565b50508584038187015286518085528782019482019350915060005b828110156147e3578451845293810193928101926001016147c7565b5091979650505050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fdfea164736f6c6343000813000a",
}

// EASABI is the input ABI used to generate the binding from.
// Deprecated: Use EASMetaData.ABI instead.
var EASABI = EASMetaData.ABI

// EASBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EASMetaData.Bin instead.
var EASBin = EASMetaData.Bin

// DeployEAS deploys a new Ethereum contract, binding an instance of EAS to it.
func DeployEAS(auth *bind.TransactOpts, backend bind.ContractBackend, registry common.Address) (common.Address, *types.Transaction, *EAS, error) {
	parsed, err := EASMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EASBin), backend, registry)
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
// Solidity: function getSchemaRegistry() view returns(address)
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
// Solidity: function getSchemaRegistry() view returns(address)
func (_EAS *EASSession) GetSchemaRegistry() (common.Address, error) {
	return _EAS.Contract.GetSchemaRegistry(&_EAS.CallOpts)
}

// GetSchemaRegistry is a free data retrieval call binding the contract method 0xf10b5cc8.
//
// Solidity: function getSchemaRegistry() view returns(address)
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

// AttestByDelegation is a paid mutator transaction binding the contract method 0xe13458fc.
//
// Solidity: function attestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256),(uint8,bytes32,bytes32),address) delegatedRequest) payable returns(bytes32)
func (_EAS *EASTransactor) AttestByDelegation(opts *bind.TransactOpts, delegatedRequest DelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "attestByDelegation", delegatedRequest)
}

// AttestByDelegation is a paid mutator transaction binding the contract method 0xe13458fc.
//
// Solidity: function attestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256),(uint8,bytes32,bytes32),address) delegatedRequest) payable returns(bytes32)
func (_EAS *EASSession) AttestByDelegation(delegatedRequest DelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.AttestByDelegation(&_EAS.TransactOpts, delegatedRequest)
}

// AttestByDelegation is a paid mutator transaction binding the contract method 0xe13458fc.
//
// Solidity: function attestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256),(uint8,bytes32,bytes32),address) delegatedRequest) payable returns(bytes32)
func (_EAS *EASTransactorSession) AttestByDelegation(delegatedRequest DelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.AttestByDelegation(&_EAS.TransactOpts, delegatedRequest)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_EAS *EASTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_EAS *EASSession) Initialize() (*types.Transaction, error) {
	return _EAS.Contract.Initialize(&_EAS.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_EAS *EASTransactorSession) Initialize() (*types.Transaction, error) {
	return _EAS.Contract.Initialize(&_EAS.TransactOpts)
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

// MultiAttestByDelegation is a paid mutator transaction binding the contract method 0x831e05a1.
//
// Solidity: function multiAttestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[],(uint8,bytes32,bytes32)[],address)[] multiDelegatedRequests) payable returns(bytes32[])
func (_EAS *EASTransactor) MultiAttestByDelegation(opts *bind.TransactOpts, multiDelegatedRequests []MultiDelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiAttestByDelegation", multiDelegatedRequests)
}

// MultiAttestByDelegation is a paid mutator transaction binding the contract method 0x831e05a1.
//
// Solidity: function multiAttestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[],(uint8,bytes32,bytes32)[],address)[] multiDelegatedRequests) payable returns(bytes32[])
func (_EAS *EASSession) MultiAttestByDelegation(multiDelegatedRequests []MultiDelegatedAttestationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiAttestByDelegation(&_EAS.TransactOpts, multiDelegatedRequests)
}

// MultiAttestByDelegation is a paid mutator transaction binding the contract method 0x831e05a1.
//
// Solidity: function multiAttestByDelegation((bytes32,(address,uint64,bool,bytes32,bytes,uint256)[],(uint8,bytes32,bytes32)[],address)[] multiDelegatedRequests) payable returns(bytes32[])
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

// MultiRevokeByDelegation is a paid mutator transaction binding the contract method 0xe45d03f9.
//
// Solidity: function multiRevokeByDelegation((bytes32,(bytes32,uint256)[],(uint8,bytes32,bytes32)[],address)[] multiDelegatedRequests) payable returns()
func (_EAS *EASTransactor) MultiRevokeByDelegation(opts *bind.TransactOpts, multiDelegatedRequests []MultiDelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "multiRevokeByDelegation", multiDelegatedRequests)
}

// MultiRevokeByDelegation is a paid mutator transaction binding the contract method 0xe45d03f9.
//
// Solidity: function multiRevokeByDelegation((bytes32,(bytes32,uint256)[],(uint8,bytes32,bytes32)[],address)[] multiDelegatedRequests) payable returns()
func (_EAS *EASSession) MultiRevokeByDelegation(multiDelegatedRequests []MultiDelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.MultiRevokeByDelegation(&_EAS.TransactOpts, multiDelegatedRequests)
}

// MultiRevokeByDelegation is a paid mutator transaction binding the contract method 0xe45d03f9.
//
// Solidity: function multiRevokeByDelegation((bytes32,(bytes32,uint256)[],(uint8,bytes32,bytes32)[],address)[] multiDelegatedRequests) payable returns()
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

// RevokeByDelegation is a paid mutator transaction binding the contract method 0xe57a6b1b.
//
// Solidity: function revokeByDelegation((bytes32,(bytes32,uint256),(uint8,bytes32,bytes32),address) delegatedRequest) payable returns()
func (_EAS *EASTransactor) RevokeByDelegation(opts *bind.TransactOpts, delegatedRequest DelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.contract.Transact(opts, "revokeByDelegation", delegatedRequest)
}

// RevokeByDelegation is a paid mutator transaction binding the contract method 0xe57a6b1b.
//
// Solidity: function revokeByDelegation((bytes32,(bytes32,uint256),(uint8,bytes32,bytes32),address) delegatedRequest) payable returns()
func (_EAS *EASSession) RevokeByDelegation(delegatedRequest DelegatedRevocationRequest) (*types.Transaction, error) {
	return _EAS.Contract.RevokeByDelegation(&_EAS.TransactOpts, delegatedRequest)
}

// RevokeByDelegation is a paid mutator transaction binding the contract method 0xe57a6b1b.
//
// Solidity: function revokeByDelegation((bytes32,(bytes32,uint256),(uint8,bytes32,bytes32),address) delegatedRequest) payable returns()
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
	Schema    [32]byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAttested is a free log retrieval operation binding the contract event 0x8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b35.
//
// Solidity: event Attested(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schema)
func (_EAS *EASFilterer) FilterAttested(opts *bind.FilterOpts, recipient []common.Address, attester []common.Address, schema [][32]byte) (*EASAttestedIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaRule []interface{}
	for _, schemaItem := range schema {
		schemaRule = append(schemaRule, schemaItem)
	}

	logs, sub, err := _EAS.contract.FilterLogs(opts, "Attested", recipientRule, attesterRule, schemaRule)
	if err != nil {
		return nil, err
	}
	return &EASAttestedIterator{contract: _EAS.contract, event: "Attested", logs: logs, sub: sub}, nil
}

// WatchAttested is a free log subscription operation binding the contract event 0x8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b35.
//
// Solidity: event Attested(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schema)
func (_EAS *EASFilterer) WatchAttested(opts *bind.WatchOpts, sink chan<- *EASAttested, recipient []common.Address, attester []common.Address, schema [][32]byte) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaRule []interface{}
	for _, schemaItem := range schema {
		schemaRule = append(schemaRule, schemaItem)
	}

	logs, sub, err := _EAS.contract.WatchLogs(opts, "Attested", recipientRule, attesterRule, schemaRule)
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
// Solidity: event Attested(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schema)
func (_EAS *EASFilterer) ParseAttested(log types.Log) (*EASAttested, error) {
	event := new(EASAttested)
	if err := _EAS.contract.UnpackLog(event, "Attested", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EASInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the EAS contract.
type EASInitializedIterator struct {
	Event *EASInitialized // Event containing the contract specifics and raw log

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
func (it *EASInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EASInitialized)
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
		it.Event = new(EASInitialized)
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
func (it *EASInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EASInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EASInitialized represents a Initialized event raised by the EAS contract.
type EASInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_EAS *EASFilterer) FilterInitialized(opts *bind.FilterOpts) (*EASInitializedIterator, error) {

	logs, sub, err := _EAS.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &EASInitializedIterator{contract: _EAS.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_EAS *EASFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *EASInitialized) (event.Subscription, error) {

	logs, sub, err := _EAS.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EASInitialized)
				if err := _EAS.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_EAS *EASFilterer) ParseInitialized(log types.Log) (*EASInitialized, error) {
	event := new(EASInitialized)
	if err := _EAS.contract.UnpackLog(event, "Initialized", log); err != nil {
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
	Schema    [32]byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRevoked is a free log retrieval operation binding the contract event 0xf930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f615.
//
// Solidity: event Revoked(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schema)
func (_EAS *EASFilterer) FilterRevoked(opts *bind.FilterOpts, recipient []common.Address, attester []common.Address, schema [][32]byte) (*EASRevokedIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaRule []interface{}
	for _, schemaItem := range schema {
		schemaRule = append(schemaRule, schemaItem)
	}

	logs, sub, err := _EAS.contract.FilterLogs(opts, "Revoked", recipientRule, attesterRule, schemaRule)
	if err != nil {
		return nil, err
	}
	return &EASRevokedIterator{contract: _EAS.contract, event: "Revoked", logs: logs, sub: sub}, nil
}

// WatchRevoked is a free log subscription operation binding the contract event 0xf930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f615.
//
// Solidity: event Revoked(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schema)
func (_EAS *EASFilterer) WatchRevoked(opts *bind.WatchOpts, sink chan<- *EASRevoked, recipient []common.Address, attester []common.Address, schema [][32]byte) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var attesterRule []interface{}
	for _, attesterItem := range attester {
		attesterRule = append(attesterRule, attesterItem)
	}

	var schemaRule []interface{}
	for _, schemaItem := range schema {
		schemaRule = append(schemaRule, schemaItem)
	}

	logs, sub, err := _EAS.contract.WatchLogs(opts, "Revoked", recipientRule, attesterRule, schemaRule)
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
// Solidity: event Revoked(address indexed recipient, address indexed attester, bytes32 uid, bytes32 indexed schema)
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
