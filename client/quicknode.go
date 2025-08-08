package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type QuickNodeClient struct {
	httpClient   *http.Client
	tronEndpoint string
	ethEndpoint  string
	apiKey       string
	apiKeyHeader string
}

func NewQuickNodeClient(tronEndpoint, ethEndpoint, apiKey, apiKeyHeader string) *QuickNodeClient {
	return &QuickNodeClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tronEndpoint: tronEndpoint,
		ethEndpoint:  ethEndpoint,
		apiKey:       apiKey,
		apiKeyHeader: apiKeyHeader,
	}
}

type AddWatchedAddressesRequest struct {
	AddItems []string `json:"addItems"`
}

type GetWatchedAddressesResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items []string `json:"items"`
	} `json:"data"`
}

func (c *QuickNodeClient) AddWatchedAddresses(ctx context.Context, addresses []string, chain string) error {
	if len(addresses) == 0 {
		return nil
	}
	reqBody := AddWatchedAddressesRequest{
		AddItems: addresses,
	}
	var endpoint string
	if chain == "TRON" {
		endpoint = c.tronEndpoint
	} else {
		endpoint = c.ethEndpoint
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		endpoint,
		bytes.NewBuffer(reqBodyBytes),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set(c.apiKeyHeader, c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add watched addresses, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetWatchedAddresses gets all addresses from the watched-addresses list from quicknode K-V store
func (c *QuickNodeClient) GetWatchedAddresses(ctx context.Context, chain string) ([]string, error) {
	var endpoint string
	if chain == "TRON" {
		endpoint = c.tronEndpoint
	} else {
		endpoint = c.ethEndpoint
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(c.apiKeyHeader, c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get watched addresses, status code: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var response GetWatchedAddressesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return response.Data.Items, nil
}
