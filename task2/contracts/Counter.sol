// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract Counter {
	uint256 public value;

	event Incremented(address indexed account, uint256 value);

	function increment() external {
		value += 1;
		emit Incremented(msg.sender, value);
	}
}