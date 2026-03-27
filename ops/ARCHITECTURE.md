# Architecture

## Components
- **Contracts**: `PriceOracle` emits `PriceUpdated(price, ts)`.
- **Agent**: submits on-chain updates and exposes `/admin/trigger`, `/healthz`, `/metrics`.
- **Monitor**: polls events, stores into Postgres, exposes `/prices`, `/prices/latest`, `/healthz`, `/metrics`.
- **Infra**: Anvil (local), Postgres, Prometheus, Grafana, Kubernetes + Terraform.

## Data Flow
1. Agent submits `updatePrice` to the oracle.
2. Contract emits `PriceUpdated`.
3. Monitor ingests event logs and persists to Postgres.
4. APIs and metrics expose operational state.

## Deployment
- Local: `docker compose up --build` from repo root.
- Cloud: Terraform provisions EKS + ECR; Helm deploys services.
