// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// OracleView is the interface that consumer contracts use to query the price oracle.
interface OracleView {
    // price returns the current oracle price
    function price() external view returns (uint256);
    // lastUpdated returns the block timestamp of the last price update
    function lastUpdated() external view returns (uint256);
}

// OracleConsumer demonstrates how other contracts can safely consume prices from PriceOracle.
// It validates that prices are fresh and available before using them.
contract OracleConsumer {
    error PriceUnavailable();
    error PriceTooOld(uint256 updatedAt, uint256 checkedAt);

    OracleView public oracle; // Reference to the PriceOracle contract
    uint256 public maxDelay; // Maximum acceptable price age in seconds

    constructor(address oracleAddress, uint256 maxDelaySeconds) {
        oracle = OracleView(oracleAddress);
        maxDelay = maxDelaySeconds;
    }

    // getLatestPrice fetches and validates the current price from the oracle.
    // Reverts if price is unavailable or too stale (older than maxDelay).
    function getLatestPrice() external view returns (uint256 p, uint256 ts) {
        p = oracle.price();
        ts = oracle.lastUpdated();

        if (p == 0 || ts == 0) revert PriceUnavailable();

        uint256 nowTs = block.timestamp;
        if (nowTs - ts > maxDelay) revert PriceTooOld(ts, nowTs);
    }
}
