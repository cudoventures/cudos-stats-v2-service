package config

import (
	"os"

	"github.com/forbole/juno/v2/node/remote"
	"gopkg.in/yaml.v2"
)

func NewConfig(configPath string) (Config, error) {
	config := Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return config, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	if err := d.Decode(&config); err != nil {
		return config, err
	}

	return config, nil
}

type Config struct {
	Port    int `yaml:"port"`
	Genesis struct {
		InitialHeight         int64  `yaml:"initial_height"`
		NormTimePassed        string `yaml:"norm_time_passed"`
		BlocksPerDay          string `yaml:"blocks_per_day"`
		MintDenom             string `yaml:"mint_denom"`
		GravityAccountAddress string `yaml:"gravity_account_address"`
	} `yaml:"genesis"`
	Cudos struct {
		NodeDetails remote.Details `yaml:"node"`
		REST        struct {
			Address string `yaml:"address"`
		} `yaml:"rest"`
	} `yaml:"cudos"`
	Eth struct {
		EthNode      string   `yaml:"node"`
		TokenAddress string   `yaml:"token_address"`
		EthAccounts  []string `yaml:"accounts"`
	} `yaml:"eth"`
	Storage struct {
		APRKey              string `yaml:"apr_key"`
		AnnualProvisionsKey string `yaml:"annual_provisions"`
		InflationKey        string `yaml:"inflation_key"`
		AllTokensSupplyKey  string `yaml:"all_tokens_supply_key"`
		SupplyKey           string `yaml:"supply_key"`
	} `yaml:"storage"`
}
