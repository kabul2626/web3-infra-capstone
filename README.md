# web3-infra-capstone

Production-style oracle client + event monitor:
- Solidity (Foundry): PriceOracle + OracleConsumer + tests (unit/fuzz/invariants)
- Go services: agent + monitor
- Infra: Docker Compose, Postgres, Prometheus, Grafana
## Setup

### Local Development

1. **Copy environment template**:
   ```bash
   cp .env.example .env
   ```
   
2. **Update .env with your values** (optional for local dev - defaults work with Anvil):
   - `PRIVATE_KEY`: Oracle agent's signing key
   - `ORACLE_ADDRESS`: Deployed smart contract address
   - `POSTGRES_PASSWORD`: Database password

3. **Start local stack**:
   ```bash
   docker compose up --build
   ```
   
   Services will be available at:
   - Agent: `http://localhost:8081/metrics`
   - Monitor: `http://localhost:8082/metrics`
   - Prometheus: `http://localhost:9090`
   - Grafana: `http://localhost:3000`
   - Anvil RPC: `http://localhost:8545`

### Production Deployment

For AWS/Kubernetes deployment, set these GitHub Secrets:
- `AWS_ACCESS_KEY_ID`: AWS account credentials
- `AWS_SECRET_ACCESS_KEY`: AWS account credentials  
- `RPC_URL`: Blockchain RPC endpoint
- `DEPLOYER_PK`: Private key for contract deployment

## Security Notes

- Never commit `.env` file with real secrets
- `.env.example` shows required variables (use as template)
- Private keys in docker-compose.yml reference env vars for local development only
- Production values come from GitHub Secrets (see `.github/workflows/`)