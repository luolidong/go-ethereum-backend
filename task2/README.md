# Task 2: Generate Contract Bindings

This example compiles a Solidity counter, generates Go bindings with `abigen`, and uses them to deploy and interact with the contract on Sepolia.

## Requirements

- Go 1.25 or newer
- Solidity compiler (`solc`) 0.8.20 or newer
- Sepolia RPC access and test ETH for deployment and writes

## Compile and generate bindings

From this directory:

```sh
solc --abi --bin --overwrite -o contracts/build contracts/Counter.sol
go install github.com/ethereum/go-ethereum/cmd/abigen@v1.17.10
abigen --abi contracts/build/Counter.abi --bin contracts/build/Counter.bin --pkg bindings --type Counter --out bindings/counter.go
```

The generated ABI and bytecode in `contracts/build` are kept with the project so the binding can be regenerated.

## Configure

```sh
cp .env.example .env
```

Fill in `INFURA_API_KEY` or `SEPOLIA_RPC_URL`. Set `SEPOLIA_PRIVATE_KEY` to a Sepolia-only test account. Never commit `.env` or use a mainnet key.

## Deploy and interact

```sh
go run . deploy
```

After deployment, set the printed address as `COUNTER_CONTRACT_ADDRESS` in `.env`, then run:

```sh
go run . read
go run . increment
```

`increment` waits for the transaction receipt and then prints the updated counter value. Transactions cannot be undone, so use Sepolia test ETH only.

## Verify

```sh
go test ./...
```