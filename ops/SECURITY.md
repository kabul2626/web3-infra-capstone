# Security

- Never commit private keys or secrets.
- Use least-privilege IAM roles for CI/CD.
- Rotate keys regularly; prefer AWS Secrets Manager or GitHub Environments.
- Separate dev/testnet/prod environments.
- Use TLS for external endpoints and restrict admin endpoints.
