// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract PriceOracle {
    error NotOwner();
    error InvalidPrice();

    address public owner;
    uint256 public price;
    uint256 public lastUpdated;

    event PriceUpdated(uint256 price, uint256 ts);

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    constructor() {
        owner = msg.sender;
    }

    function updatePrice(uint256 newPrice) external onlyOwner {
        if (newPrice == 0) revert InvalidPrice();
        price = newPrice;
        lastUpdated = block.timestamp;
        emit PriceUpdated(newPrice, lastUpdated);
    }
}
