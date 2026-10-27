package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/luolidong/go-ethereum-backend/task2/bindings"
)

const sepoliaChainID = 11155111

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return usageError()
	}
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}

	rpcURL, err := sepoliaRPCURL()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("connect to Sepolia: %w", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("read network chain ID: %w", err)
	}
	if chainID.Cmp(big.NewInt(sepoliaChainID)) != 0 {
		return fmt.Errorf("refusing to continue: connected chain ID is %s, expected Sepolia (%d)", chainID, sepoliaChainID)
	}

	switch args[0] {
	case "deploy":
		return deployCounter(ctx, client, chainID)
	case "read":
		return readCounter(ctx, client)
	case "increment":
		return incrementCounter(ctx, client, chainID)
	default:
		return usageError()
	}
}

func usageError() error {
	return fmt.Errorf("usage: go run . deploy | go run . read | go run . increment")
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

func deployCounter(ctx context.Context, client *ethclient.Client, chainID *big.Int) error {
	auth, err := transactionOptions(ctx, chainID)
	if err != nil {
		return err
	}

	address, transaction, _, err := bindings.DeployCounter(auth, client)
	if err != nil {
		return fmt.Errorf("deploy Counter: %w", err)
	}
	if _, err := waitForSuccess(ctx, client, transaction); err != nil {
		return err
	}

	fmt.Printf("Contract address: %s\n", address.Hex())
	fmt.Printf("Deploy transaction: %s\n", transaction.Hash().Hex())
	fmt.Println("Set COUNTER_CONTRACT_ADDRESS to this address in .env")
	return nil
}

func readCounter(ctx context.Context, client *ethclient.Client) error {
	address, err := contractAddress()
	if err != nil {
		return err
	}
	contract, err := bindings.NewCounterCaller(address, client)
	if err != nil {
		return fmt.Errorf("bind Counter at %s: %w", address.Hex(), err)
	}

	value, err := contract.Value(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("read counter value: %w", err)
	}
	fmt.Printf("Counter value: %s\n", value)
	return nil
}

func incrementCounter(ctx context.Context, client *ethclient.Client, chainID *big.Int) error {
	address, err := contractAddress()
	if err != nil {
		return err
	}
	auth, err := transactionOptions(ctx, chainID)
	if err != nil {
		return err
	}
	contract, err := bindings.NewCounter(address, client)
	if err != nil {
		return fmt.Errorf("bind Counter at %s: %w", address.Hex(), err)
	}

	transaction, err := contract.Increment(auth)
	if err != nil {
		return fmt.Errorf("submit increment transaction: %w", err)
	}
	receipt, err := waitForSuccess(ctx, client, transaction)
	if err != nil {
		return err
	}
	value, err := contract.Value(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("read counter value after increment: %w", err)
	}

	fmt.Printf("Transaction: %s\n", transaction.Hash().Hex())
	fmt.Printf("Mined in block: %d\n", receipt.BlockNumber.Uint64())
	fmt.Printf("Counter value: %s\n", value)
	return nil
}

func transactionOptions(ctx context.Context, chainID *big.Int) (*bind.TransactOpts, error) {
	privateKeyText := strings.TrimPrefix(os.Getenv("SEPOLIA_PRIVATE_KEY"), "0x")
	if privateKeyText == "" {
		return nil, fmt.Errorf("set SEPOLIA_PRIVATE_KEY to a Sepolia test account private key")
	}
	privateKey, err := crypto.HexToECDSA(privateKeyText)
	if err != nil {
		return nil, fmt.Errorf("invalid SEPOLIA_PRIVATE_KEY: %w", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("create transaction signer: %w", err)
	}
	auth.Context = ctx
	return auth, nil
}

func contractAddress() (common.Address, error) {
	addressText := os.Getenv("COUNTER_CONTRACT_ADDRESS")
	if !common.IsHexAddress(addressText) {
		return common.Address{}, fmt.Errorf("set COUNTER_CONTRACT_ADDRESS to the deployed contract address")
	}
	address := common.HexToAddress(addressText)
	if address == (common.Address{}) {
		return common.Address{}, fmt.Errorf("COUNTER_CONTRACT_ADDRESS must not be the zero address")
	}
	return address, nil
}

func waitForSuccess(ctx context.Context, client *ethclient.Client, transaction *types.Transaction) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(ctx, client, transaction)
	if err != nil {
		return nil, fmt.Errorf("wait for transaction %s: %w", transaction.Hash().Hex(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction %s was mined but failed (block %d)", transaction.Hash().Hex(), receipt.BlockNumber.Uint64())
	}
	return receipt, nil
}
