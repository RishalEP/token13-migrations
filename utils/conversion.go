// Package utils provides utility functions for the quicknode-migrator application.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcutil/base58"
	"math/big"
	"strings"
)

// ConvertTronToHexAddress converts a TRON address to hex format if it's not already
func ConvertTronToHexAddress(address string) (string, error) {
	//already eth style hex format return
	if len(address) > 2 && address[:2] == "0x" {
		return address, nil
	}
	// Decode from base58
	addrBytes := base58.Decode(address)
	if len(addrBytes) != 25 {
		return "", fmt.Errorf("invalid address length: %d", len(addrBytes))
	}

	hexAddr := addrBytes[1 : 1+20] // 20 bytes = 40 hex chars

	// Convert to hex string with 0x prefix
	return "0x" + strings.ToUpper(hex.EncodeToString(hexAddr)), nil
}

// HexToTronAddress converts a hex address to TRON format
func HexToTronAddress(hexAddr string) (string, error) {
	hexAddr = strings.TrimPrefix(hexAddr, "0x")
	addrBytes, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", err
	}
	if len(addrBytes) != 20 {
		return "", fmt.Errorf("invalid address length: got %d bytes", len(addrBytes))
	}
	prefixed := append([]byte{0x41}, addrBytes...)
	checksum := sha256d(prefixed)[:4]
	final := append(prefixed, checksum...)
	return base58.Encode(final), nil
}

// sha256d applies sha256 twice
func sha256d(data []byte) []byte {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return second[:]
}

// ConvertWeiToEth converts Wei to Ether
func ConvertWeiToEth(wei *big.Int) string {
	weiFloat := new(big.Float).SetInt(wei)
	ethValue := new(big.Float).Quo(weiFloat, big.NewFloat(1e18))
	str := fmt.Sprintf("%.18f", ethValue)
	str = strings.TrimRight(str, "0")
	if strings.HasSuffix(str, ".") {
		str = str[:len(str)-1]
	}
	return str
}

// UniqueStrings returns a new slice with duplicate strings removed
func UniqueStrings(input []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, s := range input {
		key := strings.ToLower(s)
		if !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	return result
}
