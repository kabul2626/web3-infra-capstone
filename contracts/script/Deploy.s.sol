// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// Foundry scripting utilities
import "forge-std/Script.sol";

// Contracts to deploy
import "../src/PriceOracle.sol";
import "../src/OracleConsumer.sol";

contract Deploy is Script {
    function run() external {
        // Load deployer private key from environment
        uint256 deployerKey = vm.envUint("PRIVATE_KEY");

        // Start broadcasting transactions to the network
        vm.startBroadcast(deployerKey);

        // Deploy the price oracle
        PriceOracle oracle = new PriceOracle();

        // Deploy the consumer with a 60-second freshness window
        OracleConsumer consumer =
            new OracleConsumer(address(oracle), 60);

        // Stop broadcasting transactions
        vm.stopBroadcast();

        // Log deployed addresses for CI output
        console2.log("PriceOracle deployed at:", address(oracle));
        console2.log("OracleConsumer deployed at:", address(consumer));
    }
}
