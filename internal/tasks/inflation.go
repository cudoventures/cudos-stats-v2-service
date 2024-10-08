package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	cudoMintTypes "github.com/CudoVentures/cudos-node/x/cudoMint/types"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/config"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/rest/bank"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/forbole/juno/v2/node/remote"
)

func getCalculateInflationHandler(genesisState cudoMintTypes.GenesisState, cfg config.Config, nodeClient *remote.Node, bankingClient bankQueryClient, storage keyValueStorage) func() error {
	return func() error {
		//client, err := ethclient.Dial(cfg.Eth.EthNode)
		//if err != nil {
		//	return fmt.Errorf("failed to dial eth node: %s", err)
		//}

		//latestEthBlock, err := getLatestEthBlock(client)
		//if err != nil {
		//	return fmt.Errorf("faield to get latest eth block: %s", err)
		//}

		//inflationEthStartBlock := big.NewInt(latestEthBlock.Int64() - (cfg.Calculation.InflationSinceDays * ethBlocksPerDay))

		//ethStartSupply, err := getEthCirculatingSupplyAtHeight(inflationEthStartBlock, client, cfg)
		//if err != nil {
		//	return err
		//}

		// ethCurrentSupply, err := getEthCirculatingSupplyAtHeight(latestEthBlock, client, cfg)
		// if err != nil {
		// 	return err
		// }

		latestCudosBlock, err := nodeClient.LatestHeight()
		if err != nil {
			return fmt.Errorf("failed to get last block height %s", err)
		}

		inflationCudosStartBlock := int64(0)
		inflationSinceDaysValue := int64(cfg.Calculation.InflationSinceDays)

		// TODO: This can be removed after the chain is working for more than INFLATION_SINCE_DAYS

		for inflationCudosStartBlock < 1 {
			inflationCudosStartBlock = latestCudosBlock - (inflationSinceDaysValue * genesisState.Params.BlocksPerDay.Int64())
			inflationSinceDaysValue--
		}

		//cudosStartSupply, err := getCudosNetworkCirculatingSupplyAtHeight(inflationCudosStartBlock, bankingClient, cfg)
		//if err != nil {
		//	return err
		//}

		cudosCurrentSupply, err := getCudosNetworkCirculatingSupplyAtHeight(latestCudosBlock, bankingClient, cfg)
		if err != nil {
			return err
		}

		//startTotalSupply := ethStartSupply.Add(cudosStartSupply)
		// currentTotalSupply := ethCurrentSupply.Add(cudosCurrentSupply)
		currentTotalSupply := cudosCurrentSupply.Sub(sdk.NewIntWithDecimal(1942421346, 18))

		//inflation := currentTotalSupply.Sub(startTotalSupply).ToDec().Quo(startTotalSupply.ToDec())
		inflation := sdk.MustNewDecFromStr("0.01")

		if err := storage.SetValue(cfg.Storage.InflationKey, inflation.String()); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", inflation.String(), cfg.Storage.InflationKey)
		}

		if err := storage.SetInt64Value(cfg.Storage.InflationHeightKey, latestCudosBlock); err != nil {
			return fmt.Errorf("failed to set value %d for key %s", latestCudosBlock, cfg.Storage.InflationHeightKey)
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
		defer cancelFunc()

		totalSupply, err := bankingClient.GetTotalSupply(ctx, latestCudosBlock)
		if err != nil {
			return fmt.Errorf("error while getting total supply: %s", err)
		}

		var cudosNetworkTotalSupply sdk.Int

		for i := 0; i < len(totalSupply.Supply); i++ {
			if totalSupply.Supply[i].Denom == cfg.InflationGenesis.MintDenom {
				cudosNetworkTotalSupply = totalSupply.Supply[i].Amount
				totalSupply.Supply[i].Amount = currentTotalSupply
			}
		}

		totalSupplyJSON, err := json.Marshal(totalSupply)
		if err != nil {
			return fmt.Errorf("error while convering supply to JSON: %s", err)
		}

		if err := storage.SetValue(cfg.Storage.AllTokensSupplyKey, string(totalSupplyJSON)); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", string(totalSupplyJSON), cfg.Storage.AllTokensSupplyKey)
		}

		if err := storage.SetValue(cfg.Storage.SupplyKey, currentTotalSupply.String()); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", currentTotalSupply.String(), cfg.Storage.SupplyKey)
		}

		if err := storage.SetInt64Value(cfg.Storage.SupplyHeightKey, latestCudosBlock); err != nil {
			return fmt.Errorf("failed to set value %d for key %s", latestCudosBlock, cfg.Storage.SupplyHeightKey)
		}

		if err := storage.SetValue(cfg.Storage.CudosNetworkTotalSupplyKey, cudosNetworkTotalSupply.String()); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", cudosNetworkTotalSupply.String(), cfg.Storage.CudosNetworkTotalSupplyKey)
		}

		return nil
	}
}

func getEthCirculatingSupplyAtHeight(height *big.Int, client *ethclient.Client, cfg config.Config) (sdk.Int, error) {

	ethAccountsBalance, err := getEthAccountsBalanceAtBlock(client, cfg.Eth.TokenAddress, cfg.Eth.EthAccounts, height)
	if err != nil {
		return sdk.Int{}, fmt.Errorf("failed to get eth accounts balance: %s", err)
	}

	ethAccountsBalanceInt, ok := sdk.NewIntFromString(ethAccountsBalance.String())
	if !ok {
		return sdk.Int{}, fmt.Errorf("failed to convert big.Int to sdk.Int: %s", ethAccountsBalance.String())
	}

	totalSupply, _ := sdk.NewIntFromString(maxSupply)

	return totalSupply.Sub(ethAccountsBalanceInt), nil
}

func getCudosNetworkCirculatingSupplyAtHeight(height int64, bankingClient bankQueryClient, cfg config.Config) (sdk.Int, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFunc()

	var totalSupply bank.TotalSupplyResponse
	var err error

	// TODO: Just hack to iterate until you reach block with non-empty supply, should be removed when we are not accessing so old blocks

	for {
		totalSupply, err = bankingClient.GetTotalSupply(ctx, height)
		if err != nil {
			return sdk.Int{}, fmt.Errorf("error while getting total supply: %s", err)
		}

		if len(totalSupply.Supply) != 0 {
			break
		}

		height += 1
	}

	var gravityModuleBalance sdk.Coin

	for {
		gravityModuleBalance, err = bankingClient.GetBalance(ctx, height, cfg.InflationGenesis.GravityAccountAddress, cfg.InflationGenesis.MintDenom)
		if err != nil {
			return sdk.Int{}, fmt.Errorf("error while getting %s balance: %s", cfg.InflationGenesis.GravityAccountAddress, err)
		}

		if !gravityModuleBalance.Amount.IsZero() {
			break
		}

		height += 1
	}

	for i := 0; i < len(totalSupply.Supply); i++ {
		if totalSupply.Supply[i].Denom == cfg.InflationGenesis.MintDenom {
			return totalSupply.Supply[i].Amount, nil
		}
	}

	return sdk.Int{}, fmt.Errorf("invalid total supply %+v", totalSupply)
}
