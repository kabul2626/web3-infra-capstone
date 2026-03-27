// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/PriceOracle.sol";
import "../src/OracleConsumer.sol";

contract OracleConsumerTest is Test {
    PriceOracle oracle;
    OracleConsumer consumer;

    function setUp() public {
        oracle = new PriceOracle();
        consumer = new OracleConsumer(address(oracle), 60);
    }

    function test_readsFreshPrice() public {
        vm.warp(100);
        oracle.updatePrice(2000);

        (uint256 p, uint256 ts) = consumer.getLatestPrice();
        assertEq(p, 2000);
        assertEq(ts, 100);
    }

    function test_revertsIfNoPriceExists() public {
        vm.expectRevert(OracleConsumer.PriceUnavailable.selector);
        consumer.getLatestPrice();
    }

    function test_revertsIfPriceIsStale() public {
        vm.warp(100);
        oracle.updatePrice(2000);

        vm.warp(1000);
        vm.expectRevert(abi.encodeWithSelector(OracleConsumer.PriceTooOld.selector, 100, 1000));
        consumer.getLatestPrice();
    }
}
