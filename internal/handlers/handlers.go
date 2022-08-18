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
			badRequest(w, err)
			return
		}

		formattedSupply, err := formatSupply(supply)
		if err != nil {
			badRequest(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(formattedSupply)); err != nil {
			badRequest(w, err)
			return
		}
	}
}

func GetCircSupplyJSONHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		supply, err := storage.GetValue(cfg.Storage.SupplyKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		formattedSupply, err := formatSupply(supply)
		if err != nil {
			badRequest(w, err)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(supplyResponse{Supply: formattedSupply}); err != nil {
			badRequest(w, err)
		}
	}
}

func GetStatsHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		supply, err := storage.GetValue(cfg.Storage.SupplyKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		formattedSupply, err := formatSupply(supply)
		if err != nil {
			badRequest(w, err)
			return
		}

		supplyHeight, err := storage.GetInt64Value(cfg.Storage.SupplyHeightKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		inflation, err := storage.GetValue(cfg.Storage.InflationKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		inflationHeight, err := storage.GetInt64Value(cfg.Storage.InflationHeightKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		apr, err := storage.GetValue(cfg.Storage.APRKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		aprHeight, err := storage.GetInt64Value(cfg.Storage.APRHeightKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(statsResponse{
			Inflation: valueAtHeight{Value: inflation, Height: inflationHeight},
			APR:       valueAtHeight{Value: apr, Height: aprHeight},
			Supply:    valueAtHeight{Value: formattedSupply, Height: supplyHeight},
		}); err != nil {
			badRequest(w, err)
		}
	}
}

func GetSupplyHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		supply, err := storage.GetValue(cfg.Storage.AllTokensSupplyKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		setHeaders(w)

		if _, err := w.Write([]byte(supply)); err != nil {
			badRequest(w, err)
		}
	}
}

func GetAPRHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		apr, err := storage.GetValue(cfg.Storage.APRKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(aprResponse{APR: apr}); err != nil {
			badRequest(w, err)
		}
	}
}

func GetAnnualProvisionsHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		annualProvisions, err := storage.GetValue(cfg.Storage.AnnualProvisionsKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(annualProvisionsResponse{AnnualProvisions: annualProvisions}); err != nil {
			badRequest(w, err)
		}
	}
}

func GetInflationHandler(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		inflation, err := storage.GetValue(cfg.Storage.InflationKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		setHeaders(w)

		if err := json.NewEncoder(w).Encode(inflationResponse{Inflation: inflation}); err != nil {
			badRequest(w, err)
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
			badRequest(w, err)
		}
	}
}

func GetCudosNetworkTotalSupply(cfg config.Config, storage keyValueStorage) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		supply, err := storage.GetValue(cfg.Storage.CudosNetworkTotalSupplyKey)
		if err != nil {
			badRequest(w, err)
			return
		}

		formattedSupply, err := formatSupply(supply)
		if err != nil {
			badRequest(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(formattedSupply)); err != nil {
			badRequest(w, err)
			return
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

func badRequest(w http.ResponseWriter, err error) {
	log.Error().Err(err).Send()
	w.WriteHeader(http.StatusBadRequest)
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

type statsResponse struct {
	Inflation valueAtHeight `json:"inflation"`
	APR       valueAtHeight `json:"apr"`
	Supply    valueAtHeight `json:"supply"`
}

type valueAtHeight struct {
	Value  string `json:"value"`
	Height int64  `json:"height"`
}

type keyValueStorage interface {
	SetValue(key, value string) error
	GetValue(key string) (string, error)
	GetInt64Value(key string) (int64, error)
}
