// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/PriceOracle.sol";

contract PriceOracleTest is Test {
    PriceOracle oracle;

    address owner = address(this);
    address attacker = address(0xBEEF);

    function setUp() public {
        oracle = new PriceOracle();
    }

    function test_ownerCanUpdate() public {
        vm.warp(1000);
        oracle.updatePrice(123);

        assertEq(oracle.price(), 123);
        assertEq(oracle.lastUpdated(), 1000);
    }

    function test_nonOwnerCannotUpdate() public {
        vm.prank(attacker);
        vm.expectRevert(PriceOracle.NotOwner.selector);
        oracle.updatePrice(123);
    }

    function test_rejectZeroPrice() public {
        vm.expectRevert(PriceOracle.InvalidPrice.selector);
        oracle.updatePrice(0);
    }
}
