package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

const sepoliaChainID = 11155111

var weiPerEther = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}

	rpcURL, err := sepoliaRPCURL()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("connect to Sepolia: %w", err)
	}
	defer client.Close()

	switch args[0] {
	case "block":
		if len(args) != 2 {
			return usageError()
		}
		return printBlock(ctx, client, args[1])
	case "send":
		if len(args) != 3 {
			return usageError()
		}
		return sendEther(ctx, client, args[1], args[2])
	default:
		return usageError()
	}
}

func usageError() error {
	return fmt.Errorf("usage: go run . block <block-number> | go run . send <recipient-address> <amount-in-ETH>")
}

func sepoliaRPCURL() (string, error) {
	if rpcURL := os.Getenv("SEPOLIA_RPC_URL"); rpcURL != "" {
		return rpcURL, nil
	}
	apiKey := os.Getenv("INFURA_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("set INFURA_API_KEY or SEPOLIA_RPC_URL")
	}
	return "https://sepolia.infura.io/v3/" + apiKey, nil
}

func printBlock(ctx context.Context, client *ethclient.Client, blockNumberText string) error {
	blockNumber, err := strconv.ParseUint(blockNumberText, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid block number %q: %w", blockNumberText, err)
	}

	block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return fmt.Errorf("query block %d: %w", blockNumber, err)
	}

	fmt.Printf("Block number:    %d\n", block.NumberU64())
	fmt.Printf("Block hash:      %s\n", block.Hash().Hex())
	fmt.Printf("Timestamp:       %d\n", block.Time())
	fmt.Printf("Transactions:    %d\n", len(block.Transactions()))
	return nil
}

func sendEther(ctx context.Context, client *ethclient.Client, recipientText, amountText string) error {
	if !common.IsHexAddress(recipientText) {
		return fmt.Errorf("invalid recipient address %q", recipientText)
	}
	recipient := common.HexToAddress(recipientText)
	if recipient == (common.Address{}) {
		return fmt.Errorf("recipient address must not be the zero address")
	}

	amountWei, err := parseEtherAmount(amountText)
	if err != nil {
		return err
	}

	privateKeyText := strings.TrimPrefix(os.Getenv("SEPOLIA_PRIVATE_KEY"), "0x")
	if privateKeyText == "" {
		return fmt.Errorf("set SEPOLIA_PRIVATE_KEY to a Sepolia test account private key")
	}
	privateKey, err := crypto.HexToECDSA(privateKeyText)
	if err != nil {
		return fmt.Errorf("invalid SEPOLIA_PRIVATE_KEY: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("read network chain ID: %w", err)
	}
	if chainID.Cmp(big.NewInt(sepoliaChainID)) != 0 {
		return fmt.Errorf("refusing to send: connected chain ID is %s, expected Sepolia (%d)", chainID, sepoliaChainID)
	}

	sender := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, sender)
	if err != nil {
		return fmt.Errorf("read sender nonce: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("suggest gas price: %w", err)
	}
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:     sender,
		To:       &recipient,
		Value:    amountWei,
		GasPrice: gasPrice,
	})
	if err != nil {
		return fmt.Errorf("estimate gas limit: %w", err)
	}
	gasLimit += gasLimit / 5

	transaction := types.NewTransaction(nonce, recipient, amountWei, gasLimit, gasPrice, nil)
	signedTransaction, err := types.SignTx(transaction, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return fmt.Errorf("sign transaction: %w", err)
	}
	if err := client.SendTransaction(ctx, signedTransaction); err != nil {
		return fmt.Errorf("send transaction: %w", err)
	}

	fmt.Printf("From:           %s\n", sender.Hex())
	fmt.Printf("To:             %s\n", recipient.Hex())
	fmt.Printf("Amount:         %s ETH\n", amountText)
	fmt.Printf("Transaction:    %s\n", signedTransaction.Hash().Hex())
	return nil
}

func parseEtherAmount(amountText string) (*big.Int, error) {
	dotCount := 0
	digitCount := 0
	for _, character := range amountText {
		switch {
		case character >= '0' && character <= '9':
			digitCount++
		case character == '.':
			dotCount++
		default:
			return nil, fmt.Errorf("amount must be a positive decimal ETH value")
		}
	}
	if digitCount == 0 || dotCount > 1 {
		return nil, fmt.Errorf("amount must be a positive decimal ETH value")
	}

	amount, ok := new(big.Rat).SetString(amountText)
	if !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be a positive decimal ETH value")
	}

	amountWei := new(big.Rat).Mul(amount, new(big.Rat).SetInt(weiPerEther))
	if amountWei.Denom().Cmp(big.NewInt(1)) != 0 {
		return nil, fmt.Errorf("amount %q has more than 18 decimal places", amountText)
	}
	return new(big.Int).Set(amountWei.Num()), nil
}