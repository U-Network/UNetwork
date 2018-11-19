# 区块浏览器及钱包使用文档
## 部署
1. 先从github上拉取代码

```shell
$ git clone git@github.com:U-Network/explorer.git
```

2. 安装npm依赖
```shell
$ cd ./U-Network/explorer
$ npm install
```

3. 修改浏览器配置文件
设置`rpcUrl`与`ipcFile`的参数
```shell
$ vim ./U-Network/explorer/app/config.json
// MacOS参考配置信息如下
{
  "rpcUrl" : "http://127.0.0.1:12321",
  "ipcFile" : "~/Library/.ethereum/unetwork.ipc"
}
```

4. 本地运行浏览器
```shell
$ cd ./U-Network/explorer
$ npm start

> UNetworkWalletExplorer@0.1.0 start /home/wwwroot/explorer
> node server.js

app listening on port 8000!
```

## 钱包使用 
1. 进入区块链浏览器页面 `http://localhost:12321`
 
2. 创建自己的账户
    - 点击`Create New Wallet`
    - 设置密码，并点击`Submit to Create`
    - 点击`Download Keystore File(UTC/JSON)`以保存文件
    - 保存私钥

3. 登录钱包
    - 点击`login my wallet`
    - 两种登录的方式
    1) 点击`选择文件`以选择刚才下载的钱包文件，并输入密码登录
    2) 输入刚才保存的私钥并登录

4. 转账
    - 在 `To Address` 中填入目标地址
    - 在 `Amount to Send` 中填入转入数值并发送

## 一键发币

1. 进入区块链浏览器页面 `http://localhost:12321`

2. 登录钱包(方式见前文)

3. 创建代币
    - 点击 `Generate My Token` 并根据要求设置参数
    - 等待区块确认交易
    - 创建完成，可以在钱包代币列表中看到自己创建的代币

4. 如果收到对方给与的代币，让代币在钱包中显示
    - 通过各种方式获取该代币的合约地址
    - 在 `Get Balance of Token` 中输入代币的合约地址
    - 完成后，可以在钱包代币列表中看到刚才添加的代币