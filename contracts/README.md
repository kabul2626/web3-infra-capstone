# Price Oracle Smart Contracts

Production-grade smart contracts for a decentralized price oracle system using Foundry.

## Overview

### PriceOracle.sol
Central oracle contract that stores and updates the latest ETH/USD price.
- **Owner**: Only the oracle agent can call `updatePrice()`
- **Price**: Stored as scaled integer (e.g., 35000 = $350.00)
- **Events**: Emits `PriceUpdated` on every price change for off-chain indexing

### OracleConsumer.sol
Reference implementation showing how other contracts safely consume prices.
- **Price Validation**: Ensures price is available and not stale
- **Freshness Window**: Configurable maximum age (e.g., 60 seconds)
- **Error Handling**: Custom errors for better UX and gas efficiency

## Development with Foundry

**Foundry** is a blazing fast, portable and modular toolkit for Ethereum application development written in Rust.

### Quick Start

#### Build contracts
```shell
forge build
```

#### Run tests (unit, fuzz, invariants)
```shell
forge test -vvv
```

#### Format code per style guide
```shell
forge fmt
```

#### Check gas usage
```shell
forge snapshot
```

#### Start local Anvil node
```shell
anvil
```

#### Deploy contracts
```shell
forge script script/Deploy.s.sol --rpc-url <RPC_URL> --private-key <PRIVATE_KEY> --broadcast
```

## Testing Strategy

- **Unit Tests**: Basic functionality (price updates, validation)
- **Fuzz Tests**: Random inputs to find edge cases
- **Invariant Tests**: Ensure protocol properties hold under any condition

Run with gas reporting:
```shell
forge test -vvv --fuzz-runs 256 --gas-report
```

## Contract Deployment

### Local (Anvil)
```shell
# Start Anvil (default chain ID: 31337)
anvil

# In another terminal, deploy
forge script script/Deploy.s.sol --rpc-url http://localhost:8545 --private-key <ANVIL_KEY> --broadcast
```

### Testnet/Mainnet
Set up environment variables in `.env`:
```bash
RPC_URL=https://your-rpc-endpoint
DEPLOYER_PK=your_private_key
```

Then deploy:
```shell
forge script script/Deploy.s.sol --broadcast
```

## Integration with Oracle Agent

The Go agent service (`services/agent`) automatically:
1. Fetches external prices (CoinCap, Coinbase API)
2. Validates against price thresholds
3. Calls `updatePrice()` on deployed PriceOracle contract
4. Retries with exponential backoff on failure

Monitor prices via:
- **Direct query**: `cast call <ORACLE_ADDRESS> "price()" --rpc-url <RPC_URL>`
- **Events**: Monitor `PriceUpdated` events for real-time updates
- **Go monitor service**: Indexes events and stores in PostgreSQL

## Documentation

- [Foundry Book](https://book.getfoundry.sh/)
- [Solidity Documentation](https://docs.soliditylang.org/)
