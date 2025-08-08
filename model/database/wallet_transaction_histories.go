package database

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type WalletTransactionHistory struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement"`
	WalletAddress   string    `gorm:"column:address;size:191;uniqueIndex:idx_wallet_tx_to;index:idx_address;index:idx_address_chain,priority:1"`
	TxHash          string    `gorm:"column:tx_hash;size:66;uniqueIndex:idx_wallet_tx_to;index:idx_tx_hash"`
	ToAddress       string    `gorm:"column:to_address;size:191;uniqueIndex:idx_wallet_tx_to;index:idx_to_address"`
	FromAddress     string    `gorm:"column:from_address;size:191;index:idx_from_address"`
	Chain           string    `gorm:"column:chain;size:20;primaryKey;index:idx_chain;index:idx_address_chain,priority:2;index:idx_chain_tx_hash,priority:1"`
	Value           string    `gorm:"column:value"`
	BlockNumber     int64     `gorm:"column:block_number"`
	BlockTime       int64     `gorm:"column:block_time;index:idx_block_time"`
	Standard        string    `gorm:"column:standard;index:idx_standard"`
	ContractAddress string    `gorm:"column:contract_address;index:idx_contract_address"`
	TokenName       string    `gorm:"column:token_name"`
	TokenSymbol     string    `gorm:"column:token_symbol"`
	TokenDecimal    uint8     `gorm:"column:token_decimal"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (t *WalletTransactionHistory) TableName() string {
	return "wallet_transaction_histories"
}

func BatchCreateWalletTransactionHistories(ctx context.Context, tx *gorm.DB, histories []*WalletTransactionHistory) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	if len(histories) == 0 {
		return nil
	}

	result := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}, {Name: "tx_hash"}},
		DoNothing: true,
	}).Create(histories)

	if result.Error != nil {
		return fmt.Errorf("failed to batch create wallet transaction histories: %w", result.Error)
	}

	return nil
}

func CheckTransactionExists(ctx context.Context, db *gorm.DB, walletAddress, txHash string) (bool, error) {
	var count int64
	result := db.WithContext(ctx).
		Model(&WalletTransactionHistory{}).
		Where("LOWER(address) = LOWER(?) AND tx_hash = ?", walletAddress, txHash).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check if transaction exists: %w", result.Error)
	}
	return count > 0, nil
}
