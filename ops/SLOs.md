# SLOs

## SLIs
- API availability: % 2xx over total requests.
- Ingest freshness: time since last `PriceUpdated` event.
- Submission success: % successful on-chain submissions.
- Latency: p95 request duration for read APIs.

## SLO Targets
- API availability: 99.5% (30d rolling).
- Ingest freshness: p95 < 90s.
- Submission success: > 99% (excluding upstream RPC outages).

## Error Budget
- 0.5% monthly budget for API errors.
