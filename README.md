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

This project is a complete oracle pipeline from external price feeds into on-chain price data and observability:

1. **Smart contracts** (`contracts/`)
   - `PriceOracle.sol`: stores the latest oracle price and timestamp on-chain.
   - `OracleConsumer.sol`: example consumer contract that reads the oracle and enforces price freshness.
   - `contracts/test/` and `contracts/src/` contain Foundry tests to validate contract behavior before deployment.

2. **Agent service** (`services/agent`)
   - Fetches price data from external providers.
   - Validates the feed against configured thresholds.
   - Submits `updatePrice()` transactions to `PriceOracle`.
   - Runs as a container in local and Kubernetes environments.

3. **Monitor service** (`services/monitor`)
   - Listens for `PriceUpdated` events emitted by `PriceOracle`.
   - Stores event data and derived state in Postgres.
   - Exposes metrics for Prometheus and health endpoints for service monitoring.

4. **Infrastructure** (`infra/`)
   - `infra/terraform/aws/`: defines AWS resources for production deployment.
   - `infra/k8s/`: Kubernetes manifests and Helm charts for deploying the full stack.
   - `infra/prometheus/` and `infra/grafana/`: observability configuration for metrics and dashboards.

## End-to-end flow

1. **Bootstrap environment**
   - Local: run `docker compose up --build` to launch Anvil, agent, monitor, Postgres, Prometheus, and Grafana.
   - Production: provision cloud resources with Terraform and deploy services to Kubernetes.

2. **Contract deployment**
   - Deploy `PriceOracle` and optionally `OracleConsumer` using `forge script`.
   - The deployed oracle contract address is provided to the agent and consumer services.

3. **Data ingestion**
   - The agent polls external price feeds on a schedule.
   - It validates each feed before submitting a transaction to the oracle.
   - Successful updates produce a `PriceUpdated` event on-chain.

4. **On-chain storage**
   - `PriceOracle` saves the latest price and timestamp.
   - Consumers can call `OracleConsumer.getLatestPrice()` to read and verify the price is not stale.

5. **Event monitoring and indexing**
   - The monitor service consumes blockchain events from the oracle contract.
   - It writes structured records to Postgres for querying and auditing.
   - It also exports Prometheus metrics for price update frequency, error rates, and service health.

6. **Observability and alerting**
   - Prometheus scrapes metrics from the agent and monitor.
   - Grafana dashboards visualize price feed status, update cadence, and system health.
   - Alerts can be configured using the Prometheus rules in `infra/prometheus/alert.rules.yml`.

## Terraform and Kubernetes

- Terraform in `infra/terraform/aws/` provisions the AWS infrastructure for production deployments.
- Kubernetes manifests in `infra/k8s/manifests/` describe the runtime services:
  - `agent`
  - `monitor`
  - `postgres`
  - `prometheus`
  - `grafana`
  - `anvil` (local development only)
- A Helm chart under `infra/k8s/helm/web3-infra-capstone/` enables configurable deployment values.
- GitHub workflows use the repository’s infrastructure and deployment definitions for CI/CD.

## Security Notes

- Never commit `.env` file with real secrets
- `.env.example` shows required variables (use as template)
- Private keys in docker-compose.yml reference env vars for local development only
- Production values come from GitHub Secrets (see `.github/workflows/`)