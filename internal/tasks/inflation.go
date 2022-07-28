package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	cudoMintTypes "github.com/CudoVentures/cudos-node/x/cudoMint/types"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/forbole/juno/v2/node/remote"
)

func getCalculateInflationHandler(genesisState cudoMintTypes.GenesisState, cfg config.Config, nodeClient *remote.Node, bankingClient banktypes.QueryClient, storage keyValueStorage) func() error {
	return func() error {
		client, err := ethclient.Dial(cfg.Eth.EthNode)
		if err != nil {
			return fmt.Errorf("failed to dial eth node: %s", err)
		}

		latestEthBlock, err := getLatestEthBlock(client)
		if err != nil {
			return fmt.Errorf("faield to get latest eth block: %s", err)
		}

		currentTotalBalance, err := getEthAccountsBalanceAtBlock(client, cfg.Eth.TokenAddress, cfg.Eth.EthAccounts, latestEthBlock)
		if err != nil {
			return fmt.Errorf("failed to get eth accounts balance: %s", err)
		}

		inflationStartBlock := latestEthBlock
		inflationStartBlock.Sub(latestEthBlock, big.NewInt(inflationSinceDays*ethBlocksPerDay))

		startTotalBalance, err := getEthAccountsBalanceAtBlock(client, cfg.Eth.TokenAddress, cfg.Eth.EthAccounts, inflationStartBlock)
		if err != nil {
			return fmt.Errorf("failed to get eth accounts balance: %s", err)
		}

		latestBlockHeight, err := nodeClient.LatestHeight()
		if err != nil {
			return fmt.Errorf("failed to get last block height %s", err)
		}

		startBlockHeight := int64(0)
		inflationSinceDays := int64(inflationSinceDays)

		// TODO: This can be removed after the chain is working for more than INFLATION_SINCE_DAYS

		for startBlockHeight < 1 {
			startBlockHeight = latestBlockHeight - (inflationSinceDays * genesisState.Params.BlocksPerDay.Int64())
			inflationSinceDays--
		}

		mintAmountInt, err := calculateMintedTokensSinceHeight(genesisState, cfg.Genesis.InitialHeight, startBlockHeight, float64(inflationSinceDays))
		if err != nil {
			return fmt.Errorf("failed to calculated minted tokens: %s", err)
		}

		startTotalBalanceInt, ok := sdk.NewIntFromString(startTotalBalance.String())
		if !ok {
			return fmt.Errorf("failed to convert big.Int to sdk.Int: %s", startTotalBalance.String())
		}

		startTotalSupply, _ := sdk.NewIntFromString(maxSupply)
		startTotalSupply = startTotalSupply.Sub(startTotalBalanceInt)

		currentTotalBalanceInt, ok := sdk.NewIntFromString(currentTotalBalance.String())
		if !ok {
			return fmt.Errorf("failed to convert big.Int to sdk.Int: %s", currentTotalBalance.String())
		}

		currentTotalSupply, _ := sdk.NewIntFromString(maxSupply)
		currentTotalSupply = currentTotalSupply.Sub(currentTotalBalanceInt).Add(mintAmountInt)

		inflation := currentTotalSupply.Sub(startTotalSupply).ToDec().Quo(startTotalSupply.ToDec())

		if err := storage.SetValue(cfg.Storage.InflationKey, inflation.String()); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", inflation.String(), cfg.Storage.InflationKey)
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
		defer cancelFunc()

		totalSupply, err := bankingClient.TotalSupply(remote.GetHeightRequestContext(ctx, latestBlockHeight), &banktypes.QueryTotalSupplyRequest{})
		if err != nil {
			return fmt.Errorf("error while getting total supply: %s", err)
		}

		for i := 0; i < len(totalSupply.Supply); i++ {
			if totalSupply.Supply[i].Denom == cfg.Genesis.MintDenom {
				totalSupply.Supply[i].Amount = currentTotalSupply
			}
		}

		totalSupplyJSON, err := json.Marshal(totalSupply)
		if err != nil {
			return fmt.Errorf("error while convering supply to JSON: %s", err)
		}

		if err := storage.SetValue(cfg.Storage.AllTokensSupplyKey, string(totalSupplyJSON)); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", currentTotalSupply.String(), cfg.Storage.AllTokensSupplyKey)
		}

		if err := storage.SetValue(cfg.Storage.SupplyKey, currentTotalSupply.String()); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", currentTotalSupply.String(), cfg.Storage.SupplyKey)
		}

		return nil
	}
}
