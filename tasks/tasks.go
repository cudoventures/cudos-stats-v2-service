package tasks

import (
	"context"
	"fmt"
	"math/big"
	"time"

	cudoMintTypes "github.com/CudoVentures/cudos-node/x/cudoMint/types"
	"github.com/CudoVentures/cudos-stats-v2-service/config"
	"github.com/CudoVentures/cudos-stats-v2-service/erc20"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/forbole/juno/v2/node/remote"
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
)

func ExecuteTasks(cfg config.Config, nodeClient *remote.Node, stakingClient stakingtypes.QueryClient, bankingClient banktypes.QueryClient, storage keyValueStorage) error {
	genesisState, err := createGenesisState(cfg)
	if err != nil {
		return err
	}

	if err := getCalculateInflationHandler(*genesisState, cfg, nodeClient, bankingClient, storage)(); err != nil {
		return fmt.Errorf("inflation calculation failed: %s", err)
	}

	if err := getCalculateAPRHandler(*genesisState, cfg, nodeClient, stakingClient, storage)(); err != nil {
		return fmt.Errorf("apr calculation failed: %s", err)
	}

	return nil
}

func RegisterTasks(cfg config.Config, nodeClient *remote.Node, stakingClient stakingtypes.QueryClient, bankingClient banktypes.QueryClient, storage keyValueStorage) error {
	scheduler := gocron.NewScheduler(time.UTC)

	genesisState, err := createGenesisState(cfg)
	if err != nil {
		return err
	}

	if _, err := scheduler.Every(1).Day().At("00:00").Do(func() {
		watchMethod(getCalculateInflationHandler(*genesisState, cfg, nodeClient, bankingClient, storage))
		watchMethod(getCalculateAPRHandler(*genesisState, cfg, nodeClient, stakingClient, storage))
	}); err != nil {
		return fmt.Errorf("scheduler failed to register tasks: %s", err)
	}

	return nil
}

func getLatestEthBlock(client *ethclient.Client) (*big.Int, error) {
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest eth block: %s", err)
	}

	return header.Number, nil
}

func getEthAccountsBalanceAtBlock(client *ethclient.Client, tokenAddress string, accounts []string, block *big.Int) (*big.Int, error) {
	instance, err := erc20.NewTokenCaller(common.HexToAddress(tokenAddress), client)
	if err != nil {
		return nil, err
	}

	totalBalance := big.NewInt(0)

	for _, account := range accounts {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		balance, err := instance.BalanceOf(&bind.CallOpts{
			BlockNumber: block,
			Context:     ctx,
		}, common.HexToAddress(account))

		if err != nil {
			return nil, err
		}

		totalBalance.Add(totalBalance, balance)
	}

	return totalBalance, nil
}

func calculateMintedTokensSinceHeight(mintParams cudoMintTypes.GenesisState, genesisInitialHeight, sinceBlock int64, periodDays float64) (sdk.Int, error) {
	minter := mintParams.Minter
	params := mintParams.Params

	if minter.NormTimePassed.GT(finalNormTimePassed) {
		return sdk.NewInt(0), nil
	}

	minter.NormTimePassed = updateNormTimePassed(mintParams, genesisInitialHeight, sinceBlock)

	mintAmountInt := sdk.NewInt(0)
	totalBlocks := int64(float64(mintParams.Params.BlocksPerDay.Int64()) * periodDays)

	for height := int64(1); height <= totalBlocks; height++ {
		if minter.NormTimePassed.GT(finalNormTimePassed) {
			break
		}

		incr := normalizeBlockHeightInc(params.BlocksPerDay)
		mintAmountDec := calculateMintedCoins(minter, incr)
		mintAmountInt = mintAmountInt.Add(mintAmountDec.TruncateInt())
		minter.NormTimePassed = minter.NormTimePassed.Add(incr)
	}

	return mintAmountInt, nil
}

func updateNormTimePassed(mintParams cudoMintTypes.GenesisState, initialBlockHeight, lastBlockHeight int64) sdk.Dec {
	// TODO: Cannot be saved at this moment because of the changes in inflation calculation
	// storage := workers.NewWorkersStorage(db, "cudomint")
	// valueStr, err := storage.GetOrDefaultValue(calculateInflationLastBlock, strconv.FormatInt(initialBlockHeight, 10))
	// if err != nil {
	// 	return sdk.Dec{}, fmt.Errorf("failed to get %s", calculateInflationLastBlock)
	// }

	// value, err := strconv.ParseInt(valueStr, 10, 64)
	// if err != nil {
	// 	return sdk.Dec{}, fmt.Errorf("failed to parse %s", calculateInflationLastBlock)
	// }

	for initialBlockHeight < lastBlockHeight {
		inc := normalizeBlockHeightInc(mintParams.Params.BlocksPerDay)
		mintParams.Minter.NormTimePassed = mintParams.Minter.NormTimePassed.Add(inc)
		initialBlockHeight++
	}

	// if err := db.SaveMintParams(&mintParams); err != nil {
	// 	return sdk.Dec{}, fmt.Errorf("failed to save mint params: %s", err)
	// }

	// if err := storage.SetValue(calculateInflationLastBlock, strconv.FormatInt(lastBlockHeight, 10)); err != nil {
	// 	return sdk.Dec{}, fmt.Errorf("failed to save %s: %s", calculateInflationLastBlock, err)
	// }

	return mintParams.Minter.NormTimePassed
}

// Normalize block height incrementation
func normalizeBlockHeightInc(incrementModifier sdk.Int) sdk.Dec {
	totalBlocks := incrementModifier.Mul(totalDays)
	return (sdk.NewDec(1).QuoInt(totalBlocks)).Mul(finalNormTimePassed)
}

// Integral of f(t) is 0,6 * t^3  - 26.5 * t^2 + 358 * t
// The function extrema is ~10.48 so after that the function is decreasing
func calculateIntegral(t sdk.Dec) sdk.Dec {
	return (zeroPointSix.Mul(t.Power(3))).Sub(twentySixPointFive.Mul(t.Power(2))).Add(sdk.NewDec(358).Mul(t))
}

func calculateMintedCoins(minter cudoMintTypes.Minter, increment sdk.Dec) sdk.Dec {
	prevStep := calculateIntegral(sdk.MinDec(minter.NormTimePassed, finalNormTimePassed))
	nextStep := calculateIntegral(sdk.MinDec(minter.NormTimePassed.Add(increment), finalNormTimePassed))
	return (nextStep.Sub(prevStep)).Mul(sdk.NewDec(10).Power(24)) // formula calculates in mil of cudos + converting to acudos
}

func createGenesisState(cfg config.Config) (*cudoMintTypes.GenesisState, error) {
	normTimePassed, err := sdk.NewDecFromStr(cfg.Genesis.NormTimePassed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NormTimePassed %s: %s", cfg.Genesis.NormTimePassed, err)
	}

	blocksPerDay, ok := sdk.NewIntFromString(cfg.Genesis.BlocksPerDay)
	if !ok {
		return nil, fmt.Errorf("failed to parse BlocksPerDay %s", cfg.Genesis.BlocksPerDay)
	}

	return cudoMintTypes.NewGenesisState(cudoMintTypes.NewMinter(sdk.NewDec(0), normTimePassed), cudoMintTypes.NewParams(blocksPerDay)), nil
}

func watchMethod(method func() error) {
	go func() {
		err := method()
		if err != nil {
			log.Error().Err(err).Send()
		}
	}()
}

var (
	// based on the assumption that we have 1 block per 5 seconds
	// if actual blocks are generated at slower rate then the network will mint tokens more than 3652 days (~10 years)
	totalDays           = sdk.NewInt(3652) // Hardcoded to 10 years
	finalNormTimePassed = sdk.NewDec(10)
	zeroPointSix        = sdk.MustNewDecFromStr("0.6")
	twentySixPointFive  = sdk.MustNewDecFromStr("26.5")
	// calculateInflationLastBlock = "CalculateInflationLastBlock"
)

const ethBlocksPerDay = 5760
const inflationSinceDays = 30 * 3
const maxSupply = "10000000000000000000000000000" // 10 billion

type keyValueStorage interface {
	SetValue(key, value string) error
	GetOrDefaultValue(key, defaultValue string) (string, error)
}
