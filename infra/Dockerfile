# Anvil Dockerfile
# Runs Foundry's Anvil local Ethereum development node for testing
FROM ghcr.io/foundry-rs/foundry:latest

# Expose JSON-RPC endpoint
EXPOSE 8545

# Start Anvil listening on all interfaces
CMD ["anvil", "--host", "0.0.0.0"]