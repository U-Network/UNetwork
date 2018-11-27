# How to deploy smart contract
In view of the booming Ethereum community and the popularity of EVM in the developer community, we chose to be fully compatible with EVM. As a result, web3.js in EVM is fully applicable to this project.

 
## Toos required
- geth
## Install tools
- Install go environment
```shell
$ brew install go
```
- install solidity compiler
```shell
npm install -g solc
```
## Smart contract deployment steps
1. Write a contract
Here we provide a little simple smart contract code
```JavaScript
// Save as `Test4UUU.sol`
pragma solidity ^0.5.0;
 
contract Test4UUU {
	uint256 private num ;
	address private owner;
	
	constructor () public{
    	owner = msg.sender;
	}
	
	function set(uint256 _num) public{
    	require(msg.sender == owner);
    	num = _num;
	}
	
	function get() public view returns(uint256) {
    	return num;
	}
}
```
 
2. Obtain the `abi` and `bytecode` of contract
```shell
$ solcjs --abi Test4UUU.sol
$ solcjs --bin Test4UUU.sol
$ ls
test4UUU.sol	test4UUU_sol_Test4UUU.abi   test4UUU_sol_Test4UUU.bin
```
 
3. Use geth to connect to UUU node(We assume that a single node is built locally, and the port number is 12321)
```shell
$ geth attach http://localhost:12321
```
 
4. Record down the `abi` and `bytecode` from step 2
```JavaScript
> abi = [{"constant":false,"inputs":[{"name":"_num","type":"uint256"}],"name":"set","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"get","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]
[{
	constant: false,
	inputs: [{
    	name: "_num",
    	type: "uint256"
	}],
	name: "set",
	outputs: [],
	payable: false,
	stateMutability: "nonpayable",
	type: "function"
}, {
	constant: true,
	inputs: [],
	name: "get",
	outputs: [{
    	name: "",
    	type: "uint256"
	}],
	payable: false,
	stateMutability: "view",
	type: "function"
}, {
	inputs: [],
	payable: false,
	stateMutability: "nonpayable",
	type: "constructor"
}]
> 
> // Rember to add 0x in front of bytecode
> bytecode = "0x608060405234801561001057600080fd5b5033600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610152806100616000396000f3fe60806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b1146100515780636d4ce63c1461008c575b600080fd5b34801561005d57600080fd5b5061008a6004803603602081101561007457600080fd5b81019080803590602001909291905050506100b7565b005b34801561009857600080fd5b506100a161011d565b6040518082815260200191505060405180910390f35b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561011357600080fd5b8060008190555050565b6000805490509056fea165627a7a723058203cd5f52b7c7e4816235494ea8350cbbd40c3d1509de4c0e7d43ebb03a7a24e460029"
"0x608060405234801561001057600080fd5b5033600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610152806100616000396000f3fe60806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b1146100515780636d4ce63c1461008c575b600080fd5b34801561005d57600080fd5b5061008a6004803603602081101561007457600080fd5b81019080803590602001909291905050506100b7565b005b34801561009857600080fd5b506100a161011d565b6040518082815260200191505060405180910390f35b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561011357600080fd5b8060008190555050565b6000805490509056fea165627a7a723058203cd5f52b7c7e4816235494ea8350cbbd40c3d1509de4c0e7d43ebb03a7a24e460029"
>    	
```
 
5. Create an account and unlock
```javaScript
> personal.newAccount("123")    // 123为密码
"0xce1a4382f815e224a4261efd4480b5045343d7f2"
> personal.unlockAccount(eth.accounts[0])
Unlock account 0xce1a4382f815e224a4261efd4480b5045343d7f2
Passphrase:
true
```
 
6. Instantiate the created contract
```javaScript
> var contract = eth.contract(abi)
undefined
> var initializer = {from:eth.accounts[0], data:bytecode, gas:300000, gasPrice:0}
undefined
> var theTest = contract.new(initializer)
undefined
>
> // Instantiate your own smart contract
> mycontract = contract.at(theTest.address)
{
  abi: [{
  	constant: false,
  	inputs: [{...}],
  	name: "set",
  	outputs: [],
  	payable: false,
  	stateMutability: "nonpayable",
  	type: "function"
  }, {
  	constant: true,
  	inputs: [],
  	name: "get",
  	outputs: [{...}],
  	payable: false,
  	stateMutability: "view",
  	type: "function"
  }, {
  	inputs: [],
  	payable: false,
  	stateMutability: "nonpayable",
  	type: "constructor"
  }],
  address: "0xe0970a9b8c5ef7c543523d52925328bff930e219",
  transactionHash: null,
  allEvents: function(),
  get: function(),
  set: function()
}
```
 
7. Call smart contract
```JavaScript
> personal.unlockAccount(eth.accounts[0],"123")
true
> mycontract.set.sendTransaction(321,{from:eth.accounts[0],gasPrice:0})
"0x2c1f7f7ebf5b10c1e3731a7fda419caa47038c0ec32cb70dabc41f76ed6c9050"
> mycontract.get.call()
321
```
 
At this point, the call and deployment of smart contract is finished.
