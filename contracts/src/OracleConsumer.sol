// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface OracleView {
    function price() external view returns (uint256);
    function lastUpdated() external view returns (uint256);
}

contract OracleConsumer {
    error PriceUnavailable();
    error PriceTooOld(uint256 updatedAt, uint256 checkedAt);

    OracleView public oracle;
    uint256 public maxDelay;

    constructor(address oracleAddress, uint256 maxDelaySeconds) {
        oracle = OracleView(oracleAddress);
        maxDelay = maxDelaySeconds;
    }

    function getLatestPrice() external view returns (uint256 p, uint256 ts) {
        p = oracle.price();
        ts = oracle.lastUpdated();

        if (p == 0 || ts == 0) revert PriceUnavailable();

        uint256 nowTs = block.timestamp;
        if (nowTs - ts > maxDelay) revert PriceTooOld(ts, nowTs);
    }
}
