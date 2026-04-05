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

## Architecture Overview

This project is built as a full-stack oracle system with three main layers:

1. **Smart contracts** (`contracts/`)
   - `PriceOracle.sol`: on-chain oracle storage and permissioned price updates.
   - `OracleConsumer.sol`: example consumer contract that reads and validates prices.
   - Foundry tests ensure contract behavior and CI quality.

2. **Agent and monitor services** (`services/`)
   - `services/agent`: fetches price feeds from external providers, validates values, and writes updates to the on-chain oracle.
   - `services/monitor`: listens for contract events, indexes oracle activity, and exposes internal metrics.
   - Both services are built in Go, with OpenAPI definitions and production-ready Dockerfiles.

3. **Infrastructure** (`infra/`)
   - `infra/terraform/aws/`: Terraform code for AWS infrastructure provisioning, including the Kubernetes cluster and supporting cloud resources.
   - `infra/k8s/`: Kubernetes manifests and Helm charts used to deploy the local and cloud environment.
   - `infra/prometheus/` and `infra/grafana/`: monitoring configuration for observability.

## How it works

- In local development, `docker compose up --build` brings up Anvil, the agent, the monitor, Postgres, Prometheus, and Grafana.
- The agent service polls external price providers, validates the feed, and calls `PriceOracle.updatePrice()` on the deployed contract.
- The monitor service watches blockchain events and stores indexable data in Postgres, while also exposing Prometheus metrics.
- The `PriceOracle` contract stores the latest price and timestamp; consumers can safely query `OracleConsumer` to ensure prices are fresh and available.

## Terraform and Kubernetes

- The Terraform configuration in `infra/terraform/aws/` is responsible for provisioning AWS resources required for a production deployment.
- Kubernetes manifests in `infra/k8s/manifests/` define the runtime components, including:
  - `agent` and `monitor` services
  - `anvil` (local Ethereum node for development)
  - `postgres` database
  - `prometheus` and `grafana` for observability
- A Helm chart is available under `infra/k8s/helm/web3-infra-capstone/` for configurable deployments.
- The GitHub workflows deploy services and contracts using the same infrastructure configuration when targeting cloud environments.

## Security Notes

- Never commit `.env` file with real secrets
- `.env.example` shows required variables (use as template)
- Private keys in docker-compose.yml reference env vars for local development only
- Production values come from GitHub Secrets (see `.github/workflows/`)