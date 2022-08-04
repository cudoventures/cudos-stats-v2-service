package bank

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	sdkTypes "github.com/cosmos/cosmos-sdk/types"
)

type bankRESTClient struct {
	Url string
}

func NewRestClient(url string) *bankRESTClient {
	return &bankRESTClient{Url: url}
}

func (brc bankRESTClient) GetTotalSupply(ctx context.Context, height int64) (TotalSupplyResponse, error) {
	respStr, err := brc.get(ctx, "/bank/total", height)
	if err != nil {
		return TotalSupplyResponse{}, err
	}

	var res totalSupplyResult
	if err := json.Unmarshal([]byte(respStr), &res); err != nil {
		return TotalSupplyResponse{}, err
	}

	return res.Result, nil
}

func (brc bankRESTClient) GetBalance(ctx context.Context, height int64, address, denom string) (sdkTypes.Coin, error) {
	respStr, err := brc.get(ctx, fmt.Sprintf("/bank/balances/%s", address), height)
	if err != nil {
		return sdkTypes.Coin{}, err
	}

	var res balanceResponse
	if err := json.Unmarshal([]byte(respStr), &res); err != nil {
		return sdkTypes.Coin{}, err
	}

	if len(res.Result) == 0 {
		return sdkTypes.Coin{}, nil
	}

	for _, balance := range res.Result {
		if balance.Denom == denom {
			return balance, nil
		}
	}

	return sdkTypes.Coin{}, fmt.Errorf("denom %s not found in %+v", denom, res)
}

func (brc bankRESTClient) get(ctx context.Context, uri string, height int64) (string, error) {
	getReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s%s?height=%d", brc.Url, uri, height), nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type balanceResponse struct {
	Result sdkTypes.Coins `json:"result"`
}

type totalSupplyResult struct {
	Result TotalSupplyResponse `json:"result"`
}

type TotalSupplyResponse struct {
	Supply     sdkTypes.Coins `json:"supply"`
	Pagination PageResponse   `json:"pagination"`
}

type PageResponse struct {
	NextKey []byte `json:"next_key,omitempty"`
	Total   string `json:"total,omitempty"`
}
