package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nanmu42/etherscan-api"
	"gorm.io/gorm/clause"
	"log"
	"math/big"
	"quicknode/client"
	"quicknode/global"
	global2 "quicknode/global"
	qndb "quicknode/model/database"
	"quicknode/utils"
	"strings"
	"sync"
	"time"
)

// TransactionStats holds statistics about transaction processing
type TransactionStats struct {
	// Addresses processed
	TronAddressesProcessed int
	EthAddressesProcessed  int

	// Transactions added to DB
	TronTransactionsAdded int
	EthTransactionsAdded  int

	// Failed addresses
	FailedTronAddresses []string
	FailedEthAddresses  []string
}

type TransactionService struct {
	quickNodeClient *client.QuickNodeClient
	tronGridClient  *global.TronGridClient
	etherscanCli    *etherscan.Client
}

func NewTransactionService(tronGridURL, tronGridAPIKey string, etherscanUrl string, etherscanApiKey string,
	quickNodeTronEndpoint, quickNodeEthEndpoint, quickNodeAPIKey, quickNodeAPIKeyHeader string) (*TransactionService, error) {
	tronGridClient, err := global.NewTronGridClient(tronGridURL, tronGridAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create TronGrid client: %w", err)
	}
	etherScanClient := etherscan.NewCustomized(etherscan.Customization{
		Timeout: 30 * time.Second,
		Key:     etherscanApiKey,
		BaseURL: etherscanUrl,
	})

	return &TransactionService{
		quickNodeClient: client.NewQuickNodeClient(quickNodeTronEndpoint, quickNodeEthEndpoint, quickNodeAPIKey, quickNodeAPIKeyHeader),
		tronGridClient:  tronGridClient,
		etherscanCli:    etherScanClient,
	}, nil
}
func GetBindAddress(ctx context.Context, chainId, action int) ([]string, error) {
	address := make([]string, 0)
	db := global2.MustGetDB(global2.DBNameToken13).WithContext(ctx)
	var err error
	query := db.Table("account_bind_info").Where("chain_id = ? ", chainId)
	switch action {
	case 0:
		err = query.Select("address").Find(&address).Error
	default:
		err = query.Where("action = ?", action).Select("address").Find(&address).Error
	}
	if err != nil {
		return nil, fmt.Errorf("db GetBindAddress(%v), err:%s", err.Error())
	}
	return address, nil
}

func (s *TransactionService) SyncAddressesToQuickNode(ctx context.Context) (error, int, int) {
	// Fetch ETH addresses (ChainId = 0)
	ethAddresses, err := GetBindAddress(ctx, int(global.ConstChainIdETH), 0)
	if err != nil {
		return fmt.Errorf("failed to get ETH addresses: %w", err), 0, 0
	}

	// Fetch TRON addresses (ChainId = 2)
	tronAddresses, err := GetBindAddress(ctx, int(global.ConstChainIdTRON), 0)
	if err != nil {
		return fmt.Errorf("failed to get TRON addresses: %w", err), 0, 0
	}

	// Convert all TRON addresses to hex format
	var TronHexAddresses []string

	if err := s.quickNodeClient.AddWatchedAddresses(ctx, utils.UniqueStrings(ethAddresses), "ETH"); err != nil {
		return fmt.Errorf("failed to add addresses to QuickNode: %w", err), 0, 0
	}
	for _, addr := range tronAddresses {
		addrHex, _ := utils.ConvertTronToHexAddress(addr)
		TronHexAddresses = append(TronHexAddresses, addrHex)
	}
	if err := s.quickNodeClient.AddWatchedAddresses(ctx, utils.UniqueStrings(TronHexAddresses), "TRON"); err != nil {
		return fmt.Errorf("failed to add addresses to QuickNode: %w", err), 0, 0
	}
	log.Printf("Successfully synced %d ethereum addresses & %d Tron Addresses to QuickNode", len(ethAddresses), len(tronAddresses))
	return nil, len(ethAddresses), len(tronAddresses)
}

func (s *TransactionService) SyncAddressToQuickNode(ctx context.Context, address string, chain string) error {
	var hexAddresses []string
	addr, _ := utils.ConvertTronToHexAddress(address)
	hexAddresses = append(hexAddresses, addr)
	if err := s.quickNodeClient.AddWatchedAddresses(ctx, hexAddresses, chain); err != nil {
		return fmt.Errorf("failed to add addresses to QuickNode: %w", err)
	}

	txCount, err := s.processAddress(ctx, addr, chain)
	if err != nil {
		log.Printf("Error processing address %s: %v", addr, err)
	} else {
		log.Printf("Successfully processed address %s, added %d transactions", addr, txCount)
	}

	log.Printf("Successfully synced %s", address)
	return nil
}

func (s *TransactionService) FetchAndStoreTransactions(ctx context.Context) (error, *TransactionStats) {
	stats := &TransactionStats{}

	TronAddresses, err := s.quickNodeClient.GetWatchedAddresses(ctx, "TRON")
	if err != nil {
		return fmt.Errorf("failed to get addresses from QuickNode: %w", err), stats
	}
	EthAddresses, err := s.quickNodeClient.GetWatchedAddresses(ctx, "ETH")
	if err != nil {
		return fmt.Errorf("failed to get addresses from QuickNode: %w", err), stats
	}

	stats.TronAddressesProcessed = len(TronAddresses)
	stats.EthAddressesProcessed = len(EthAddresses)

	log.Printf("Fetching transactions for %d Tron addresses & %d Eth Addresses", len(TronAddresses), len(EthAddresses))
	batchSize := 10
	var wg sync.WaitGroup
	var statsMutex sync.Mutex

	tronSemaphore := make(chan struct{}, 3)
	ethSemaphore := make(chan struct{}, 2)

	// Process TRON addresses
	for i := 0; i < len(TronAddresses); i += batchSize {
		end := i + batchSize
		if end > len(TronAddresses) {
			end = len(TronAddresses)
		}

		batch := TronAddresses[i:end]
		wg.Add(1)
		tronSemaphore <- struct{}{}

		go func(addrs []string) {
			defer wg.Done()
			defer func() { <-tronSemaphore }()

			for _, addr := range addrs {
				txCount, err := s.processAddress(ctx, addr, "TRON")
				if err != nil {
					log.Printf("Error processing TRON address %s: %v", addr, err)
					statsMutex.Lock()
					stats.FailedTronAddresses = append(stats.FailedTronAddresses, addr)
					statsMutex.Unlock()
				} else {
					statsMutex.Lock()
					stats.TronTransactionsAdded += txCount
					statsMutex.Unlock()
				}
			}
		}(batch)
	}

	// Process ETH addresses
	for i := 0; i < len(EthAddresses); i += batchSize {
		end := i + batchSize
		if end > len(EthAddresses) {
			end = len(EthAddresses)
		}

		batch := EthAddresses[i:end]
		wg.Add(1)
		ethSemaphore <- struct{}{}

		go func(addrs []string) {
			defer wg.Done()
			defer func() { <-ethSemaphore }()

			for _, addr := range addrs {
				txCount, err := s.processAddress(ctx, addr, "ETH")
				if err != nil {
					log.Printf("Error processing ETH address %s: %v", addr, err)
					statsMutex.Lock()
					stats.FailedEthAddresses = append(stats.FailedEthAddresses, addr)
					statsMutex.Unlock()
				} else {
					statsMutex.Lock()
					stats.EthTransactionsAdded += txCount
					statsMutex.Unlock()
				}
			}
		}(batch)
	}
	wg.Wait()

	// Retry failed addresses
	if len(stats.FailedTronAddresses) > 0 || len(stats.FailedEthAddresses) > 0 {
		log.Printf("Retrying %d failed TRON addresses and %d failed ETH addresses",
			len(stats.FailedTronAddresses), len(stats.FailedEthAddresses))

		failedTronAddrs := make([]string, len(stats.FailedTronAddresses))
		copy(failedTronAddrs, stats.FailedTronAddresses)
		failedEthAddrs := make([]string, len(stats.FailedEthAddresses))
		copy(failedEthAddrs, stats.FailedEthAddresses)

		stats.FailedTronAddresses = []string{}
		stats.FailedEthAddresses = []string{}

		// Retry TRON addresses
		for _, addr := range failedTronAddrs {
			txCount, err := s.processAddress(ctx, addr, "TRON")
			if err != nil {
				log.Printf("Error retrying TRON address %s: %v", addr, err)
				stats.FailedTronAddresses = append(stats.FailedTronAddresses, addr)
			} else {
				stats.TronTransactionsAdded += txCount
			}
		}

		// Retry ETH addresses
		for _, addr := range failedEthAddrs {
			txCount, err := s.processAddress(ctx, addr, "ETH")
			if err != nil {
				log.Printf("Error retrying ETH address %s: %v", addr, err)
				stats.FailedEthAddresses = append(stats.FailedEthAddresses, addr)
			} else {
				stats.EthTransactionsAdded += txCount
			}
		}
	}
	return nil, stats
}

func (s *TransactionService) processAddress(ctx context.Context, address string, chain string) (int, error) {
	txCount := 0

	if chain == "TRON" {
		tronAddress, err := utils.HexToTronAddress(address)
		if err != nil {
			log.Printf("Failed to convert hex to TRON address for %s: %v", address, err)
			return 0, err
		}
		count, err := s.fetchAndStoreTronTransactions(ctx, tronAddress)
		if err != nil {
			log.Printf("Failed to fetch TRON transactions for %s: %v", tronAddress, err)
			return 0, err
		}
		txCount = count
	} else {
		count, err := s.fetchAndStoreEthTransactions(ctx, address)
		if err != nil {
			log.Printf("Failed to fetch Eth transactions for %s: %v", address, err)
			return 0, err
		}
		txCount = count
	}
	return txCount, nil
}

func (s *TransactionService) fetchAndStoreTronTransactions(ctx context.Context, address string) (int, error) {
	txCount := 0
	txData, err := s.tronGridClient.GetAddressTrxTransactions(address, true)
	if err != nil {
		return 0, fmt.Errorf("Tron: failed to get TRON transactions: %w", err)
	}

	// Fetch TRC-20 token transfers
	tokenTxs, err := s.fetchTRC20Transfers(ctx, address)
	if err != nil {
		log.Printf("Tron: Error fetching TRC-20 token transfers: %v", err)
	}

	if (txData == nil || len(txData.Data) == 0) && (tokenTxs == nil || len(tokenTxs.Data) == 0) {
		log.Printf("Tron: No transactions or token transfers found for address %s", address)
		return 0, nil
	}

	walletAddressHex, err := utils.ConvertTronToHexAddress(address)
	if err != nil {
		log.Printf("Tron: Error converting wallet address to hex: %v", err)
		walletAddressHex = address
	}
	if txData != nil && len(txData.Data) > 0 {
		db := global.MustGetDB(global.DBNameToken13)
		tx := db.Begin()
		if tx.Error != nil {
			return 0, fmt.Errorf("failed to begin transaction: %w", tx.Error)
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		var histories []*qndb.WalletTransactionHistory
		for _, txn := range txData.Data {
			blockTime := txn.BlockTimestamp
			blockNumber := txn.BlockNumber
			txHash := txn.TxId
			exists, err := qndb.CheckTransactionExists(ctx, db, walletAddressHex, txHash)
			if err != nil {
				log.Printf("Tron: Error checking if transaction exists: %v", err)
				continue
			}
			if exists {
				log.Printf("Tron: Transaction %s already exists for address %s, skipping", txHash, walletAddressHex)
				continue
			}
			if len(txn.RawData.Contract) > 0 {
				contract := txn.RawData.Contract[0]
				switch contract.Type {
				case "TransferContract":
					var param global2.TransferContractParameter
					if err := json.Unmarshal(contract.Parameter, &param); err == nil {
						fromHex := param.Value.OwnerAddress
						toHex := param.Value.ToAddress

						histories = append(histories, &qndb.WalletTransactionHistory{
							WalletAddress: strings.ToLower(walletAddressHex),
							TxHash:        txHash,
							Chain:         "tron",
							BlockNumber:   blockNumber,
							BlockTime:     blockTime,
							FromAddress:   strings.ToLower(fromHex),
							ToAddress:     strings.ToLower(toHex),
							Value:         fmt.Sprintf("%d", param.Value.Amount),
							Standard:      "native",
							CreatedAt:     time.Now(),
						})
					}
				case "TransferAssetContract":
					var param global.TransferAssetContractParameter
					if err := json.Unmarshal(contract.Parameter, &param); err == nil {
						fromHex := param.Value.OwnerAddress
						toHex := param.Value.ToAddress
						histories = append(histories, &qndb.WalletTransactionHistory{
							WalletAddress: strings.ToLower(walletAddressHex),
							TxHash:        txHash,
							Chain:         "tron",
							BlockNumber:   blockNumber,
							BlockTime:     blockTime,
							FromAddress:   strings.ToLower(fromHex),
							ToAddress:     strings.ToLower(toHex),
							Value:         fmt.Sprintf("%d", param.Value.Amount),
							Standard:      "TRC10",
							TokenName:     param.Value.AssetName,
							CreatedAt:     time.Now(),
						})
					}
				case "TriggerSmartContract":
					//trc-20 contracts handled seperately
					continue
				}
			}
		}

		// Store transactions
		tronTxCount := len(histories)
		if tronTxCount > 0 {
			// Use OnConflict to ignore duplicate records (based on the unique index on address and tx_hash)
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "address"}, {Name: "tx_hash"}},
				DoNothing: true,
			}).Create(&histories).Error; err != nil {
				tx.Rollback()
				return 0, fmt.Errorf("Tron: failed to insert transactions: %w", err)
			}
			log.Printf("Tron: Stored %d TRON transactions for address %s", tronTxCount, walletAddressHex)
			txCount += tronTxCount
		}

		if err := tx.Commit().Error; err != nil {
			return 0, fmt.Errorf("Tron: failed to commit transaction: %w", err)
		}
	}

	// Process TRC-20 token transfers
	if tokenTxs != nil && len(tokenTxs.Data) > 0 {
		tokenTxCount, err := s.storeTRC20Transfers(ctx, address, tokenTxs)
		if err != nil {
			log.Printf("Error storing TRC-20 token transfers: %v", err)
			// Continue even if storing token transfers fails
		} else {
			txCount += tokenTxCount
		}
	}
	return txCount, nil
}

// fetchTRC20Transfers fetches TRC-20 token transfers for a given address
func (s *TransactionService) fetchTRC20Transfers(ctx context.Context, address string) (*global.ListTransactionsResponse, error) {
	// Fetch TRC-20 token transfers
	time.Sleep(1500 * time.Millisecond)
	tokenTxs, err := s.tronGridClient.GetAddressTrc20Transactions(address, "", true)
	if err != nil {
		return nil, fmt.Errorf("failed to get TRC-20 token transfers: %w", err)
	}

	if tokenTxs == nil || len(tokenTxs.Data) == 0 {
		log.Printf("No TRC-20 token transfers found for %s", address)
		return nil, nil
	}

	return tokenTxs, nil
}

func (s *TransactionService) storeTRC20Transfers(ctx context.Context, address string, tokenTxs *global.ListTransactionsResponse) (int, error) {
	if tokenTxs == nil || len(tokenTxs.Data) == 0 {
		return 0, nil
	}

	walletAddressHex, err := utils.ConvertTronToHexAddress(address)
	if err != nil {
		log.Printf("Error converting wallet address to hex: %v", err)
		walletAddressHex = address // Fallback to original address if conversion fails
	}
	db := global.MustGetDB(global.DBNameToken13)
	tx := db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var histories []*qndb.WalletTransactionHistory
	for _, tokenTx := range tokenTxs.Data {
		exists, err := qndb.CheckTransactionExists(ctx, db, walletAddressHex, tokenTx.ID)
		if err != nil {
			log.Printf("Tron: Error checking if token transfer exists: %v", err)
			continue
		}
		if exists {
			log.Printf("Tron: Token transfer %s already exists for address %s, skipping", tokenTx.ID, walletAddressHex)
			continue
		}
		fromHex, _ := utils.ConvertTronToHexAddress(tokenTx.From)
		toHex, _ := utils.ConvertTronToHexAddress(tokenTx.To)

		history := &qndb.WalletTransactionHistory{
			WalletAddress:   strings.ToLower(walletAddressHex),
			TxHash:          tokenTx.ID,
			Chain:           "tron",
			BlockNumber:     0, // Not available in the response
			BlockTime:       tokenTx.BlockTimestamp,
			FromAddress:     strings.ToLower(fromHex),
			ToAddress:       strings.ToLower(toHex),
			Value:           tokenTx.Value,
			Standard:        "TRC20",
			ContractAddress: strings.ToLower(tokenTx.Token.Address),
			TokenName:       tokenTx.Token.Name,
			TokenSymbol:     tokenTx.Token.Symbol,
			TokenDecimal:    uint8(tokenTx.Token.Decimals),
			CreatedAt:       time.Now(),
		}

		histories = append(histories, history)
	}

	txCount := len(histories)
	if txCount > 0 {
		// Use OnConflict to ignore duplicate records (based on the unique index on address and tx_hash)
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "address"}, {Name: "tx_hash"}},
			DoNothing: true,
		}).Create(&histories).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to store token transfers: %w", err)
		}
		log.Printf("Stored %d TRC-20 token transfers for address %s", txCount, walletAddressHex)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return txCount, nil
}

// fetchEthTransactions fetches Ethereum transactions for a given address
func (s *TransactionService) fetchEthTransactions(ctx context.Context, address string) ([]etherscan.NormalTx, error) {
	time.Sleep(1500 * time.Millisecond)
	txs, err := s.etherscanCli.NormalTxByAddress(address, nil, nil, 1, 100, true)
	if err != nil {
		if strings.Contains(err.Error(), "No transactions") {
			log.Printf("No transactions found for %s, Continue..", address)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get ETH transactions: %w", err)
	}

	if txs == nil || len(txs) == 0 {
		return nil, nil
	}
	return txs, nil
}

// fetchERC20Transfers fetches ERC20 token transfers for a given address
func (s *TransactionService) fetchERC20Transfers(ctx context.Context, address string) ([]etherscan.ERC20Transfer, error) {
	time.Sleep(1500 * time.Millisecond)
	addrPtr := &address
	tokenTxs, err := s.etherscanCli.ERC20Transfers(nil, addrPtr, nil, nil, 1, 100, true)
	if err != nil {
		if strings.Contains(err.Error(), "No transactions") {
			log.Printf("No token transfers found for %s, Continue..", address)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get ERC20 token transfers: %w", err)
	}

	if tokenTxs == nil || len(tokenTxs) == 0 {
		return nil, nil
	}

	return tokenTxs, nil
}

// storeEthTransactions stores Ethereum transactions in the database and returns the number of transactions stored
func (s *TransactionService) storeEthTransactions(ctx context.Context, address string, txs []etherscan.NormalTx) (int, error) {
	if txs == nil || len(txs) == 0 {
		return 0, nil
	}

	db := global.MustGetDB(global.DBNameToken13)
	tx := db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("ETH: failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var histories []*qndb.WalletTransactionHistory
	for _, txn := range txs {
		// Check if transaction already exists
		exists, err := qndb.CheckTransactionExists(ctx, db, address, txn.Hash)
		if err != nil {
			log.Printf("ETH: Error checking if transaction exists: %v", err)
			continue
		}
		if exists {
			log.Printf("ETH: Transaction %s already exists for address %s, skipping", txn.Hash, address)
			continue
		}

		// Skip transactions with value = 0 as they are likely token transfers
		if (*big.Int)(txn.Value).Cmp(big.NewInt(0)) == 0 {
			log.Printf("ETH: Transaction %s has value 0, skipping as it's likely a token transfer", txn.Hash)
			continue
		}

		history := &qndb.WalletTransactionHistory{
			WalletAddress: strings.ToLower(address),
			TxHash:        txn.Hash,
			Chain:         "ETH",
			BlockNumber:   int64(txn.BlockNumber),
			BlockTime:     time.Time(txn.TimeStamp).Unix(),
			FromAddress:   strings.ToLower(txn.From),
			ToAddress:     strings.ToLower(txn.To),
			Value:         utils.ConvertWeiToEth((*big.Int)(txn.Value)),
			CreatedAt:     time.Now(),
		}

		histories = append(histories, history)
	}

	txCount := len(histories)
	if txCount > 0 {
		if err := qndb.BatchCreateWalletTransactionHistories(ctx, tx, histories); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("ETH: failed to store transactions: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Stored %d ETH transactions for address %s", txCount, address)
	return txCount, nil
}

// storeERC20Transfers stores ERC20 token transfers in the database and returns the number of transfers stored
func (s *TransactionService) storeERC20Transfers(ctx context.Context, address string, tokenTxs []etherscan.ERC20Transfer) (int, error) {
	if tokenTxs == nil || len(tokenTxs) == 0 {
		return 0, nil
	}
	db := global.MustGetDB(global.DBNameToken13)
	tx := db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("ETH: failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var histories []*qndb.WalletTransactionHistory
	for _, tokenTx := range tokenTxs {
		// Check if transaction already exists
		exists, err := qndb.CheckTransactionExists(ctx, db, address, tokenTx.Hash)
		if err != nil {
			log.Printf("ETH: Error checking if token transfer exists: %v", err)
			continue
		}
		if exists {
			log.Printf("ETH: Token transfer %s already exists for address %s, skipping", tokenTx.Hash, address)
			continue
		}
		tokenValue := "0"
		if tokenTx.Value != nil {
			divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tokenTx.TokenDecimal)), nil))
			tokenValueFloat := new(big.Float).Quo(new(big.Float).SetInt((*big.Int)(tokenTx.Value)), divisor)
			tokenValue = tokenValueFloat.Text('f', int(tokenTx.TokenDecimal))
			tokenValue = strings.TrimRight(tokenValue, "0")
			if strings.HasSuffix(tokenValue, ".") {
				tokenValue = tokenValue[:len(tokenValue)-1]
			}
		}

		history := &qndb.WalletTransactionHistory{
			WalletAddress:   strings.ToLower(address),
			TxHash:          tokenTx.Hash,
			Chain:           "ETH",
			BlockNumber:     int64(tokenTx.BlockNumber),
			BlockTime:       time.Time(tokenTx.TimeStamp).Unix(),
			FromAddress:     strings.ToLower(tokenTx.From),
			ToAddress:       strings.ToLower(tokenTx.To),
			Value:           tokenValue,
			Standard:        "ERC20",
			ContractAddress: strings.ToLower(tokenTx.ContractAddress),
			TokenName:       tokenTx.TokenName,
			TokenSymbol:     tokenTx.TokenSymbol,
			TokenDecimal:    tokenTx.TokenDecimal,
			CreatedAt:       time.Now(),
		}

		histories = append(histories, history)
	}

	txCount := len(histories)
	if txCount > 0 {
		if err := qndb.BatchCreateWalletTransactionHistories(ctx, tx, histories); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("ETH: failed to store token transfers: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("ETH: failed to commit transaction: %w", err)
	}
	log.Printf("ETH: Stored %d ERC20 token transfers for address %s", txCount, address)
	return txCount, nil
}

// fetchAndStoreEthTransactions fetches and stores Ethereum transactions for a given address
func (s *TransactionService) fetchAndStoreEthTransactions(ctx context.Context, address string) (int, error) {
	txCount := 0
	txs, err := s.fetchEthTransactions(ctx, address)
	if err != nil {
		return txCount, err
	}
	tokenTxs, err := s.fetchERC20Transfers(ctx, address)
	if err != nil {
		return txCount, err
	}
	if txs != nil && len(txs) > 0 {
		ethTxCount, err := s.storeEthTransactions(ctx, address, txs)
		if err != nil {
			return txCount, fmt.Errorf("failed to store ETH transactions: %w", err)
		}
		txCount += ethTxCount
	}
	if tokenTxs != nil && len(tokenTxs) > 0 {
		tokenTxCount, err := s.storeERC20Transfers(ctx, address, tokenTxs)
		if err != nil {
			return txCount, fmt.Errorf("failed to store ERC20 token transfers: %w", err)
		}
		txCount += tokenTxCount
	}
	if txCount == 0 {
		log.Printf("ETH: No transactions or token transfers found for address %s", address)
	} else {
		log.Printf("ETH: Added a total of %d transactions for address %s", txCount, address)
	}
	return txCount, nil
}
