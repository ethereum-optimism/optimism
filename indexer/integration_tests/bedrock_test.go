package integration_tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/legacy"
	"github.com/ethereum-optimism/optimism/indexer/services/l1"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"

	_ "github.com/lib/pq"
)

func TestBedrockIndexer(t *testing.T) {
	dbParams := createTestDB(t)

	cfg := op_e2e.DefaultSystemConfig(t)
	cfg.DeployConfig.FinalizationPeriodSeconds = 2
	sys, err := cfg.Start()
	require.NoError(t, err)
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Client := sys.Clients["sequencer"]
	fromAddr := cfg.Secrets.Addresses().Alice

	// wait a couple of blocks
	require.NoError(t, e2eutils.WaitBlock(e2eutils.TimeoutCtx(t, 30*time.Second), l2Client, 10))

	l1SB, err := bindings.NewL1StandardBridge(predeploys.DevL1StandardBridgeAddr, l1Client)
	require.NoError(t, err)
	l2SB, err := bindings.NewL2StandardBridge(predeploys.L2StandardBridgeAddr, l2Client)
	require.NoError(t, err)
	portal, err := bindings.NewOptimismPortal(predeploys.DevOptimismPortalAddr, l1Client)
	require.NoError(t, err)
	l1Opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L1ChainIDBig())
	require.NoError(t, err)
	l2Opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L2ChainIDBig())
	require.NoError(t, err)

	idxrCfg := legacy.Config{
		ChainID:                        cfg.DeployConfig.L1ChainID,
		L1EthRpc:                       sys.Nodes["l1"].HTTPEndpoint(),
		L2EthRpc:                       sys.Nodes["sequencer"].HTTPEndpoint(),
		PollInterval:                   time.Second,
		DBHost:                         dbParams.Host,
		DBPort:                         dbParams.Port,
		DBUser:                         dbParams.User,
		DBPassword:                     dbParams.Password,
		DBName:                         dbParams.Name,
		LogLevel:                       "info",
		LogTerminal:                    true,
		L1StartBlockNumber:             0,
		L1ConfDepth:                    1,
		L2ConfDepth:                    1,
		MaxHeaderBatchSize:             2,
		RESTHostname:                   "127.0.0.1",
		RESTPort:                       7980,
		DisableIndexer:                 false,
		Bedrock:                        true,
		BedrockL1StandardBridgeAddress: cfg.DeployConfig.L1StandardBridgeProxy,
		BedrockOptimismPortalAddress:   cfg.DeployConfig.OptimismPortalProxy,
	}
	idxr, err := legacy.NewIndexer(idxrCfg)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- idxr.Start()
	}()

	t.Cleanup(func() {
		idxr.Stop()
		require.NoError(t, <-errCh)
	})

	makeURL := func(path string) string {
		return fmt.Sprintf("http://%s:%d/%s", idxrCfg.RESTHostname, idxrCfg.RESTPort, path)
	}

	t.Run("deposit ETH", func(t *testing.T) {
		l1Opts.Value = big.NewInt(params.Ether)
		depTx, err := l1SB.DepositETH(l1Opts, 200_000, nil)
		require.NoError(t, err)
		depReceipt, err := e2eutils.WaitReceiptOK(e2eutils.TimeoutCtx(t, 10*time.Second), l1Client, depTx.Hash())
		require.NoError(t, err)
		require.Greaterf(t, len(depReceipt.Logs), 0, "must have logs")
		var l2Hash common.Hash
		for _, eLog := range depReceipt.Logs {
			if len(eLog.Topics) == 0 || eLog.Topics[0] != derive.DepositEventABIHash {
				continue
			}

			depLog, err := derive.UnmarshalDepositLogEvent(eLog)
			require.NoError(t, err)
			tx := types.NewTx(depLog)
			l2Hash = tx.Hash()
		}
		require.NotEqual(t, common.Hash{}, l2Hash)
		_, err = e2eutils.WaitReceiptOK(e2eutils.TimeoutCtx(t, 15*time.Second), l2Client, l2Hash)
		require.NoError(t, err)

		// Poll for indexer deposit
		var depPage *db.PaginatedDeposits
		require.NoError(t, e2eutils.WaitFor(e2eutils.TimeoutCtx(t, 30*time.Second), 100*time.Millisecond, func() (bool, error) {
			res := new(db.PaginatedDeposits)
			err := getJSON(makeURL(fmt.Sprintf("v1/deposits/%s", fromAddr)), res)
			if err != nil {
				return false, err
			}

			if len(res.Deposits) == 0 {
				return false, nil
			}

			depPage = res
			return true, nil
		}))

		// Make sure deposit is what we expect
		require.Equal(t, 1, len(depPage.Deposits))
		deposit := depPage.Deposits[0]
		require.Equal(t, big.NewInt(params.Ether).String(), deposit.Amount)
		require.Equal(t, depTx.Hash().String(), deposit.TxHash)
		require.Equal(t, depReceipt.BlockNumber.Uint64(), deposit.BlockNumber)
		require.Equal(t, fromAddr.String(), deposit.FromAddress)
		require.Equal(t, fromAddr.String(), deposit.ToAddress)
		require.EqualValues(t, db.ETHL1Token, deposit.L1Token)
		require.Equal(t, l1.ZeroAddress.String(), deposit.L2Token)
		require.NotEmpty(t, deposit.GUID)

		// Perform withdrawal through bridge
		l2Opts.Value = big.NewInt(0.5 * params.Ether)
		wdTx, err := l2SB.Withdraw(l2Opts, predeploys.LegacyERC20ETHAddr, big.NewInt(0.5*params.Ether), 0, nil)
		require.NoError(t, err)
		wdReceipt, err := e2eutils.WaitReceiptOK(e2eutils.TimeoutCtx(t, 30*time.Second), l2Client, wdTx.Hash())
		require.NoError(t, err)

		var wdPage *db.PaginatedWithdrawals
		require.NoError(t, e2eutils.WaitFor(e2eutils.TimeoutCtx(t, 30*time.Second), 100*time.Millisecond, func() (bool, error) {
			res := new(db.PaginatedWithdrawals)
			err := getJSON(makeURL(fmt.Sprintf("v1/withdrawals/%s", fromAddr)), res)
			if err != nil {
				return false, err
			}

			if len(res.Withdrawals) == 0 {
				return false, nil
			}

			wdPage = res
			return true, nil
		}))

		require.Equal(t, 1, len(wdPage.Withdrawals))
		withdrawal := wdPage.Withdrawals[0]
		require.Nil(t, withdrawal.BedrockProvenTxHash)
		require.Nil(t, withdrawal.BedrockFinalizedTxHash)
		require.Equal(t, big.NewInt(0.5*params.Ether).String(), withdrawal.Amount)
		require.Equal(t, wdTx.Hash().String(), withdrawal.TxHash)
		require.Equal(t, wdReceipt.BlockNumber.Uint64(), withdrawal.BlockNumber)
		// use fromaddr twice here because the user is withdrawing
		// to themselves
		require.Equal(t, fromAddr.String(), withdrawal.FromAddress)
		require.Equal(t, fromAddr.String(), withdrawal.ToAddress)
		require.EqualValues(t, l1.ZeroAddress.String(), withdrawal.L1Token)
		require.Equal(t, db.ETHL2Token, withdrawal.L2Token)
		require.NotEmpty(t, withdrawal.GUID)

		finBlockNum, err := withdrawals.WaitForFinalizationPeriod(
			e2eutils.TimeoutCtx(t, time.Minute),
			l1Client,
			predeploys.DevOptimismPortalAddr,
			wdReceipt.BlockNumber,
		)
		require.NoError(t, err)
		finHeader, err := l2Client.HeaderByNumber(context.Background(), big.NewInt(int64(finBlockNum)))
		require.NoError(t, err)

		rpcClient, err := rpc.Dial(sys.Nodes["sequencer"].HTTPEndpoint())
		require.NoError(t, err)
		proofCl := gethclient.New(rpcClient)
		receiptCl := ethclient.NewClient(rpcClient)
		oracle, err := bindings.NewL2OutputOracleCaller(predeploys.DevL2OutputOracleAddr, l1Client)
		require.Nil(t, err)
		wParams, err := withdrawals.ProveWithdrawalParameters(context.Background(), proofCl, receiptCl, wdTx.Hash(), finHeader, oracle)
		require.NoError(t, err)

		l1Opts.Value = big.NewInt(0)
		withdrawalTx := bindings.TypesWithdrawalTransaction{
			Nonce:    wParams.Nonce,
			Sender:   wParams.Sender,
			Target:   wParams.Target,
			Value:    wParams.Value,
			GasLimit: wParams.GasLimit,
			Data:     wParams.Data,
		}

		// Prove our withdrawal
		proveTx, err := portal.ProveWithdrawalTransaction(
			l1Opts,
			withdrawalTx,
			wParams.L2OutputIndex,
			wParams.OutputRootProof,
			wParams.WithdrawalProof,
		)
		require.NoError(t, err)

		proveReceipt, err := e2eutils.WaitReceiptOK(e2eutils.TimeoutCtx(t, time.Minute), l1Client, proveTx.Hash())
		require.NoError(t, err)

		wdPage = nil
		require.NoError(t, e2eutils.WaitFor(e2eutils.TimeoutCtx(t, 30*time.Second), 100*time.Millisecond, func() (bool, error) {
			res := new(db.PaginatedWithdrawals)
			err := getJSON(makeURL(fmt.Sprintf("v1/withdrawals/%s", fromAddr)), res)
			if err != nil {
				return false, err
			}

			if res.Withdrawals[0].BedrockProvenTxHash == nil {
				return false, nil
			}

			wdPage = res
			return true, nil
		}))

		wd := wdPage.Withdrawals[0]
		require.Equal(t, proveReceipt.TxHash.String(), *wd.BedrockProvenTxHash)
		require.Nil(t, wd.BedrockFinalizedTxHash)

		// Wait for the finalization period to elapse
		_, err = withdrawals.WaitForFinalizationPeriod(
			e2eutils.TimeoutCtx(t, time.Minute),
			l1Client,
			predeploys.DevOptimismPortalAddr,
			finHeader.Number,
		)
		require.NoError(t, err)

		// Send our finalize withdrawal transaction
		finTx, err := portal.FinalizeWithdrawalTransaction(
			l1Opts,
			withdrawalTx,
		)
		require.NoError(t, err)

		finReceipt, err := e2eutils.WaitReceiptOK(e2eutils.TimeoutCtx(t, time.Minute), l1Client, finTx.Hash())
		require.NoError(t, err)

		wdPage = nil
		require.NoError(t, e2eutils.WaitFor(e2eutils.TimeoutCtx(t, 30*time.Second), 100*time.Millisecond, func() (bool, error) {
			res := new(db.PaginatedWithdrawals)
			err := getJSON(makeURL(fmt.Sprintf("v1/withdrawals/%s", fromAddr)), res)
			if err != nil {
				return false, err
			}

			if res.Withdrawals[0].BedrockFinalizedTxHash == nil {
				return false, nil
			}

			wdPage = res
			return true, nil
		}))

		wd = wdPage.Withdrawals[0]
		require.Equal(t, proveReceipt.TxHash.String(), *wd.BedrockProvenTxHash)
		require.Equal(t, finReceipt.TxHash.String(), *wd.BedrockFinalizedTxHash)
		require.True(t, *wd.BedrockFinalizedSuccess)

		wdPage = new(db.PaginatedWithdrawals)
		err = getJSON(makeURL(fmt.Sprintf("v1/withdrawals/%s?finalized=false", fromAddr)), wdPage)
		require.NoError(t, err)
		require.Equal(t, 0, len(wdPage.Withdrawals))
	})
}

type testDBParams struct {
	Host     string
	Port     uint64
	User     string
	Password string
	Name     string
}

func createTestDB(t *testing.T) *testDBParams {
	user := os.Getenv("DB_USER")
	name := fmt.Sprintf("indexer_test_%d", time.Now().Unix())

	dsn := "postgres://"
	if user != "" {
		dsn += user
		dsn += "@"
	}
	dsn += "localhost:5432?sslmode=disable"
	pg, err := sql.Open(
		"postgres",
		dsn,
	)
	require.NoError(t, err)

	_, err = pg.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err = pg.Exec("DROP DATABASE " + name)
		require.NoError(t, err)
		pg.Close()
	})

	return &testDBParams{
		Host: "localhost",
		Port: 5432,
		Name: name,
		User: user,
	}
}

func getJSON(url string, out interface{}) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("non-200 status code %d", res.StatusCode)
	}

	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	return dec.Decode(out)
}
