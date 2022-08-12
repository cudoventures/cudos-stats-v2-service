package main

import (
	"fmt"
	"net/http"
	"time"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/config"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/handlers"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/rest/bank"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/storage"
	"github.com/CudoVentures/cudos-stats-v2-service/internal/tasks"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/forbole/juno/v2/node/remote"
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.NewConfig("config.yaml")
	if err != nil {
		log.Fatal().Err(fmt.Errorf("creating config failed: %s", err)).Send()
		return
	}

	encodingConfig := makeEncodingConfig([]module.BasicManager{
		cudosapp.ModuleBasics,
	})()

	nodeClient, err := remote.NewNode(&cfg.Cudos.NodeDetails, encodingConfig.Marshaler)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("error while creating node client: %s", err)).Send()
		return
	}

	source, err := remote.NewSource(cfg.Cudos.NodeDetails.GRPC)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("error while creating remote source: %s", err)).Send()
		return
	}

	stakingClient := stakingtypes.NewQueryClient(source.GrpcConn)
	bankingRestClient := bank.NewRestClient(cfg.Cudos.REST.Address)

	keyValueStorage := storage.NewStorage()

	log.Info().Msg("Executing tasks")

	if err := tasks.ExecuteTasks(cfg, nodeClient, stakingClient, bankingRestClient, keyValueStorage); err != nil {
		log.Fatal().Err(fmt.Errorf("error while executing tasks: %s", err)).Send()
		return
	}

	log.Info().Msg("Registering tasks")
	scheduler := gocron.NewScheduler(time.UTC)

	if err := tasks.RegisterTasks(scheduler, cfg, nodeClient, stakingClient, bankingRestClient, keyValueStorage); err != nil {
		log.Fatal().Err(fmt.Errorf("error while registering tasks: %s", err)).Send()
		return
	}

	scheduler.StartAsync()

	log.Info().Msg("Registering http handlers")

	http.HandleFunc("/cosmos/mint/v1beta1/annual_provisions", handlers.GetAnnualProvisionsHandler(cfg, keyValueStorage))
	http.HandleFunc("/cosmos/mint/v1beta1/inflation", handlers.GetInflationHandler(cfg, keyValueStorage))
	http.HandleFunc("/cosmos/mint/v1beta1/params", handlers.GetParamsHandler(cfg))
	http.HandleFunc("/cosmos/bank/v1beta1/supply", handlers.GetSupplyHandler(cfg, keyValueStorage))
	http.HandleFunc("/circulating-supply", handlers.GetCircSupplyTextHandler(cfg, keyValueStorage))
	http.HandleFunc("/json/circulating-supply", handlers.GetCircSupplyJSONHandler(cfg, keyValueStorage))
	http.HandleFunc("/stats", handlers.GetStatsHandler(cfg, keyValueStorage))

	log.Info().Msg(fmt.Sprintf("Listening on port: %d", cfg.Port))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
		log.Fatal().Err(fmt.Errorf("error while listening: %s", err))
	}
}

func makeEncodingConfig(managers []module.BasicManager) func() params.EncodingConfig {
	return func() params.EncodingConfig {
		encodingConfig := params.MakeTestEncodingConfig()
		std.RegisterLegacyAminoCodec(encodingConfig.Amino)
		std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		manager := mergeBasicManagers(managers)
		manager.RegisterLegacyAminoCodec(encodingConfig.Amino)
		manager.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		return encodingConfig
	}
}

func mergeBasicManagers(managers []module.BasicManager) module.BasicManager {
	var union = module.BasicManager{}
	for _, manager := range managers {
		for k, v := range manager {
			union[k] = v
		}
	}
	return union
}
