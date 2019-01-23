[![Build Status](https://travis-ci.org/UNetworkProject/UNetwork.svg?branch=master)](https://travis-ci.org/UNetworkProject/UNetwork)

# UNetwork 

UNetwork是go语言实现的基于区块链技术的去中心化的分布式网络协议。可以用来数字化资产和金融相关业务包括资产注册，发行，转账等。

## 特性

* 可扩展的通用智能合约
* 高度优化的交易处理速度
* 基于IPFS的分布式存储和文件共享解决方案
* 节点访问权限控制
* P2P连接链路加密
* 可配置区块生成时间
* 可配置电子货币模型
* 可配置的分区共识
* 接近零成本的交易费用

# 编译
成功编译UNetwork需要以下准备：

* Go版本在1.11及以上
* 正确的Go语言开发环境

克隆UNetwork仓库到$GOPATH/src目录

```shell
$ git clone https://github.com/U-Network/UNetworkDev.git
```

用make编译源码

```shell
$ make
```

成功编译后会生成两个可以执行程序

* `uuu`: 节点程序

# 部署

成功运行UNetwork有两种方式

* 单节点运行
* 多节点运行

## 单节点运行

我们可以通过命令行的方式部署，先对环境进行初始化

```shell
$ ./uuu node --log_level="debug" init "~/genesis.json"
```

genesis.json配置文件参考
```shell
$ cat ./genesis.json
{
  "config": {
    "chainId": 9384,
    "homesteadBlock": 0,
    "eip155Block": 0,
    "eip158Block": 0
  },
  "alloc": {
    "0xd03b5d1bf0715fffad8d821d546bd9e8aa2c9b10": {
      "balance": "1000000000000000000000000000"
    }
  },
  "nonce": "0x0000000000000042",
  "difficulty": "0x020000",
  "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "timestamp": "0x00",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "extraData": "",
  "gasLimit": "0xffffffff"
}
```

`0xd03b5d1bf0715fffad8d821d546bd9e8aa2c9b10`地址对应的私钥为`52a0b3688fd46ab9ab7d62372719e3037681d5cf62e862336bc9c3c468c4a448`

初始化之后，通过下面的命令来运行节点

```shell
$ ./uuu node start --log_level="debug"  --ethparam="--rpcapi db,eth,net,web3,personal"
```

## 多节点运行

当进行多节点运行UNetwork时，我们需要至少四个节点。通过以下命令，自动生成4个节点的配置信息。

```shell
$ uuu node --log_level="debug" createnode 4 "chain-test-123"
```

这样会生成相关的配置文件，目录如下所示
```shell
$ tree ~/.unetwork/

├── config
│   └── config.toml
├── config1
│   ├── ID.json
│   ├── genesis.json
│   ├── node_key.json
│   └── priv_validator.json
├── config2
│   ├── ID.json
│   ├── genesis.json
│   ├── node_key.json
│   └── priv_validator.json
├── config3
│   ├── ID.json
│   ├── genesis.json
│   ├── node_key.json
│   └── priv_validator.json
├── config4
│   ├── ID.json
│   ├── genesis.json
│   ├── node_key.json
│   └── priv_validator.json
└── data
```

为了后面的配置正常进行，在此我们需要对相关的配置文件作简要解释

以`config1`为例

以下是分配给节点1的ID号
```shell
$ cat ID.json
e6c30179e0ba47bb004934355bfe728f96404b30
```

以下是UNetwork网络中各个挖矿节点的相关信息

```shell
$ cat genesis.json
{
  "genesis_time": "2018-11-06T08:30:42.258885Z",
  "chain_id": "chain-test-123",
  "consensus_params": {
    "block_size_params": {
      "max_bytes": "22020096",
      "max_gas": "-1"
    },
    "evidence_params": {
      "max_age": "100000"
    }
  },
  "validators": [
    {
      "address": "51800FE9E6DD438AAA6467A229FC6B406FFD69B3",
      "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": "pCZ8aOE35ncgrY3ppiacvBkJRH26fKe7cGCMmZfNwVM="
      },
      "power": "10",
      "name": ""
    },
    {
      "address": "F9E1465D713C13D62BA49F1CA55489F513412CA3",
      "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": "VxnkqEhXC3YlRN+V7NsByTQ+A0RcOHLwtXKlA+p9bsk="
      },
      "power": "10",
      "name": ""
    },
    {
      "address": "7C3053ECFA194A90710B1813270919AC599712F1",
      "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": "FfzJWI2Iogoh1w/qndxsM2eZWnLyqSfh9N3wyoZViyA="
      },
      "power": "10",
      "name": ""
    },
    {
      "address": "21BA95111CD501F514144F504FD09FFF48AB5800",
      "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": "hUQsAk4/GUiFgwROVDD0ab5nVCTdxNQHvSVcNPFC4XA="
      },
      "power": "10",
      "name": ""
    }
  ],
  "app_hash": ""
```

以下是挖矿节点的私钥信息

```shell
$ cat node_key.json
{"priv_key":{"type":"tendermint/PrivKeyEd25519","value":"jEXTYjqv2ybwAYXX+ez76JwnI8obOVUfp4yQsNxk9M8unXg5R1Wf+w9uq89Vl8i4wdtMLqRTebmkB6dkJ15aEQ=="}}
```
以下是

```shell
$ cat priv_validator.json 
{
  "address": "51800FE9E6DD438AAA6467A229FC6B406FFD69B3",
  "pub_key": {
    "type": "tendermint/PubKeyEd25519",
    "value": "pCZ8aOE35ncgrY3ppiacvBkJRH26fKe7cGCMmZfNwVM="
  },
  "last_height": "0",
  "last_round": "0",
  "last_step": 0,
  "priv_key": {
    "type": "tendermint/PrivKeyEd25519",
    "value": "0vkuAcQhciPq9dSn3uxq6F/eczsvQHhy1nqp2R927TakJnxo4TfmdyCtjemmJpy8GQlEfbp8p7twYIyZl83BUw=="
  }
}
```

接下来我们将在目标主机上进行下列操作：

1. 将相关文件复制到目标主机，包括：
    - `uuu.exe`
    - `config1`文件夹(以`config1/`为例)

2. 执行以下命令进行初始化：
    `$ uuu node --log_level="debug" init "~/genesis.json"`

3. 将`config1`中的文件覆盖至目标主机的`~/.unetwork/config`下

多机配置完成，每个节点的目录结构如下

```shell
$ tree ~/.unetwork/
├── config
│   ├── config.toml
│   ├── ID.json
│   ├── genesis.json
│   ├── node_key.json
│   └── priv_validator.json
├── uuu.exe
└── data
```


## 运行
以任意顺序在各个主机上以如下格式执行命令

```shell
$ uuu node start
--log_level="debug"
--consensus.create_empty_blocks=true
--consensus.timeout_commit=20000
--p2p.persistent_peers="_id1@_ip1:_port1_,_id2@_ip2:_port2,_id3@_ip3:_port3,_id4@_ip4:_port4"
--ethparam="--rpcport 9148"
```

其中各参数解释如下

--log_level: 打印日志的类型
--consensus.create_empty_blocks: 是否允许出空块
--consensus.timeout_commit: 出块间隔(ms)
--p2p.persistent_peers: 各共识节点的地址信息
    _idx: x节点的ID编号(见ID.json)
    _ipx: x节点的ip地址
    _portx: x节点的端口号
--ethparam: 以太坊的参数信息


# 贡献代码

请您以签过名的commit发送pull request请求，我们期待您的加入！
您也可以通过邮件的方式发送你的代码到开发者邮件列表，欢迎加入UNetwork邮件列表和开发者论坛。

另外，在您想为本项目贡献代码时请提供详细的提交信息，格式参考如下：

	Header line: explain the commit in one line (use the imperative)

	Body of commit message is a few lines of text, explaining things
	in more detail, possibly giving some background about the issue
	being fixed, etc etc.

	The body of the commit message can be several paragraphs, and
	please do proper word-wrap and keep columns shorter than about
	74 characters or so. That way "git log" will show things
	nicely even when it's indented.

	Make sure you explain your solution and why you're doing what you're
	doing, as opposed to describing what you're doing. Reviewers and your
	future self can read the patch, but might not understand why a
	particular solution was implemented.

	Reported-by: whoever-reported-it
	Signed-off-by: Your Name <youremail@yourhost.com>

# 开源社区


## 网站

- http://www.U.network


# 许可证

UNetwork遵守Apache License, 版本2.0。 详细信息请查看项目根目录下的LICENSE文件。


