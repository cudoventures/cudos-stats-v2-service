package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/CudoVentures/cudos-stats-v2-service/internal/config"
	"github.com/rs/zerolog/log"
)

func GetCircSupplyTextHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		supply, err := storage.GetValue(cfg.Storage.SupplyKey)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		formattedSupply, err := formatSupply(supply)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(formattedSupply)); err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

func GetCircSupplyJSONHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		supply, err := storage.GetValue(cfg.Storage.SupplyKey)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		formattedSupply, err := formatSupply(supply)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(supplyResponse{Supply: formattedSupply}); err != nil {
			log.Error().Err(err).Send()
		}
	}
}

func GetSupplyHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		supply, err := storage.GetValue(cfg.Storage.AllTokensSupplyKey)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		setHeaders(w)

		if _, err := w.Write([]byte(supply)); err != nil {
			log.Error().Err(err).Send()
		}
	}
}

func GetAPRHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		apr, err := storage.GetValue(cfg.Storage.APRKey)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(aprResponse{APR: apr}); err != nil {
			log.Error().Err(err).Send()
		}
	}
}

func GetAnnualProvisionsHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		annualProvisions, err := storage.GetValue(cfg.Storage.AnnualProvisionsKey)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(annualProvisionsResponse{AnnualProvisions: annualProvisions}); err != nil {
			log.Error().Err(err).Send()
		}
	}
}

func GetInflationHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		inflation, err := storage.GetValue(cfg.Storage.InflationKey)
		if err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(inflationResponse{Inflation: inflation}); err != nil {
			log.Error().Err(err).Send()
		}
	}
}

func GetParamsHandler(cfg config.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		setHeaders(w)

		if err := json.NewEncoder(w).Encode(paramsResponse{
			Params: params{
				MintDenom:           cfg.Genesis.MintDenom,
				InflationRateChange: "0.0",
				InflationMax:        "0.0",
				InflationMin:        "0.0",
				GoalBonded:          "0.0",
				BlocksPerYear:       cfg.Genesis.BlocksPerDay,
			},
		}); err != nil {
			log.Error().Err(err).Send()
		}
	}
}

func formatSupply(supply string) (string, error) {
	bigSupply, ok := new(big.Int).SetString(supply, 10)
	if !ok || bigSupply == nil {
		return "", fmt.Errorf("failed to convert %s to big.Int", supply)
	}

	divisor := new(big.Int).SetInt64(1000000000000000000)
	formattedSupply := new(big.Int).SetInt64(0)
	formattedSupply.Div(bigSupply, divisor)

	return formattedSupply.String(), nil
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

type aprResponse struct {
	APR string `json:"apr"`
}

type paramsResponse struct {
	Params params `json:"params"`
}

type params struct {
	MintDenom           string `json:"mint_denom"`
	InflationRateChange string `json:"inflation_rate_change"`
	InflationMax        string `json:"inflation_max"`
	InflationMin        string `json:"inflation_min"`
	GoalBonded          string `json:"goal_bonded"`
	BlocksPerYear       string `json:"blocks_per_year"`
}

type inflationResponse struct {
	Inflation string `json:"inflation"`
}

type annualProvisionsResponse struct {
	AnnualProvisions string `json:"annual_provisions"`
}

type supplyResponse struct {
	Supply string `json:"supply"`
}

type keyValueStorage interface {
	SetValue(key, value string) error
	GetValue(key string) (string, error)
}
