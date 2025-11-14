package validations

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

var validateFunc = w3.MustNewFunc("validate((address proxyAdminAddress,address systemConfigAddress,bytes32 absolutePrestate,uint256 chainID) input,bool allowFailure)", "string")

var validateFuncOpcmValidator = w3.MustNewFunc("validate((address sysCfg,bytes32 absolutePrestate,uint256 l2ChainID,address proposer) input,bool allowFailure)", "string")

type validateFuncArgs struct {
	ProxyAdminAddress   common.Address
	SystemConfigAddress common.Address
	AbsolutePrestate    common.Hash
	ChainID             *big.Int
}

type validateFuncArgsOpcmValidator struct {
	SysCfg           common.Address `w3:"sysCfg"`
	AbsolutePrestate common.Hash    `w3:"absolutePrestate"`
	L2ChainID        *big.Int       `w3:"l2ChainID"`
	Proposer         common.Address `w3:"proposer"`
}

type Validator interface {
	Validate(ctx context.Context, input BaseValidatorInput) ([]string, error)
}

type BaseValidator struct {
	client  *rpc.Client
	release string
}

type OPCMStandardValidator struct {
	client        *rpc.Client
	validatorAddr common.Address
}

type BaseValidatorInput struct {
	ProxyAdminAddress   common.Address
	SystemConfigAddress common.Address
	AbsolutePrestate    common.Hash
	L2ChainID           *big.Int
	Proposer            common.Address
}

func newBaseValidator(client *rpc.Client, release string) *BaseValidator {
	return &BaseValidator{client: client, release: release}
}

func (v *BaseValidator) Validate(ctx context.Context, input BaseValidatorInput) ([]string, error) {
	l1ChainID, err := ethclient.NewClient(v.client).ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	validatorAddr, err := ValidatorAddress(l1ChainID.Uint64(), v.release)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator address: %w", err)
	}

	var rawOutput []byte
	if err := w3.NewClient(v.client).CallCtx(
		ctx,
		eth.Call(&w3types.Message{
			To:   &validatorAddr,
			Func: validateFunc,
			Args: []any{
				validateFuncArgs{
					ProxyAdminAddress:   input.ProxyAdminAddress,
					SystemConfigAddress: input.SystemConfigAddress,
					AbsolutePrestate:    input.AbsolutePrestate,
					ChainID:             input.L2ChainID,
				},
				true,
			},
		}, nil, nil).Returns(&rawOutput),
	); err != nil {
		return nil, fmt.Errorf("failed to call validate: %w", err)
	}

	var output string
	if err := validateFunc.DecodeReturns(rawOutput, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}
	return parseErrors(output), nil
}

func (v *OPCMStandardValidator) Validate(ctx context.Context, input BaseValidatorInput) ([]string, error) {
	if input.Proposer == (common.Address{}) {
		return nil, fmt.Errorf("proposer address is required for OPCMStandardValidator")
	}

	if v.validatorAddr == (common.Address{}) {
		return nil, fmt.Errorf("validator address is required for OPCMStandardValidator")
	}

	var rawOutput []byte
	if err := w3.NewClient(v.client).CallCtx(
		ctx,
		eth.Call(&w3types.Message{
			To:   &v.validatorAddr,
			Func: validateFuncOpcmValidator,
			Args: []any{
				validateFuncArgsOpcmValidator{
					SysCfg:           input.SystemConfigAddress,
					AbsolutePrestate: input.AbsolutePrestate,
					L2ChainID:        input.L2ChainID,
					Proposer:         input.Proposer,
				},
				true,
			},
		}, nil, nil).Returns(&rawOutput),
	); err != nil {
		return nil, fmt.Errorf("failed to call validate: %w", err)
	}

	var output string
	if err := validateFuncOpcmValidator.DecodeReturns(rawOutput, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}
	return parseErrors(output), nil
}

func NewOPCMStandardValidator(client *rpc.Client, validatorAddr common.Address) *OPCMStandardValidator {
	return &OPCMStandardValidator{
		client:        client,
		validatorAddr: validatorAddr,
	}
}

func NewV180Validator(client *rpc.Client) *BaseValidator {
	return newBaseValidator(client, standard.ContractsV180Tag)
}

func NewV200Validator(client *rpc.Client) *BaseValidator {
	return newBaseValidator(client, standard.ContractsV200Tag)
}

func NewV300Validator(client *rpc.Client) *BaseValidator {
	return newBaseValidator(client, standard.ContractsV300Tag)
}

func NewV400Validator(client *rpc.Client) *BaseValidator {
	return newBaseValidator(client, standard.ContractsV400Tag)
}

func NewV410Validator(client *rpc.Client) *BaseValidator {
	return newBaseValidator(client, standard.ContractsV410Tag)
}

func NewV500Validator(client *rpc.Client) *BaseValidator {
	return newBaseValidator(client, standard.ContractsV500Tag)
}

func parseErrors(output string) []string {
	if idx := strings.Index(output, ":"); idx != -1 && strings.HasPrefix(output, "Chain") {
		output = output[idx+1:]
	}
	parts := strings.Split(output, ",")
	var errors []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			errors = append(errors, part)
		}
	}
	return errors
}
