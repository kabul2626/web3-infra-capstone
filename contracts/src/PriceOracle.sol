// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// PriceOracle is the central contract that stores the latest ETH/USD price.
// The price is updated by the oracle agent via updatePrice() function.
// Consumers query price() to get the latest price and lastUpdated timestamp.
contract PriceOracle {
    error NotOwner();
    error InvalidPrice();

    address public owner; // Oracle agent address that can call updatePrice
    uint256 public price; // Current price scaled by 100 (e.g., 35000 = $350.00)
    uint256 public lastUpdated; // Block timestamp of last price update

    event PriceUpdated(uint256 price, uint256 ts);

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    // constructor sets the deployer as the initial oracle agent
    constructor() {
        owner = msg.sender;
    }

    // updatePrice updates the stored price and emits PriceUpdated event.
    // Only callable by the oracle agent (onlyOwner).
    function updatePrice(uint256 newPrice) external onlyOwner {
        if (newPrice == 0) revert InvalidPrice();
        price = newPrice;
        lastUpdated = block.timestamp;
        emit PriceUpdated(newPrice, lastUpdated);
    }
}
