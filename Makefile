# Foundry test suite for Solidity contracts with fuzzing and gas reporting
test-contracts:
	cd web3-infra-capstone/contracts && forge test -vvv --fuzz-runs 256 --gas-report

# Deploy full local stack with Docker Compose
deploy-local:
	docker compose up --build

# Alias for deploy-local
run-compose:
	docker compose up --build

# Run Go unit tests with race condition detection
test-go:
	cd web3-infra-capstone/services/agent && go test ./... -race
	cd web3-infra-capstone/services/monitor && go test ./... -race

# Run Go linter on all services
lint-go:
	cd web3-infra-capstone/services/agent && golangci-lint run ./...
	cd web3-infra-capstone/services/monitor && golangci-lint run ./...
