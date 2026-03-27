test-contracts:
	cd web3-infra-capstone/contracts && forge test -vvv --fuzz-runs 256 --gas-report

deploy-local:
	docker compose up --build

run-compose:
	docker compose up --build

test-go:
	cd web3-infra-capstone/services/agent && go test ./... -race
	cd web3-infra-capstone/services/monitor && go test ./... -race

lint-go:
	cd web3-infra-capstone/services/agent && golangci-lint run ./...
	cd web3-infra-capstone/services/monitor && golangci-lint run ./...
