## 如何部署智能合约
鉴于现在以太坊社区的蓬勃发展，以及EVM的在开发者社区中的普及程度，我们选择完全兼容EVM，因此EVM中的web3.js等完全适用于本项目

### 需要的工具
- geth
### 安装工具
- 安装go环境
```shell
$ brew install go
```
- 安装solidity编译器
```shell
npm install -g solc 
```
### 部署合约步骤
1. 编写合约
此处我们提供一点简单的智能合约代码
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

2. 获取合约的 `abi` 和 `bytecode` 
```shell
$ solcjs --abi Test4UUU.sol
$ solcjs --bin Test4UUU.sol
$ ls
test4UUU.sol    test4UUU_sol_Test4UUU.abi   test4UUU_sol_Test4UUU.bin
```

3. 用geth连接到UUU节点(我们假设本地搭建了一个单节点，端口号为12321)
```shell
$ geth attach http://localhost:12321
```

4. 将刚才的 `abi` 和 `bytecode` 记录下来
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
> // 记得在字节码前加0x
> bytecode = "0x608060405234801561001057600080fd5b5033600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610152806100616000396000f3fe60806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b1146100515780636d4ce63c1461008c575b600080fd5b34801561005d57600080fd5b5061008a6004803603602081101561007457600080fd5b81019080803590602001909291905050506100b7565b005b34801561009857600080fd5b506100a161011d565b6040518082815260200191505060405180910390f35b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561011357600080fd5b8060008190555050565b6000805490509056fea165627a7a723058203cd5f52b7c7e4816235494ea8350cbbd40c3d1509de4c0e7d43ebb03a7a24e460029"
"0x608060405234801561001057600080fd5b5033600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610152806100616000396000f3fe60806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b1146100515780636d4ce63c1461008c575b600080fd5b34801561005d57600080fd5b5061008a6004803603602081101561007457600080fd5b81019080803590602001909291905050506100b7565b005b34801561009857600080fd5b506100a161011d565b6040518082815260200191505060405180910390f35b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561011357600080fd5b8060008190555050565b6000805490509056fea165627a7a723058203cd5f52b7c7e4816235494ea8350cbbd40c3d1509de4c0e7d43ebb03a7a24e460029"
>        
```

5. 创建账户并解锁
```javaScript
> personal.newAccount("123")    // 123为密码
"0xce1a4382f815e224a4261efd4480b5045343d7f2"
> personal.unlockAccount(eth.accounts[0])
Unlock account 0xce1a4382f815e224a4261efd4480b5045343d7f2
Passphrase:
true
```

6. 实例化创建合约
```javaScript
> var contract = eth.contract(abi)
undefined
> var initializer = {from:eth.accounts[0], data:bytecode, gas:300000, gasPrice:0}
undefined
> var theTest = contract.new(initializer) 
undefined
> 
> // 实例化自己的智能合约
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

7. 调用智能合约
```JavaScript
> personal.unlockAccount(eth.accounts[0],"123")
true
> mycontract.set.sendTransaction(321,{from:eth.accounts[0],gasPrice:0})
"0x2c1f7f7ebf5b10c1e3731a7fda419caa47038c0ec32cb70dabc41f76ed6c9050"
> mycontract.get.call()
321
```

到这里，合约的调用和部署就结束了。