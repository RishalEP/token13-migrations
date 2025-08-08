package handler

import (
	"context"
	"github.com/gin-gonic/gin"
	"log"
	"quicknode/service"
)

type Handler struct {
	transactionService *service.TransactionService
}

func NewHandler(transactionService *service.TransactionService) *Handler {
	return &Handler{
		transactionService: transactionService,
	}
}

// SyncAddresses triggers the QuickNode workflow to sync addresses
func (h *Handler) SyncAddresses(c *gin.Context) {
	err, ethAddressesCount, tronAddressesCount := h.transactionService.SyncAddressesToQuickNode(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"status":        "success",
		"message":       "Addresses synced successfully",
		"address_count": ethAddressesCount + tronAddressesCount,
	})
}

// FetchTransactions triggers the QuickNode workflow to fetch transactions
func (h *Handler) FetchTransactions(c *gin.Context) {
	bgCtx := context.Background()

	// Return immediately with a 201 status code
	c.JSON(201, gin.H{
		"status":  "success",
		"message": "Transaction fetch started in background",
	})

	// Process in background
	go func() {
		err, stats := h.transactionService.FetchAndStoreTransactions(bgCtx)
		if err != nil {
			log.Printf("Background fetch error: %v", err)
			return
		}

		// Log detailed statistics
		log.Printf("Background fetch completed successfully:")
		log.Printf("- Processed %d ETH addresses and %d TRON addresses", 
			stats.EthAddressesProcessed, stats.TronAddressesProcessed)
		log.Printf("- Added %d ETH transactions and %d TRON transactions", 
			stats.EthTransactionsAdded, stats.TronTransactionsAdded)

		if len(stats.FailedEthAddresses) > 0 {
			log.Printf("- Failed to process %d ETH addresses: %v", 
				len(stats.FailedEthAddresses), stats.FailedEthAddresses)
		}

		if len(stats.FailedTronAddresses) > 0 {
			log.Printf("- Failed to process %d TRON addresses: %v", 
				len(stats.FailedTronAddresses), stats.FailedTronAddresses)
		}
	}()
}

// SyncAddressToQuicknode triggers the QuickNode workflow to add a specific address
func (h *Handler) SyncAddressToQuicknode(c *gin.Context) {
	var req struct {
		Address string `json:"address"`
		Chain   string `json:"chain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"status":  "error",
			"message": "Invalid request: " + err.Error(),
		})
		return
	}
	err := h.transactionService.SyncAddressToQuickNode(c.Request.Context(), req.Address, req.Chain)
	if err != nil {
		c.JSON(500, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "Address synced to QuickNode successfully",
	})
}
