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

// IAllowanceTransferAllowanceTransferDetails is an auto generated low-level Go binding around an user-defined struct.
type IAllowanceTransferAllowanceTransferDetails struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Token  common.Address
}

// IAllowanceTransferPermitBatch is an auto generated low-level Go binding around an user-defined struct.
type IAllowanceTransferPermitBatch struct {
	Details     []IAllowanceTransferPermitDetails
	Spender     common.Address
	SigDeadline *big.Int
}

// IAllowanceTransferPermitDetails is an auto generated low-level Go binding around an user-defined struct.
type IAllowanceTransferPermitDetails struct {
	Token      common.Address
	Amount     *big.Int
	Expiration *big.Int
	Nonce      *big.Int
}

// IAllowanceTransferPermitSingle is an auto generated low-level Go binding around an user-defined struct.
type IAllowanceTransferPermitSingle struct {
	Details     IAllowanceTransferPermitDetails
	Spender     common.Address
	SigDeadline *big.Int
}

// IAllowanceTransferTokenSpenderPair is an auto generated low-level Go binding around an user-defined struct.
type IAllowanceTransferTokenSpenderPair struct {
	Token   common.Address
	Spender common.Address
}

// ISignatureTransferPermitBatchTransferFrom is an auto generated low-level Go binding around an user-defined struct.
type ISignatureTransferPermitBatchTransferFrom struct {
	Permitted []ISignatureTransferTokenPermissions
	Nonce     *big.Int
	Deadline  *big.Int
}

// ISignatureTransferPermitTransferFrom is an auto generated low-level Go binding around an user-defined struct.
type ISignatureTransferPermitTransferFrom struct {
	Permitted ISignatureTransferTokenPermissions
	Nonce     *big.Int
	Deadline  *big.Int
}

// ISignatureTransferSignatureTransferDetails is an auto generated low-level Go binding around an user-defined struct.
type ISignatureTransferSignatureTransferDetails struct {
	To              common.Address
	RequestedAmount *big.Int
}

// ISignatureTransferTokenPermissions is an auto generated low-level Go binding around an user-defined struct.
type ISignatureTransferTokenPermissions struct {
	Token  common.Address
	Amount *big.Int
}

// Permit2MetaData contains all meta data concerning the Permit2 contract.
var Permit2MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"AllowanceExpired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ExcessiveInvalidation\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"InsufficientAllowance\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"maxAmount\",\"type\":\"uint256\"}],\"name\":\"InvalidAmount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidContractSignature\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidNonce\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignature\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignatureLength\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSigner\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LengthMismatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"signatureDeadline\",\"type\":\"uint256\"}],\"name\":\"SignatureExpired\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"uint48\",\"name\":\"expiration\",\"type\":\"uint48\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"Lockdown\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint48\",\"name\":\"newNonce\",\"type\":\"uint48\"},{\"indexed\":false,\"internalType\":\"uint48\",\"name\":\"oldNonce\",\"type\":\"uint48\"}],\"name\":\"NonceInvalidation\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"indexed\":false,\"internalType\":\"uint48\",\"name\":\"expiration\",\"type\":\"uint48\"},{\"indexed\":false,\"internalType\":\"uint48\",\"name\":\"nonce\",\"type\":\"uint48\"}],\"name\":\"Permit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"word\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"mask\",\"type\":\"uint256\"}],\"name\":\"UnorderedNonceInvalidation\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"internalType\":\"uint48\",\"name\":\"expiration\",\"type\":\"uint48\"},{\"internalType\":\"uint48\",\"name\":\"nonce\",\"type\":\"uint48\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"internalType\":\"uint48\",\"name\":\"expiration\",\"type\":\"uint48\"}],\"name\":\"approve\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint48\",\"name\":\"newNonce\",\"type\":\"uint48\"}],\"name\":\"invalidateNonces\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"wordPos\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"mask\",\"type\":\"uint256\"}],\"name\":\"invalidateUnorderedNonces\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"internalType\":\"structIAllowanceTransfer.TokenSpenderPair[]\",\"name\":\"approvals\",\"type\":\"tuple[]\"}],\"name\":\"lockdown\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"nonceBitmap\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"internalType\":\"uint48\",\"name\":\"expiration\",\"type\":\"uint48\"},{\"internalType\":\"uint48\",\"name\":\"nonce\",\"type\":\"uint48\"}],\"internalType\":\"structIAllowanceTransfer.PermitDetails[]\",\"name\":\"details\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"sigDeadline\",\"type\":\"uint256\"}],\"internalType\":\"structIAllowanceTransfer.PermitBatch\",\"name\":\"permitBatch\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"internalType\":\"uint48\",\"name\":\"expiration\",\"type\":\"uint48\"},{\"internalType\":\"uint48\",\"name\":\"nonce\",\"type\":\"uint48\"}],\"internalType\":\"structIAllowanceTransfer.PermitDetails\",\"name\":\"details\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"sigDeadline\",\"type\":\"uint256\"}],\"internalType\":\"structIAllowanceTransfer.PermitSingle\",\"name\":\"permitSingle\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.TokenPermissions\",\"name\":\"permitted\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.PermitTransferFrom\",\"name\":\"permit\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"requestedAmount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.SignatureTransferDetails\",\"name\":\"transferDetails\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permitTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.TokenPermissions[]\",\"name\":\"permitted\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.PermitBatchTransferFrom\",\"name\":\"permit\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"requestedAmount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.SignatureTransferDetails[]\",\"name\":\"transferDetails\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permitTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.TokenPermissions\",\"name\":\"permitted\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.PermitTransferFrom\",\"name\":\"permit\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"requestedAmount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.SignatureTransferDetails\",\"name\":\"transferDetails\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"witness\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"witnessTypeString\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permitWitnessTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.TokenPermissions[]\",\"name\":\"permitted\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.PermitBatchTransferFrom\",\"name\":\"permit\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"requestedAmount\",\"type\":\"uint256\"}],\"internalType\":\"structISignatureTransfer.SignatureTransferDetails[]\",\"name\":\"transferDetails\",\"type\":\"tuple[]\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"witness\",\"type\":\"bytes32\"},{\"internalType\":\"string\",\"name\":\"witnessTypeString\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permitWitnessTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"internalType\":\"structIAllowanceTransfer.AllowanceTransferDetails[]\",\"name\":\"transferDetails\",\"type\":\"tuple[]\"}],\"name\":\"transferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint160\",\"name\":\"amount\",\"type\":\"uint160\"},{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"}],\"name\":\"transferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60c0346100bb574660a052602081017f8cad95687ba82c2ce50e74f7b754645e5117c3a5bec8151c0726d5857980a86681527f9ac997416e8ff9d2ff6bebeb7149f65cdae5e32e2b90440b566bb3044041d36a60408301524660608301523060808301526080825260a082019180831060018060401b038411176100a557826040525190206080526123c090816100c1823960805181611b47015260a05181611b210152f35b634e487b7160e01b600052604160045260246000fd5b600080fdfe6040608081526004908136101561001557600080fd5b600090813560e01c80630d58b1db1461126c578063137c29fe146110755780632a2d80d114610db75780632b67b57014610bde57806330f28b7a14610ade5780633644e51514610a9d57806336c7851614610a285780633ff9dcb1146109a85780634fe02b441461093f57806365d9723c146107ac57806387517c451461067a578063927da105146105c3578063cc53287f146104a3578063edd9444b1461033a5763fe8ec1a7146100c657600080fd5b346103365760c07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126103365767ffffffffffffffff833581811161033257610114903690860161164b565b60243582811161032e5761012b903690870161161a565b6101336114e6565b9160843585811161032a5761014b9036908a016115c1565b98909560a43590811161032657610164913691016115c1565b969095815190610173826113ff565b606b82527f5065726d697442617463685769746e6573735472616e7366657246726f6d285460208301527f6f6b656e5065726d697373696f6e735b5d207065726d69747465642c61646472838301527f657373207370656e6465722c75696e74323536206e6f6e63652c75696e74323560608301527f3620646561646c696e652c000000000000000000000000000000000000000000608083015282519a8b9181610222602085018096611f93565b918237018a8152039961025b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe09b8c8101835282611437565b5190209085515161026b81611ebb565b908a5b8181106102f95750506102f6999a6102ed9183516102a081610294602082018095611f66565b03848101835282611437565b519020602089810151858b015195519182019687526040820192909252336060820152608081019190915260a081019390935260643560c08401528260e081015b03908101835282611437565b51902093611cf7565b80f35b8061031161030b610321938c5161175e565b51612054565b61031b828661175e565b52611f0a565b61026e565b8880fd5b8780fd5b8480fd5b8380fd5b5080fd5b5091346103365760807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126103365767ffffffffffffffff9080358281116103325761038b903690830161164b565b60243583811161032e576103a2903690840161161a565b9390926103ad6114e6565b9160643590811161049f576103c4913691016115c1565b949093835151976103d489611ebb565b98885b81811061047d5750506102f697988151610425816103f9602082018095611f66565b037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08101835282611437565b5190206020860151828701519083519260208401947ffcf35f5ac6a2c28868dc44c302166470266239195f02b0ee408334829333b7668652840152336060840152608083015260a082015260a081526102ed8161141b565b808b61031b8261049461030b61049a968d5161175e565b9261175e565b6103d7565b8680fd5b5082346105bf57602090817ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126103325780359067ffffffffffffffff821161032e576104f49136910161161a565b929091845b848110610504578580f35b8061051a610515600193888861196c565b61197c565b61052f84610529848a8a61196c565b0161197c565b3389528385528589209173ffffffffffffffffffffffffffffffffffffffff80911692838b528652868a20911690818a5285528589207fffffffffffffffffffffffff000000000000000000000000000000000000000081541690558551918252848201527f89b1add15eff56b3dfe299ad94e01f2b52fbcb80ae1a3baea6ae8c04cb2b98a4853392a2016104f9565b8280fd5b50346103365760607ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033657610676816105ff6114a0565b936106086114c3565b6106106114e6565b73ffffffffffffffffffffffffffffffffffffffff968716835260016020908152848420928816845291825283832090871683528152919020549251938316845260a083901c65ffffffffffff169084015260d09190911c604083015281906060820190565b0390f35b50346103365760807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc360112610336576106b26114a0565b906106bb6114c3565b916106c46114e6565b65ffffffffffff926064358481169081810361032a5779ffffffffffff0000000000000000000000000000000000000000947fda9fa7c1b00402c17d0161b249b1ab8bbec047c5a52207b9c112deffd817036b94338a5260016020527fffffffffffff0000000000000000000000000000000000000000000000000000858b209873ffffffffffffffffffffffffffffffffffffffff809416998a8d5260205283878d209b169a8b8d52602052868c209486156000146107a457504216925b8454921697889360a01b16911617179055815193845260208401523392a480f35b905092610783565b5082346105bf5760607ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf576107e56114a0565b906107ee6114c3565b9265ffffffffffff604435818116939084810361032a57338852602091600183528489209673ffffffffffffffffffffffffffffffffffffffff80911697888b528452858a20981697888a5283528489205460d01c93848711156109175761ffff9085840316116108f05750907f55eb90d810e1700b35a8e7e25395ff7f2b2259abd7415ca2284dfb1c246418f393929133895260018252838920878a528252838920888a5282528389209079ffffffffffffffffffffffffffffffffffffffffffffffffffff7fffffffffffff000000000000000000000000000000000000000000000000000083549260d01b16911617905582519485528401523392a480f35b84517f24d35a26000000000000000000000000000000000000000000000000000000008152fd5b5084517f756688fe000000000000000000000000000000000000000000000000000000008152fd5b503461033657807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc360112610336578060209273ffffffffffffffffffffffffffffffffffffffff61098f6114a0565b1681528084528181206024358252845220549051908152f35b5082346105bf57817ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf577f3704902f963766a4e561bbaab6e6cdc1b1dd12f6e9e99648da8843b3f46b918d90359160243533855284602052818520848652602052818520818154179055815193845260208401523392a280f35b8234610a9a5760807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc360112610a9a57610a606114a0565b610a686114c3565b610a706114e6565b6064359173ffffffffffffffffffffffffffffffffffffffff8316830361032e576102f6936117a1565b80fd5b503461033657817ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033657602090610ad7611b1e565b9051908152f35b508290346105bf576101007ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf57610b1a3661152a565b90807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7c36011261033257610b4c611478565b9160e43567ffffffffffffffff8111610bda576102f694610b6f913691016115c1565b939092610b7c8351612054565b6020840151828501519083519260208401947f939c21a48a8dbe3a9a2404a1d46691e4d39f6583d6ec6b35714604c986d801068652840152336060840152608083015260a082015260a08152610bd18161141b565b51902091611c25565b8580fd5b509134610336576101007ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033657610c186114a0565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdc360160c08112610332576080855191610c51836113e3565b1261033257845190610c6282611398565b73ffffffffffffffffffffffffffffffffffffffff91602435838116810361049f578152604435838116810361049f57602082015265ffffffffffff606435818116810361032a5788830152608435908116810361049f576060820152815260a435938285168503610bda576020820194855260c4359087830182815260e43567ffffffffffffffff811161032657610cfe90369084016115c1565b929093804211610d88575050918591610d786102f6999a610d7e95610d238851611fbe565b90898c511690519083519260208401947ff3841cd1ff0085026a6327b620b67997ce40f282c88a8e905a7a5626e310f3d086528401526060830152608082015260808152610d70816113ff565b519020611bd9565b916120c7565b519251169161199d565b602492508a51917fcd21db4f000000000000000000000000000000000000000000000000000000008352820152fd5b5091346103365760607ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc93818536011261033257610df36114a0565b9260249081359267ffffffffffffffff9788851161032a578590853603011261049f578051978589018981108282111761104a578252848301358181116103265785019036602383011215610326578382013591610e50836115ef565b90610e5d85519283611437565b838252602093878584019160071b83010191368311611046578801905b828210610fe9575050508a526044610e93868801611509565b96838c01978852013594838b0191868352604435908111610fe557610ebb90369087016115c1565b959096804211610fba575050508998995151610ed681611ebb565b908b5b818110610f9757505092889492610d7892610f6497958351610f02816103f98682018095611f66565b5190209073ffffffffffffffffffffffffffffffffffffffff9a8b8b51169151928551948501957faf1b0d30d2cab0380e68f0689007e3254993c596f2fdd0aaa7f4d04f794408638752850152830152608082015260808152610d70816113ff565b51169082515192845b848110610f78578580f35b80610f918585610f8b600195875161175e565b5161199d565b01610f6d565b80610311610fac8e9f9e93610fb2945161175e565b51611fbe565b9b9a9b610ed9565b8551917fcd21db4f000000000000000000000000000000000000000000000000000000008352820152fd5b8a80fd5b6080823603126110465785608091885161100281611398565b61100b85611509565b8152611018838601611509565b838201526110278a8601611607565b8a8201528d611037818701611607565b90820152815201910190610e7a565b8c80fd5b84896041867f4e487b7100000000000000000000000000000000000000000000000000000000835252fd5b5082346105bf576101407ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc3601126105bf576110b03661152a565b91807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7c360112610332576110e2611478565b67ffffffffffffffff93906101043585811161049f5761110590369086016115c1565b90936101243596871161032a57611125610bd1966102f6983691016115c1565b969095825190611134826113ff565b606482527f5065726d69745769746e6573735472616e7366657246726f6d28546f6b656e5060208301527f65726d697373696f6e73207065726d69747465642c6164647265737320737065848301527f6e6465722c75696e74323536206e6f6e63652c75696e7432353620646561646c60608301527f696e652c0000000000000000000000000000000000000000000000000000000060808301528351948591816111e3602085018096611f93565b918237018b8152039361121c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe095868101835282611437565b5190209261122a8651612054565b6020878101518589015195519182019687526040820192909252336060820152608081019190915260a081019390935260e43560c08401528260e081016102e1565b5082346105bf576020807ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc36011261033257813567ffffffffffffffff92838211610bda5736602383011215610bda5781013592831161032e576024906007368386831b8401011161049f57865b8581106112e5578780f35b80821b83019060807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdc83360301126103265761139288876001946060835161132c81611398565b611368608461133c8d8601611509565b9485845261134c60448201611509565b809785015261135d60648201611509565b809885015201611509565b918291015273ffffffffffffffffffffffffffffffffffffffff80808093169516931691166117a1565b016112da565b6080810190811067ffffffffffffffff8211176113b457604052565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6060810190811067ffffffffffffffff8211176113b457604052565b60a0810190811067ffffffffffffffff8211176113b457604052565b60c0810190811067ffffffffffffffff8211176113b457604052565b90601f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0910116810190811067ffffffffffffffff8211176113b457604052565b60c4359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b600080fd5b6004359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b6024359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b6044359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b359073ffffffffffffffffffffffffffffffffffffffff8216820361149b57565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc01906080821261149b576040805190611563826113e3565b8082941261149b57805181810181811067ffffffffffffffff8211176113b457825260043573ffffffffffffffffffffffffffffffffffffffff8116810361149b578152602435602082015282526044356020830152606435910152565b9181601f8401121561149b5782359167ffffffffffffffff831161149b576020838186019501011161149b57565b67ffffffffffffffff81116113b45760051b60200190565b359065ffffffffffff8216820361149b57565b9181601f8401121561149b5782359167ffffffffffffffff831161149b576020808501948460061b01011161149b57565b91909160608184031261149b576040805191611666836113e3565b8294813567ffffffffffffffff9081811161149b57830182601f8201121561149b578035611693816115ef565b926116a087519485611437565b818452602094858086019360061b8501019381851161149b579086899897969594939201925b8484106116e3575050505050855280820135908501520135910152565b90919293949596978483031261149b578851908982019082821085831117611730578a928992845261171487611509565b81528287013583820152815201930191908897969594936116c6565b602460007f4e487b710000000000000000000000000000000000000000000000000000000081526041600452fd5b80518210156117725760209160051b010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b92919273ffffffffffffffffffffffffffffffffffffffff604060008284168152600160205282828220961695868252602052818120338252602052209485549565ffffffffffff8760a01c16804211611884575082871696838803611812575b5050611810955016926118b5565b565b878484161160001461184f57602488604051907ff96fb0710000000000000000000000000000000000000000000000000000000082526004820152fd5b7fffffffffffffffffffffffff000000000000000000000000000000000000000084846118109a031691161790553880611802565b602490604051907fd81b2f2e0000000000000000000000000000000000000000000000000000000082526004820152fd5b9060006064926020958295604051947f23b872dd0000000000000000000000000000000000000000000000000000000086526004860152602485015260448401525af13d15601f3d116001600051141617161561190e57565b60646040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f5452414e534645525f46524f4d5f4641494c45440000000000000000000000006044820152fd5b91908110156117725760061b0190565b3573ffffffffffffffffffffffffffffffffffffffff8116810361149b5790565b9065ffffffffffff908160608401511673ffffffffffffffffffffffffffffffffffffffff908185511694826020820151169280866040809401511695169560009187835260016020528383208984526020528383209916988983526020528282209184835460d01c03611af5579185611ace94927fc6a377bfc4eb120024a8ac08eef205be16b817020812c73223e81d1bdb9708ec98979694508715600014611ad35779ffffffffffff00000000000000000000000000000000000000009042165b60a01b167fffffffffffff00000000000000000000000000000000000000000000000000006001860160d01b1617179055519384938491604091949373ffffffffffffffffffffffffffffffffffffffff606085019616845265ffffffffffff809216602085015216910152565b0390a4565b5079ffffffffffff000000000000000000000000000000000000000087611a60565b600484517f756688fe000000000000000000000000000000000000000000000000000000008152fd5b467f000000000000000000000000000000000000000000000000000000000000000003611b69577f000000000000000000000000000000000000000000000000000000000000000090565b60405160208101907f8cad95687ba82c2ce50e74f7b754645e5117c3a5bec8151c0726d5857980a86682527f9ac997416e8ff9d2ff6bebeb7149f65cdae5e32e2b90440b566bb3044041d36a604082015246606082015230608082015260808152611bd3816113ff565b51902090565b611be1611b1e565b906040519060208201927f190100000000000000000000000000000000000000000000000000000000000084526022830152604282015260428152611bd381611398565b9192909360a435936040840151804211611cc65750602084510151808611611c955750918591610d78611c6594611c60602088015186611e47565b611bd9565b73ffffffffffffffffffffffffffffffffffffffff809151511692608435918216820361149b57611810936118b5565b602490604051907f3728b83d0000000000000000000000000000000000000000000000000000000082526004820152fd5b602490604051907fcd21db4f0000000000000000000000000000000000000000000000000000000082526004820152fd5b959093958051519560409283830151804211611e175750848803611dee57611d2e918691610d7860209b611c608d88015186611e47565b60005b868110611d42575050505050505050565b611d4d81835161175e565b5188611d5a83878a61196c565b01359089810151808311611dbe575091818888886001968596611d84575b50505050505001611d31565b611db395611dad9273ffffffffffffffffffffffffffffffffffffffff6105159351169561196c565b916118b5565b803888888883611d78565b6024908651907f3728b83d0000000000000000000000000000000000000000000000000000000082526004820152fd5b600484517fff633a38000000000000000000000000000000000000000000000000000000008152fd5b6024908551907fcd21db4f0000000000000000000000000000000000000000000000000000000082526004820152fd5b9073ffffffffffffffffffffffffffffffffffffffff600160ff83161b9216600052600060205260406000209060081c6000526020526040600020818154188091551615611e9157565b60046040517f756688fe000000000000000000000000000000000000000000000000000000008152fd5b90611ec5826115ef565b611ed26040519182611437565b8281527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0611f0082946115ef565b0190602036910137565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114611f375760010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b805160208092019160005b828110611f7f575050505090565b835185529381019392810192600101611f71565b9081519160005b838110611fab575050016000815290565b8060208092840101518185015201611f9a565b60405160208101917f65626cad6cb96493bf6f5ebea28756c966f023ab9e8a83a7101849d5573b3678835273ffffffffffffffffffffffffffffffffffffffff8082511660408401526020820151166060830152606065ffffffffffff9182604082015116608085015201511660a082015260a0815260c0810181811067ffffffffffffffff8211176113b45760405251902090565b6040516020808201927f618358ac3db8dc274f0cd8829da7e234bd48cd73c4a740aede1adec9846d06a1845273ffffffffffffffffffffffffffffffffffffffff81511660408401520151606082015260608152611bd381611398565b919082604091031261149b576020823592013590565b6000843b61222e5750604182036121ac576120e4828201826120b1565b939092604010156117725760209360009360ff6040608095013560f81c5b60405194855216868401526040830152606082015282805260015afa156121a05773ffffffffffffffffffffffffffffffffffffffff806000511691821561217657160361214c57565b60046040517f815e1d64000000000000000000000000000000000000000000000000000000008152fd5b60046040517f8baa579f000000000000000000000000000000000000000000000000000000008152fd5b6040513d6000823e3d90fd5b60408203612204576121c0918101906120b1565b91601b7f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff84169360ff1c019060ff8211611f375760209360009360ff608094612102565b60046040517f4be6321b000000000000000000000000000000000000000000000000000000008152fd5b929391601f928173ffffffffffffffffffffffffffffffffffffffff60646020957fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0604051988997889687947f1626ba7e000000000000000000000000000000000000000000000000000000009e8f8752600487015260406024870152816044870152868601378b85828601015201168101030192165afa9081156123a857829161232a575b507fffffffff000000000000000000000000000000000000000000000000000000009150160361230057565b60046040517fb0669cbc000000000000000000000000000000000000000000000000000000008152fd5b90506020813d82116123a0575b8161234460209383611437565b810103126103365751907fffffffff0000000000000000000000000000000000000000000000000000000082168203610a9a57507fffffffff0000000000000000000000000000000000000000000000000000000090386122d4565b3d9150612337565b6040513d84823e3d90fdfea164736f6c6343000811000a",
}

// Permit2ABI is the input ABI used to generate the binding from.
// Deprecated: Use Permit2MetaData.ABI instead.
var Permit2ABI = Permit2MetaData.ABI

// Permit2Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Permit2MetaData.Bin instead.
var Permit2Bin = Permit2MetaData.Bin

// DeployPermit2 deploys a new Ethereum contract, binding an instance of Permit2 to it.
func DeployPermit2(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Permit2, error) {
	parsed, err := Permit2MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Permit2Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Permit2{Permit2Caller: Permit2Caller{contract: contract}, Permit2Transactor: Permit2Transactor{contract: contract}, Permit2Filterer: Permit2Filterer{contract: contract}}, nil
}

// Permit2 is an auto generated Go binding around an Ethereum contract.
type Permit2 struct {
	Permit2Caller     // Read-only binding to the contract
	Permit2Transactor // Write-only binding to the contract
	Permit2Filterer   // Log filterer for contract events
}

// Permit2Caller is an auto generated read-only Go binding around an Ethereum contract.
type Permit2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Permit2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Permit2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Permit2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Permit2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Permit2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Permit2Session struct {
	Contract     *Permit2          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Permit2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Permit2CallerSession struct {
	Contract *Permit2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// Permit2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Permit2TransactorSession struct {
	Contract     *Permit2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// Permit2Raw is an auto generated low-level Go binding around an Ethereum contract.
type Permit2Raw struct {
	Contract *Permit2 // Generic contract binding to access the raw methods on
}

// Permit2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Permit2CallerRaw struct {
	Contract *Permit2Caller // Generic read-only contract binding to access the raw methods on
}

// Permit2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Permit2TransactorRaw struct {
	Contract *Permit2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewPermit2 creates a new instance of Permit2, bound to a specific deployed contract.
func NewPermit2(address common.Address, backend bind.ContractBackend) (*Permit2, error) {
	contract, err := bindPermit2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Permit2{Permit2Caller: Permit2Caller{contract: contract}, Permit2Transactor: Permit2Transactor{contract: contract}, Permit2Filterer: Permit2Filterer{contract: contract}}, nil
}

// NewPermit2Caller creates a new read-only instance of Permit2, bound to a specific deployed contract.
func NewPermit2Caller(address common.Address, caller bind.ContractCaller) (*Permit2Caller, error) {
	contract, err := bindPermit2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Permit2Caller{contract: contract}, nil
}

// NewPermit2Transactor creates a new write-only instance of Permit2, bound to a specific deployed contract.
func NewPermit2Transactor(address common.Address, transactor bind.ContractTransactor) (*Permit2Transactor, error) {
	contract, err := bindPermit2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Permit2Transactor{contract: contract}, nil
}

// NewPermit2Filterer creates a new log filterer instance of Permit2, bound to a specific deployed contract.
func NewPermit2Filterer(address common.Address, filterer bind.ContractFilterer) (*Permit2Filterer, error) {
	contract, err := bindPermit2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Permit2Filterer{contract: contract}, nil
}

// bindPermit2 binds a generic wrapper to an already deployed contract.
func bindPermit2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(Permit2ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Permit2 *Permit2Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Permit2.Contract.Permit2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Permit2 *Permit2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Permit2.Contract.Permit2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Permit2 *Permit2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Permit2.Contract.Permit2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Permit2 *Permit2CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Permit2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Permit2 *Permit2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Permit2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Permit2 *Permit2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Permit2.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Permit2 *Permit2Caller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Permit2.contract.Call(opts, &out, "DOMAIN_SEPARATOR")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Permit2 *Permit2Session) DOMAINSEPARATOR() ([32]byte, error) {
	return _Permit2.Contract.DOMAINSEPARATOR(&_Permit2.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Permit2 *Permit2CallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _Permit2.Contract.DOMAINSEPARATOR(&_Permit2.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0x927da105.
//
// Solidity: function allowance(address , address , address ) view returns(uint160 amount, uint48 expiration, uint48 nonce)
func (_Permit2 *Permit2Caller) Allowance(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 common.Address) (struct {
	Amount     *big.Int
	Expiration *big.Int
	Nonce      *big.Int
}, error) {
	var out []interface{}
	err := _Permit2.contract.Call(opts, &out, "allowance", arg0, arg1, arg2)

	outstruct := new(struct {
		Amount     *big.Int
		Expiration *big.Int
		Nonce      *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Amount = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Expiration = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Nonce = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Allowance is a free data retrieval call binding the contract method 0x927da105.
//
// Solidity: function allowance(address , address , address ) view returns(uint160 amount, uint48 expiration, uint48 nonce)
func (_Permit2 *Permit2Session) Allowance(arg0 common.Address, arg1 common.Address, arg2 common.Address) (struct {
	Amount     *big.Int
	Expiration *big.Int
	Nonce      *big.Int
}, error) {
	return _Permit2.Contract.Allowance(&_Permit2.CallOpts, arg0, arg1, arg2)
}

// Allowance is a free data retrieval call binding the contract method 0x927da105.
//
// Solidity: function allowance(address , address , address ) view returns(uint160 amount, uint48 expiration, uint48 nonce)
func (_Permit2 *Permit2CallerSession) Allowance(arg0 common.Address, arg1 common.Address, arg2 common.Address) (struct {
	Amount     *big.Int
	Expiration *big.Int
	Nonce      *big.Int
}, error) {
	return _Permit2.Contract.Allowance(&_Permit2.CallOpts, arg0, arg1, arg2)
}

// NonceBitmap is a free data retrieval call binding the contract method 0x4fe02b44.
//
// Solidity: function nonceBitmap(address , uint256 ) view returns(uint256)
func (_Permit2 *Permit2Caller) NonceBitmap(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Permit2.contract.Call(opts, &out, "nonceBitmap", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NonceBitmap is a free data retrieval call binding the contract method 0x4fe02b44.
//
// Solidity: function nonceBitmap(address , uint256 ) view returns(uint256)
func (_Permit2 *Permit2Session) NonceBitmap(arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	return _Permit2.Contract.NonceBitmap(&_Permit2.CallOpts, arg0, arg1)
}

// NonceBitmap is a free data retrieval call binding the contract method 0x4fe02b44.
//
// Solidity: function nonceBitmap(address , uint256 ) view returns(uint256)
func (_Permit2 *Permit2CallerSession) NonceBitmap(arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	return _Permit2.Contract.NonceBitmap(&_Permit2.CallOpts, arg0, arg1)
}

// Approve is a paid mutator transaction binding the contract method 0x87517c45.
//
// Solidity: function approve(address token, address spender, uint160 amount, uint48 expiration) returns()
func (_Permit2 *Permit2Transactor) Approve(opts *bind.TransactOpts, token common.Address, spender common.Address, amount *big.Int, expiration *big.Int) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "approve", token, spender, amount, expiration)
}

// Approve is a paid mutator transaction binding the contract method 0x87517c45.
//
// Solidity: function approve(address token, address spender, uint160 amount, uint48 expiration) returns()
func (_Permit2 *Permit2Session) Approve(token common.Address, spender common.Address, amount *big.Int, expiration *big.Int) (*types.Transaction, error) {
	return _Permit2.Contract.Approve(&_Permit2.TransactOpts, token, spender, amount, expiration)
}

// Approve is a paid mutator transaction binding the contract method 0x87517c45.
//
// Solidity: function approve(address token, address spender, uint160 amount, uint48 expiration) returns()
func (_Permit2 *Permit2TransactorSession) Approve(token common.Address, spender common.Address, amount *big.Int, expiration *big.Int) (*types.Transaction, error) {
	return _Permit2.Contract.Approve(&_Permit2.TransactOpts, token, spender, amount, expiration)
}

// InvalidateNonces is a paid mutator transaction binding the contract method 0x65d9723c.
//
// Solidity: function invalidateNonces(address token, address spender, uint48 newNonce) returns()
func (_Permit2 *Permit2Transactor) InvalidateNonces(opts *bind.TransactOpts, token common.Address, spender common.Address, newNonce *big.Int) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "invalidateNonces", token, spender, newNonce)
}

// InvalidateNonces is a paid mutator transaction binding the contract method 0x65d9723c.
//
// Solidity: function invalidateNonces(address token, address spender, uint48 newNonce) returns()
func (_Permit2 *Permit2Session) InvalidateNonces(token common.Address, spender common.Address, newNonce *big.Int) (*types.Transaction, error) {
	return _Permit2.Contract.InvalidateNonces(&_Permit2.TransactOpts, token, spender, newNonce)
}

// InvalidateNonces is a paid mutator transaction binding the contract method 0x65d9723c.
//
// Solidity: function invalidateNonces(address token, address spender, uint48 newNonce) returns()
func (_Permit2 *Permit2TransactorSession) InvalidateNonces(token common.Address, spender common.Address, newNonce *big.Int) (*types.Transaction, error) {
	return _Permit2.Contract.InvalidateNonces(&_Permit2.TransactOpts, token, spender, newNonce)
}

// InvalidateUnorderedNonces is a paid mutator transaction binding the contract method 0x3ff9dcb1.
//
// Solidity: function invalidateUnorderedNonces(uint256 wordPos, uint256 mask) returns()
func (_Permit2 *Permit2Transactor) InvalidateUnorderedNonces(opts *bind.TransactOpts, wordPos *big.Int, mask *big.Int) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "invalidateUnorderedNonces", wordPos, mask)
}

// InvalidateUnorderedNonces is a paid mutator transaction binding the contract method 0x3ff9dcb1.
//
// Solidity: function invalidateUnorderedNonces(uint256 wordPos, uint256 mask) returns()
func (_Permit2 *Permit2Session) InvalidateUnorderedNonces(wordPos *big.Int, mask *big.Int) (*types.Transaction, error) {
	return _Permit2.Contract.InvalidateUnorderedNonces(&_Permit2.TransactOpts, wordPos, mask)
}

// InvalidateUnorderedNonces is a paid mutator transaction binding the contract method 0x3ff9dcb1.
//
// Solidity: function invalidateUnorderedNonces(uint256 wordPos, uint256 mask) returns()
func (_Permit2 *Permit2TransactorSession) InvalidateUnorderedNonces(wordPos *big.Int, mask *big.Int) (*types.Transaction, error) {
	return _Permit2.Contract.InvalidateUnorderedNonces(&_Permit2.TransactOpts, wordPos, mask)
}

// Lockdown is a paid mutator transaction binding the contract method 0xcc53287f.
//
// Solidity: function lockdown((address,address)[] approvals) returns()
func (_Permit2 *Permit2Transactor) Lockdown(opts *bind.TransactOpts, approvals []IAllowanceTransferTokenSpenderPair) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "lockdown", approvals)
}

// Lockdown is a paid mutator transaction binding the contract method 0xcc53287f.
//
// Solidity: function lockdown((address,address)[] approvals) returns()
func (_Permit2 *Permit2Session) Lockdown(approvals []IAllowanceTransferTokenSpenderPair) (*types.Transaction, error) {
	return _Permit2.Contract.Lockdown(&_Permit2.TransactOpts, approvals)
}

// Lockdown is a paid mutator transaction binding the contract method 0xcc53287f.
//
// Solidity: function lockdown((address,address)[] approvals) returns()
func (_Permit2 *Permit2TransactorSession) Lockdown(approvals []IAllowanceTransferTokenSpenderPair) (*types.Transaction, error) {
	return _Permit2.Contract.Lockdown(&_Permit2.TransactOpts, approvals)
}

// Permit is a paid mutator transaction binding the contract method 0x2a2d80d1.
//
// Solidity: function permit(address owner, ((address,uint160,uint48,uint48)[],address,uint256) permitBatch, bytes signature) returns()
func (_Permit2 *Permit2Transactor) Permit(opts *bind.TransactOpts, owner common.Address, permitBatch IAllowanceTransferPermitBatch, signature []byte) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "permit", owner, permitBatch, signature)
}

// Permit is a paid mutator transaction binding the contract method 0x2a2d80d1.
//
// Solidity: function permit(address owner, ((address,uint160,uint48,uint48)[],address,uint256) permitBatch, bytes signature) returns()
func (_Permit2 *Permit2Session) Permit(owner common.Address, permitBatch IAllowanceTransferPermitBatch, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.Permit(&_Permit2.TransactOpts, owner, permitBatch, signature)
}

// Permit is a paid mutator transaction binding the contract method 0x2a2d80d1.
//
// Solidity: function permit(address owner, ((address,uint160,uint48,uint48)[],address,uint256) permitBatch, bytes signature) returns()
func (_Permit2 *Permit2TransactorSession) Permit(owner common.Address, permitBatch IAllowanceTransferPermitBatch, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.Permit(&_Permit2.TransactOpts, owner, permitBatch, signature)
}

// Permit0 is a paid mutator transaction binding the contract method 0x2b67b570.
//
// Solidity: function permit(address owner, ((address,uint160,uint48,uint48),address,uint256) permitSingle, bytes signature) returns()
func (_Permit2 *Permit2Transactor) Permit0(opts *bind.TransactOpts, owner common.Address, permitSingle IAllowanceTransferPermitSingle, signature []byte) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "permit0", owner, permitSingle, signature)
}

// Permit0 is a paid mutator transaction binding the contract method 0x2b67b570.
//
// Solidity: function permit(address owner, ((address,uint160,uint48,uint48),address,uint256) permitSingle, bytes signature) returns()
func (_Permit2 *Permit2Session) Permit0(owner common.Address, permitSingle IAllowanceTransferPermitSingle, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.Permit0(&_Permit2.TransactOpts, owner, permitSingle, signature)
}

// Permit0 is a paid mutator transaction binding the contract method 0x2b67b570.
//
// Solidity: function permit(address owner, ((address,uint160,uint48,uint48),address,uint256) permitSingle, bytes signature) returns()
func (_Permit2 *Permit2TransactorSession) Permit0(owner common.Address, permitSingle IAllowanceTransferPermitSingle, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.Permit0(&_Permit2.TransactOpts, owner, permitSingle, signature)
}

// PermitTransferFrom is a paid mutator transaction binding the contract method 0x30f28b7a.
//
// Solidity: function permitTransferFrom(((address,uint256),uint256,uint256) permit, (address,uint256) transferDetails, address owner, bytes signature) returns()
func (_Permit2 *Permit2Transactor) PermitTransferFrom(opts *bind.TransactOpts, permit ISignatureTransferPermitTransferFrom, transferDetails ISignatureTransferSignatureTransferDetails, owner common.Address, signature []byte) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "permitTransferFrom", permit, transferDetails, owner, signature)
}

// PermitTransferFrom is a paid mutator transaction binding the contract method 0x30f28b7a.
//
// Solidity: function permitTransferFrom(((address,uint256),uint256,uint256) permit, (address,uint256) transferDetails, address owner, bytes signature) returns()
func (_Permit2 *Permit2Session) PermitTransferFrom(permit ISignatureTransferPermitTransferFrom, transferDetails ISignatureTransferSignatureTransferDetails, owner common.Address, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitTransferFrom(&_Permit2.TransactOpts, permit, transferDetails, owner, signature)
}

// PermitTransferFrom is a paid mutator transaction binding the contract method 0x30f28b7a.
//
// Solidity: function permitTransferFrom(((address,uint256),uint256,uint256) permit, (address,uint256) transferDetails, address owner, bytes signature) returns()
func (_Permit2 *Permit2TransactorSession) PermitTransferFrom(permit ISignatureTransferPermitTransferFrom, transferDetails ISignatureTransferSignatureTransferDetails, owner common.Address, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitTransferFrom(&_Permit2.TransactOpts, permit, transferDetails, owner, signature)
}

// PermitTransferFrom0 is a paid mutator transaction binding the contract method 0xedd9444b.
//
// Solidity: function permitTransferFrom(((address,uint256)[],uint256,uint256) permit, (address,uint256)[] transferDetails, address owner, bytes signature) returns()
func (_Permit2 *Permit2Transactor) PermitTransferFrom0(opts *bind.TransactOpts, permit ISignatureTransferPermitBatchTransferFrom, transferDetails []ISignatureTransferSignatureTransferDetails, owner common.Address, signature []byte) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "permitTransferFrom0", permit, transferDetails, owner, signature)
}

// PermitTransferFrom0 is a paid mutator transaction binding the contract method 0xedd9444b.
//
// Solidity: function permitTransferFrom(((address,uint256)[],uint256,uint256) permit, (address,uint256)[] transferDetails, address owner, bytes signature) returns()
func (_Permit2 *Permit2Session) PermitTransferFrom0(permit ISignatureTransferPermitBatchTransferFrom, transferDetails []ISignatureTransferSignatureTransferDetails, owner common.Address, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitTransferFrom0(&_Permit2.TransactOpts, permit, transferDetails, owner, signature)
}

// PermitTransferFrom0 is a paid mutator transaction binding the contract method 0xedd9444b.
//
// Solidity: function permitTransferFrom(((address,uint256)[],uint256,uint256) permit, (address,uint256)[] transferDetails, address owner, bytes signature) returns()
func (_Permit2 *Permit2TransactorSession) PermitTransferFrom0(permit ISignatureTransferPermitBatchTransferFrom, transferDetails []ISignatureTransferSignatureTransferDetails, owner common.Address, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitTransferFrom0(&_Permit2.TransactOpts, permit, transferDetails, owner, signature)
}

// PermitWitnessTransferFrom is a paid mutator transaction binding the contract method 0x137c29fe.
//
// Solidity: function permitWitnessTransferFrom(((address,uint256),uint256,uint256) permit, (address,uint256) transferDetails, address owner, bytes32 witness, string witnessTypeString, bytes signature) returns()
func (_Permit2 *Permit2Transactor) PermitWitnessTransferFrom(opts *bind.TransactOpts, permit ISignatureTransferPermitTransferFrom, transferDetails ISignatureTransferSignatureTransferDetails, owner common.Address, witness [32]byte, witnessTypeString string, signature []byte) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "permitWitnessTransferFrom", permit, transferDetails, owner, witness, witnessTypeString, signature)
}

// PermitWitnessTransferFrom is a paid mutator transaction binding the contract method 0x137c29fe.
//
// Solidity: function permitWitnessTransferFrom(((address,uint256),uint256,uint256) permit, (address,uint256) transferDetails, address owner, bytes32 witness, string witnessTypeString, bytes signature) returns()
func (_Permit2 *Permit2Session) PermitWitnessTransferFrom(permit ISignatureTransferPermitTransferFrom, transferDetails ISignatureTransferSignatureTransferDetails, owner common.Address, witness [32]byte, witnessTypeString string, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitWitnessTransferFrom(&_Permit2.TransactOpts, permit, transferDetails, owner, witness, witnessTypeString, signature)
}

// PermitWitnessTransferFrom is a paid mutator transaction binding the contract method 0x137c29fe.
//
// Solidity: function permitWitnessTransferFrom(((address,uint256),uint256,uint256) permit, (address,uint256) transferDetails, address owner, bytes32 witness, string witnessTypeString, bytes signature) returns()
func (_Permit2 *Permit2TransactorSession) PermitWitnessTransferFrom(permit ISignatureTransferPermitTransferFrom, transferDetails ISignatureTransferSignatureTransferDetails, owner common.Address, witness [32]byte, witnessTypeString string, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitWitnessTransferFrom(&_Permit2.TransactOpts, permit, transferDetails, owner, witness, witnessTypeString, signature)
}

// PermitWitnessTransferFrom0 is a paid mutator transaction binding the contract method 0xfe8ec1a7.
//
// Solidity: function permitWitnessTransferFrom(((address,uint256)[],uint256,uint256) permit, (address,uint256)[] transferDetails, address owner, bytes32 witness, string witnessTypeString, bytes signature) returns()
func (_Permit2 *Permit2Transactor) PermitWitnessTransferFrom0(opts *bind.TransactOpts, permit ISignatureTransferPermitBatchTransferFrom, transferDetails []ISignatureTransferSignatureTransferDetails, owner common.Address, witness [32]byte, witnessTypeString string, signature []byte) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "permitWitnessTransferFrom0", permit, transferDetails, owner, witness, witnessTypeString, signature)
}

// PermitWitnessTransferFrom0 is a paid mutator transaction binding the contract method 0xfe8ec1a7.
//
// Solidity: function permitWitnessTransferFrom(((address,uint256)[],uint256,uint256) permit, (address,uint256)[] transferDetails, address owner, bytes32 witness, string witnessTypeString, bytes signature) returns()
func (_Permit2 *Permit2Session) PermitWitnessTransferFrom0(permit ISignatureTransferPermitBatchTransferFrom, transferDetails []ISignatureTransferSignatureTransferDetails, owner common.Address, witness [32]byte, witnessTypeString string, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitWitnessTransferFrom0(&_Permit2.TransactOpts, permit, transferDetails, owner, witness, witnessTypeString, signature)
}

// PermitWitnessTransferFrom0 is a paid mutator transaction binding the contract method 0xfe8ec1a7.
//
// Solidity: function permitWitnessTransferFrom(((address,uint256)[],uint256,uint256) permit, (address,uint256)[] transferDetails, address owner, bytes32 witness, string witnessTypeString, bytes signature) returns()
func (_Permit2 *Permit2TransactorSession) PermitWitnessTransferFrom0(permit ISignatureTransferPermitBatchTransferFrom, transferDetails []ISignatureTransferSignatureTransferDetails, owner common.Address, witness [32]byte, witnessTypeString string, signature []byte) (*types.Transaction, error) {
	return _Permit2.Contract.PermitWitnessTransferFrom0(&_Permit2.TransactOpts, permit, transferDetails, owner, witness, witnessTypeString, signature)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x0d58b1db.
//
// Solidity: function transferFrom((address,address,uint160,address)[] transferDetails) returns()
func (_Permit2 *Permit2Transactor) TransferFrom(opts *bind.TransactOpts, transferDetails []IAllowanceTransferAllowanceTransferDetails) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "transferFrom", transferDetails)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x0d58b1db.
//
// Solidity: function transferFrom((address,address,uint160,address)[] transferDetails) returns()
func (_Permit2 *Permit2Session) TransferFrom(transferDetails []IAllowanceTransferAllowanceTransferDetails) (*types.Transaction, error) {
	return _Permit2.Contract.TransferFrom(&_Permit2.TransactOpts, transferDetails)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x0d58b1db.
//
// Solidity: function transferFrom((address,address,uint160,address)[] transferDetails) returns()
func (_Permit2 *Permit2TransactorSession) TransferFrom(transferDetails []IAllowanceTransferAllowanceTransferDetails) (*types.Transaction, error) {
	return _Permit2.Contract.TransferFrom(&_Permit2.TransactOpts, transferDetails)
}

// TransferFrom0 is a paid mutator transaction binding the contract method 0x36c78516.
//
// Solidity: function transferFrom(address from, address to, uint160 amount, address token) returns()
func (_Permit2 *Permit2Transactor) TransferFrom0(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int, token common.Address) (*types.Transaction, error) {
	return _Permit2.contract.Transact(opts, "transferFrom0", from, to, amount, token)
}

// TransferFrom0 is a paid mutator transaction binding the contract method 0x36c78516.
//
// Solidity: function transferFrom(address from, address to, uint160 amount, address token) returns()
func (_Permit2 *Permit2Session) TransferFrom0(from common.Address, to common.Address, amount *big.Int, token common.Address) (*types.Transaction, error) {
	return _Permit2.Contract.TransferFrom0(&_Permit2.TransactOpts, from, to, amount, token)
}

// TransferFrom0 is a paid mutator transaction binding the contract method 0x36c78516.
//
// Solidity: function transferFrom(address from, address to, uint160 amount, address token) returns()
func (_Permit2 *Permit2TransactorSession) TransferFrom0(from common.Address, to common.Address, amount *big.Int, token common.Address) (*types.Transaction, error) {
	return _Permit2.Contract.TransferFrom0(&_Permit2.TransactOpts, from, to, amount, token)
}

// Permit2ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Permit2 contract.
type Permit2ApprovalIterator struct {
	Event *Permit2Approval // Event containing the contract specifics and raw log

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
func (it *Permit2ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Permit2Approval)
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
		it.Event = new(Permit2Approval)
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
func (it *Permit2ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Permit2ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Permit2Approval represents a Approval event raised by the Permit2 contract.
type Permit2Approval struct {
	Owner      common.Address
	Token      common.Address
	Spender    common.Address
	Amount     *big.Int
	Expiration *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0xda9fa7c1b00402c17d0161b249b1ab8bbec047c5a52207b9c112deffd817036b.
//
// Solidity: event Approval(address indexed owner, address indexed token, address indexed spender, uint160 amount, uint48 expiration)
func (_Permit2 *Permit2Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, token []common.Address, spender []common.Address) (*Permit2ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Permit2.contract.FilterLogs(opts, "Approval", ownerRule, tokenRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &Permit2ApprovalIterator{contract: _Permit2.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0xda9fa7c1b00402c17d0161b249b1ab8bbec047c5a52207b9c112deffd817036b.
//
// Solidity: event Approval(address indexed owner, address indexed token, address indexed spender, uint160 amount, uint48 expiration)
func (_Permit2 *Permit2Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *Permit2Approval, owner []common.Address, token []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Permit2.contract.WatchLogs(opts, "Approval", ownerRule, tokenRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Permit2Approval)
				if err := _Permit2.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0xda9fa7c1b00402c17d0161b249b1ab8bbec047c5a52207b9c112deffd817036b.
//
// Solidity: event Approval(address indexed owner, address indexed token, address indexed spender, uint160 amount, uint48 expiration)
func (_Permit2 *Permit2Filterer) ParseApproval(log types.Log) (*Permit2Approval, error) {
	event := new(Permit2Approval)
	if err := _Permit2.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Permit2LockdownIterator is returned from FilterLockdown and is used to iterate over the raw logs and unpacked data for Lockdown events raised by the Permit2 contract.
type Permit2LockdownIterator struct {
	Event *Permit2Lockdown // Event containing the contract specifics and raw log

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
func (it *Permit2LockdownIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Permit2Lockdown)
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
		it.Event = new(Permit2Lockdown)
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
func (it *Permit2LockdownIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Permit2LockdownIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Permit2Lockdown represents a Lockdown event raised by the Permit2 contract.
type Permit2Lockdown struct {
	Owner   common.Address
	Token   common.Address
	Spender common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLockdown is a free log retrieval operation binding the contract event 0x89b1add15eff56b3dfe299ad94e01f2b52fbcb80ae1a3baea6ae8c04cb2b98a4.
//
// Solidity: event Lockdown(address indexed owner, address token, address spender)
func (_Permit2 *Permit2Filterer) FilterLockdown(opts *bind.FilterOpts, owner []common.Address) (*Permit2LockdownIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _Permit2.contract.FilterLogs(opts, "Lockdown", ownerRule)
	if err != nil {
		return nil, err
	}
	return &Permit2LockdownIterator{contract: _Permit2.contract, event: "Lockdown", logs: logs, sub: sub}, nil
}

// WatchLockdown is a free log subscription operation binding the contract event 0x89b1add15eff56b3dfe299ad94e01f2b52fbcb80ae1a3baea6ae8c04cb2b98a4.
//
// Solidity: event Lockdown(address indexed owner, address token, address spender)
func (_Permit2 *Permit2Filterer) WatchLockdown(opts *bind.WatchOpts, sink chan<- *Permit2Lockdown, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _Permit2.contract.WatchLogs(opts, "Lockdown", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Permit2Lockdown)
				if err := _Permit2.contract.UnpackLog(event, "Lockdown", log); err != nil {
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

// ParseLockdown is a log parse operation binding the contract event 0x89b1add15eff56b3dfe299ad94e01f2b52fbcb80ae1a3baea6ae8c04cb2b98a4.
//
// Solidity: event Lockdown(address indexed owner, address token, address spender)
func (_Permit2 *Permit2Filterer) ParseLockdown(log types.Log) (*Permit2Lockdown, error) {
	event := new(Permit2Lockdown)
	if err := _Permit2.contract.UnpackLog(event, "Lockdown", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Permit2NonceInvalidationIterator is returned from FilterNonceInvalidation and is used to iterate over the raw logs and unpacked data for NonceInvalidation events raised by the Permit2 contract.
type Permit2NonceInvalidationIterator struct {
	Event *Permit2NonceInvalidation // Event containing the contract specifics and raw log

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
func (it *Permit2NonceInvalidationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Permit2NonceInvalidation)
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
		it.Event = new(Permit2NonceInvalidation)
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
func (it *Permit2NonceInvalidationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Permit2NonceInvalidationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Permit2NonceInvalidation represents a NonceInvalidation event raised by the Permit2 contract.
type Permit2NonceInvalidation struct {
	Owner    common.Address
	Token    common.Address
	Spender  common.Address
	NewNonce *big.Int
	OldNonce *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterNonceInvalidation is a free log retrieval operation binding the contract event 0x55eb90d810e1700b35a8e7e25395ff7f2b2259abd7415ca2284dfb1c246418f3.
//
// Solidity: event NonceInvalidation(address indexed owner, address indexed token, address indexed spender, uint48 newNonce, uint48 oldNonce)
func (_Permit2 *Permit2Filterer) FilterNonceInvalidation(opts *bind.FilterOpts, owner []common.Address, token []common.Address, spender []common.Address) (*Permit2NonceInvalidationIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Permit2.contract.FilterLogs(opts, "NonceInvalidation", ownerRule, tokenRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &Permit2NonceInvalidationIterator{contract: _Permit2.contract, event: "NonceInvalidation", logs: logs, sub: sub}, nil
}

// WatchNonceInvalidation is a free log subscription operation binding the contract event 0x55eb90d810e1700b35a8e7e25395ff7f2b2259abd7415ca2284dfb1c246418f3.
//
// Solidity: event NonceInvalidation(address indexed owner, address indexed token, address indexed spender, uint48 newNonce, uint48 oldNonce)
func (_Permit2 *Permit2Filterer) WatchNonceInvalidation(opts *bind.WatchOpts, sink chan<- *Permit2NonceInvalidation, owner []common.Address, token []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Permit2.contract.WatchLogs(opts, "NonceInvalidation", ownerRule, tokenRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Permit2NonceInvalidation)
				if err := _Permit2.contract.UnpackLog(event, "NonceInvalidation", log); err != nil {
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

// ParseNonceInvalidation is a log parse operation binding the contract event 0x55eb90d810e1700b35a8e7e25395ff7f2b2259abd7415ca2284dfb1c246418f3.
//
// Solidity: event NonceInvalidation(address indexed owner, address indexed token, address indexed spender, uint48 newNonce, uint48 oldNonce)
func (_Permit2 *Permit2Filterer) ParseNonceInvalidation(log types.Log) (*Permit2NonceInvalidation, error) {
	event := new(Permit2NonceInvalidation)
	if err := _Permit2.contract.UnpackLog(event, "NonceInvalidation", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Permit2PermitIterator is returned from FilterPermit and is used to iterate over the raw logs and unpacked data for Permit events raised by the Permit2 contract.
type Permit2PermitIterator struct {
	Event *Permit2Permit // Event containing the contract specifics and raw log

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
func (it *Permit2PermitIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Permit2Permit)
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
		it.Event = new(Permit2Permit)
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
func (it *Permit2PermitIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Permit2PermitIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Permit2Permit represents a Permit event raised by the Permit2 contract.
type Permit2Permit struct {
	Owner      common.Address
	Token      common.Address
	Spender    common.Address
	Amount     *big.Int
	Expiration *big.Int
	Nonce      *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPermit is a free log retrieval operation binding the contract event 0xc6a377bfc4eb120024a8ac08eef205be16b817020812c73223e81d1bdb9708ec.
//
// Solidity: event Permit(address indexed owner, address indexed token, address indexed spender, uint160 amount, uint48 expiration, uint48 nonce)
func (_Permit2 *Permit2Filterer) FilterPermit(opts *bind.FilterOpts, owner []common.Address, token []common.Address, spender []common.Address) (*Permit2PermitIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Permit2.contract.FilterLogs(opts, "Permit", ownerRule, tokenRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &Permit2PermitIterator{contract: _Permit2.contract, event: "Permit", logs: logs, sub: sub}, nil
}

// WatchPermit is a free log subscription operation binding the contract event 0xc6a377bfc4eb120024a8ac08eef205be16b817020812c73223e81d1bdb9708ec.
//
// Solidity: event Permit(address indexed owner, address indexed token, address indexed spender, uint160 amount, uint48 expiration, uint48 nonce)
func (_Permit2 *Permit2Filterer) WatchPermit(opts *bind.WatchOpts, sink chan<- *Permit2Permit, owner []common.Address, token []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Permit2.contract.WatchLogs(opts, "Permit", ownerRule, tokenRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Permit2Permit)
				if err := _Permit2.contract.UnpackLog(event, "Permit", log); err != nil {
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

// ParsePermit is a log parse operation binding the contract event 0xc6a377bfc4eb120024a8ac08eef205be16b817020812c73223e81d1bdb9708ec.
//
// Solidity: event Permit(address indexed owner, address indexed token, address indexed spender, uint160 amount, uint48 expiration, uint48 nonce)
func (_Permit2 *Permit2Filterer) ParsePermit(log types.Log) (*Permit2Permit, error) {
	event := new(Permit2Permit)
	if err := _Permit2.contract.UnpackLog(event, "Permit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Permit2UnorderedNonceInvalidationIterator is returned from FilterUnorderedNonceInvalidation and is used to iterate over the raw logs and unpacked data for UnorderedNonceInvalidation events raised by the Permit2 contract.
type Permit2UnorderedNonceInvalidationIterator struct {
	Event *Permit2UnorderedNonceInvalidation // Event containing the contract specifics and raw log

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
func (it *Permit2UnorderedNonceInvalidationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Permit2UnorderedNonceInvalidation)
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
		it.Event = new(Permit2UnorderedNonceInvalidation)
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
func (it *Permit2UnorderedNonceInvalidationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Permit2UnorderedNonceInvalidationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Permit2UnorderedNonceInvalidation represents a UnorderedNonceInvalidation event raised by the Permit2 contract.
type Permit2UnorderedNonceInvalidation struct {
	Owner common.Address
	Word  *big.Int
	Mask  *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterUnorderedNonceInvalidation is a free log retrieval operation binding the contract event 0x3704902f963766a4e561bbaab6e6cdc1b1dd12f6e9e99648da8843b3f46b918d.
//
// Solidity: event UnorderedNonceInvalidation(address indexed owner, uint256 word, uint256 mask)
func (_Permit2 *Permit2Filterer) FilterUnorderedNonceInvalidation(opts *bind.FilterOpts, owner []common.Address) (*Permit2UnorderedNonceInvalidationIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _Permit2.contract.FilterLogs(opts, "UnorderedNonceInvalidation", ownerRule)
	if err != nil {
		return nil, err
	}
	return &Permit2UnorderedNonceInvalidationIterator{contract: _Permit2.contract, event: "UnorderedNonceInvalidation", logs: logs, sub: sub}, nil
}

// WatchUnorderedNonceInvalidation is a free log subscription operation binding the contract event 0x3704902f963766a4e561bbaab6e6cdc1b1dd12f6e9e99648da8843b3f46b918d.
//
// Solidity: event UnorderedNonceInvalidation(address indexed owner, uint256 word, uint256 mask)
func (_Permit2 *Permit2Filterer) WatchUnorderedNonceInvalidation(opts *bind.WatchOpts, sink chan<- *Permit2UnorderedNonceInvalidation, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _Permit2.contract.WatchLogs(opts, "UnorderedNonceInvalidation", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Permit2UnorderedNonceInvalidation)
				if err := _Permit2.contract.UnpackLog(event, "UnorderedNonceInvalidation", log); err != nil {
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

// ParseUnorderedNonceInvalidation is a log parse operation binding the contract event 0x3704902f963766a4e561bbaab6e6cdc1b1dd12f6e9e99648da8843b3f46b918d.
//
// Solidity: event UnorderedNonceInvalidation(address indexed owner, uint256 word, uint256 mask)
func (_Permit2 *Permit2Filterer) ParseUnorderedNonceInvalidation(log types.Log) (*Permit2UnorderedNonceInvalidation, error) {
	event := new(Permit2UnorderedNonceInvalidation)
	if err := _Permit2.contract.UnpackLog(event, "UnorderedNonceInvalidation", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
