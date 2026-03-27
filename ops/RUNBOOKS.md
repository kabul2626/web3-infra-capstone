# Runbooks

## Agent Update Failures
1. Check agent logs for RPC errors.
2. Verify `RPC_URL` and `ORACLE_ADDRESS` envs.
3. Confirm chain is reachable and funded.

## Monitor No Updates
1. Check monitor logs for RPC or DB issues.
2. Verify Postgres connectivity and table.
3. Ensure agent is submitting transactions.

## Grafana/Prometheus Down
1. Check container/pod health.
2. Verify ports (Grafana 3000, Prometheus 9090).
