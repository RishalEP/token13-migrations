package global

import (
	"encoding/json"
	"errors"
	"fmt"
	gresty "github.com/go-resty/resty/v2"
	"math/big"
	"time"
)

var errTronGridHTTPError = errors.New("tron grid http error")

type ContractParameter struct {
	TypeURL string `json:"type_url"`
	Value   struct {
		Amount       *big.Int `json:"amount"`
		OwnerAddress string   `json:"owner_address"`
		ToAddress    string   `json:"to_address"`
	} `json:"value"`
}

type Contract struct {
	Parameter ContractParameter `json:"parameter"`
	Type      string            `json:"type"`
}

type RawData struct {
	Contract      []Contract `json:"contract"`
	Expiration    int64      `json:"expiration"`
	RefBlockBytes string     `json:"ref_block_bytes"`
	RefBlockHash  string     `json:"ref_block_hash"`
	Timestamp     int64      `json:"timestamp"`
}

type Ret struct {
	ContractRet string `json:"contractRet"`
	Fee         int64  `json:"fee"`
}

type Transactions struct {
	BlockNumber          int64         `json:"blockNumber"`
	BlockTimestamp       int64         `json:"block_timestamp"`
	EnergyFee            int64         `json:"energy_fee"`
	EnergyUsage          int64         `json:"energy_usage"`
	EnergyUsageTotal     int64         `json:"energy_usage_total"`
	InternalTransactions []interface{} `json:"internal_transactions"`
	NetFee               int64         `json:"net_fee"`
	NetUsage             int64         `json:"net_usage"`
	RawData              RawData       `json:"raw_data"`
	RawDataHex           string        `json:"raw_data_hex"`
	Ret                  []Ret         `json:"ret"`
	Signature            []string      `json:"signature"`
	TxID                 string        `json:"txID"`
}

type Meta1 struct {
	At       int64 `json:"at"`
	PageSize int   `json:"page_size"`
}

type Response struct {
	Data    []Transactions `json:"data"`
	Meta    *Meta1         `json:"meta"`
	Success bool           `json:"success"`
}

type TronGridClient struct {
	client *gresty.Client
}
type ListTransactionsResponse struct {
	Meta    *Meta          `json:"meta"`
	Data    []*Transaction `json:"data"`
	Success bool           `json:"success"`
}
type Meta struct {
	Links       *MetaLinks `json:"links"`
	Fingerprint string     `json:"fingerprint"`
	At          int64      `json:"at"`
	PageSize    int32      `json:"page_size"`
}

type MetaLinks struct {
	Next string `json:"next"`
}
type Token struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int32  `json:"decimals"`
}

type Transaction struct {
	Token          *Token          `json:"token_info"`
	From           string          `json:"from"`
	ID             string          `json:"transaction_id"`
	To             string          `json:"to"`
	Type           TransactionType `json:"type"`
	Value          string          `json:"value"`
	BlockTimestamp int64           `json:"block_timestamp"`
}

type TransactionType string

type ListTrxTransactionsResponse struct {
	//Data    []*TransactionRaw `json:"data"`
	Success bool  `json:"success"`
	Meta    *Meta `json:"meta"`
}

func NewTronGridClient(url, apiKey string) (*TronGridClient, error) {
	client := gresty.New()
	client.SetHostURL(url)
	client.SetHeader("TRON-PRO-API-KEY", apiKey)
	client.OnAfterResponse(func(c *gresty.Client, r *gresty.Response) error {
		statusCode := r.StatusCode()
		if statusCode >= 400 {
			method := r.Request.Method
			url := r.Request.URL
			return fmt.Errorf("%d cannot %s %s: %w", statusCode, method, url, errTronGridHTTPError)
		}
		return nil
	})
	return &TronGridClient{
		client: client,
	}, nil
}

type VoteWitnessContractParameter struct {
	Value struct {
		OwnerAddress string `json:"owner_address"`
		Votes        []struct {
			VoteAddress string `json:"vote_address"`
			VoteCount   int64  `json:"vote_count"`
		} `json:"votes"`
	} `json:"value"`
	TypeUrl string `json:"type_url"`
}

type TransferContractParameter struct {
	Value struct {
		Amount       int64  `json:"amount"`        // 转账金额
		OwnerAddress string `json:"owner_address"` // 发送地址
		ToAddress    string `json:"to_address"`    // 收款地址
	} `json:"value"`
	TypeUrl string `json:"type_url"`
}

type FreezeBalanceV2ContractParameter struct {
	Value struct {
		FrozenBalance int64  `json:"frozen_balance"` // 冻结余额
		OwnerAddress  string `json:"owner_address"`  // 冻结余额的地址
	} `json:"value"`
	TypeUrl string `json:"type_url"`
}

type UnfreezeBalanceV2ContractParameter struct {
	Value struct {
		UnfreezeBalance int64  `json:"unfreeze_balance"` // 冻结余额
		OwnerAddress    string `json:"owner_address"`    // 冻结余额的地址
	} `json:"value"`
	TypeUrl string `json:"type_url"`
}
type TransferValue struct {
	Amount       int64  `json:"amount"`
	OwnerAddress string `json:"owner_address"` // hex-encoded
	ToAddress    string `json:"to_address"`    // hex-encoded
}
type InternalTransaction struct {
	Hash          string `json:"hash"`
	CallerAddress string `json:"caller_address"`
	TransferTo    string `json:"transferTo"`
	From          string `json:"from"`
	To            string `json:"to"`
	CallValue     string `json:"callValue"`
	Note          string `json:"note"`
}
type TransferAssetContractParameter struct {
	Value struct {
		OwnerAddress string `json:"owner_address"`
		ToAddress    string `json:"to_address"`
		AssetName    string `json:"asset_name"` // TRC10 token ID
		Amount       int64  `json:"amount"`
	} `json:"value"`
}

type tronTxData struct {
	Data []struct {
		Ret []struct {
			ContractRet string `json:"contractRet"`
			Fee         int64  `json:"fee"`
		} `json:"ret"`
		Signature        []string `json:"signature"`
		TxId             string   `json:"txID"`
		NetUsage         int64    `json:"net_usage"`
		RawDataHex       string   `json:"raw_data_hex"`
		NetFee           int64    `json:"net_fee"` // 交易费用
		EnergyUsage      int64    `json:"energy_usage"`
		BlockNumber      int64    `json:"blockNumber"`
		BlockTimestamp   int64    `json:"block_timestamp"`
		EnergyFee        int64    `json:"energy_fee"`
		EnergyUsageTotal int64    `json:"energy_usage_total"`
		RawData          struct {
			Data     string `json:"data"`
			Contract []struct {
				Parameter json.RawMessage `json:"parameter"`
				Type      string          `json:"type"`
			} `json:"contract"`
			RefBlockBytes string `json:"ref_block_bytes"`
			RefBlockHash  string `json:"ref_block_hash"`
			Expiration    int64  `json:"expiration"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"raw_data"`
		InternalTransactions json.RawMessage `json:"internal_transactions"`
	} `json:"data"`
	Success bool `json:"success"`
	Meta    struct {
		At       int64 `json:"at"`
		PageSize int   `json:"page_size"`
	} `json:"meta"`
}

func (tgc *TronGridClient) GetAddressTrxTransactions(address string, confirmed bool) (*tronTxData, error) {
	time.Sleep(1500 * time.Millisecond)
	list := &Response{}
	response, err := tgc.client.R().
		ForceContentType("application/json").
		SetResult(list).
		SetPathParam("address", address).
		SetQueryParam("limit", "200").
		SetQueryParam("search_internal", "true").
		SetQueryParam("only_confirmed", fmt.Sprintf("%t", confirmed)).
		Get("v1/accounts/{address}/transactions")
	if err != nil {
		return nil, err
	}
	if response.StatusCode() != 200 {
		return nil, errTronGridHTTPError
	}
	var data tronTxData
	err = json.Unmarshal(response.Body(), &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (tgc *TronGridClient) GetAddressTrc20Transactions(address, contract string, confirmed bool) (*ListTransactionsResponse, error) {
	list := &ListTransactionsResponse{}
	response, err := tgc.client.R().
		ForceContentType("application/json").
		SetResult(&list).
		SetPathParam("address", address).
		SetQueryParam("contract_address", contract).
		SetQueryParam("limit", "200").
		SetQueryParam("only_confirmed", fmt.Sprintf("%t", confirmed)).
		Get("v1/accounts/{address}/transactions/trc20")
	if err != nil {
		return nil, err
	}
	if response.StatusCode() != 200 {
		return nil, errTronGridHTTPError
	}
	return list, nil
}
