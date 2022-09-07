package distribution

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type client struct {
	Url string
}

func NewRestClient(url string) *client {
	return &client{Url: url}
}

func (c client) GetParams(ctx context.Context) (ParametersResponse, error) {
	respStr, err := c.get(ctx, "/distribution/parameters")
	if err != nil {
		return ParametersResponse{}, err
	}

	var res parametersResult
	if err := json.Unmarshal([]byte(respStr), &res); err != nil {
		return ParametersResponse{}, err
	}

	return res.Result, nil
}

func (c client) get(ctx context.Context, uri string) (string, error) {
	getReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s%s", c.Url, uri), nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type parametersResult struct {
	Result ParametersResponse `json:"result"`
}

type ParametersResponse struct {
	CommunityTax string `json:"community_tax"`
}
