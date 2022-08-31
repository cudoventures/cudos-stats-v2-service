package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	cudoMintTypes "github.com/CudoVentures/cudos-node/x/cudoMint/types"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/forbole/juno/v2/node/remote"
)

func getCalculateAPRHandler(genesisState cudoMintTypes.GenesisState, cfg config.Config, nodeClient *remote.Node, stakingClient stakingtypes.QueryClient,
	distClient distributionQueryClient, storage keyValueStorage) func() error {

	return func() error {
		if genesisState.Minter.NormTimePassed.GT(finalNormTimePassed) {
			return nil
		}

		if nodeClient == nil {
			return errors.New("node client is null")
		}

		latestBlockHeight, err := nodeClient.LatestHeight()
		if err != nil {
			return fmt.Errorf("failed to get last block height %s", err)
		}

		mintAmountInt, err := calculateMintedTokensSinceHeight(genesisState, cfg.APRGenesis.InitialHeight, latestBlockHeight, 30.43)
		if err != nil {
			return fmt.Errorf("failed to calculated minted tokens: %s", err)
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
		defer cancelFunc()

		res, err := stakingClient.Pool(remote.GetHeightRequestContext(ctx, latestBlockHeight), &stakingtypes.QueryPoolRequest{})
		if err != nil {
			return fmt.Errorf("failed to get bonded_tokens: %s", err)
		}

		apr := mintAmountInt.ToDec().Quo(res.Pool.BondedTokens.ToDec()).MulInt64(int64(12))

		parametersResponse, err := distClient.GetParams(ctx)
		if err != nil {
			return fmt.Errorf("failed to get distribution parameters: %s", err)
		}

		communityTax, err := sdk.NewDecFromStr(parametersResponse.CommunityTax)
		if err != nil {
			return fmt.Errorf("failed to parse community tax (%s): %s", parametersResponse.CommunityTax, err)
		}

		communityTaxPortion := sdk.NewDec(1).Sub(communityTax)

		if communityTaxPortion.GT(sdk.NewDec(0)) {
			apr = apr.Mul(communityTaxPortion)
		}

		if err := storage.SetValue(cfg.Storage.APRKey, apr.String()); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", apr.String(), cfg.Storage.APRKey)
		}

		if err := storage.SetInt64Value(cfg.Storage.APRHeightKey, latestBlockHeight); err != nil {
			return fmt.Errorf("failed to set value %d for key %s", latestBlockHeight, cfg.Storage.APRHeightKey)
		}

		annualProvisions := mintAmountInt.ToDec().MulInt64(12)

		if err := storage.SetValue(cfg.Storage.AnnualProvisionsKey, annualProvisions.String()); err != nil {
			return fmt.Errorf("failed to set value %s for key %s", annualProvisions.String(), cfg.Storage.AnnualProvisionsKey)
		}

		return nil
	}
}
