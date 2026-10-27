# Task 1: Sepolia Blockchain Read and Write

This Go example connects to the Sepolia test network, reads a block, and sends a basic ETH transfer.

## Requirements

- Go 1.24 or newer
- An Infura API key for Sepolia
- Sepolia test ETH for the account that sends a transaction

## Configuration

Copy `.env.example` to `.env` in this directory and fill in your Infura API key or a complete Sepolia RPC URL:

```sh
cp .env.example .env
```

For transfers, set `SEPOLIA_PRIVATE_KEY` in `.env` to the private key of a Sepolia-only test account. The key is never printed. Values already set in the shell take precedence over `.env`.

Never commit `.env` or use a mainnet account. The sender is derived from the private key; the transaction is rejected unless the RPC endpoint reports Sepolia's chain ID (`11155111`).

## Query a block

```sh
go run . block 6000000
```

Prints the block number, hash, timestamp, and transaction count.

## Send a test transfer

```sh
go run . send 0xRecipientAddress 0.001
```

The final argument is the amount in ETH. The command signs and broadcasts the transaction, waits up to five minutes for its receipt, and returns successfully only when the receipt status is successful. It prints the transaction hash and confirmed block number. A mined failed transaction or a confirmation timeout is returned as an error with the transaction hash. Use only test ETH; sent transactions cannot be undone.

## Run tests

```sh
go test ./...
```