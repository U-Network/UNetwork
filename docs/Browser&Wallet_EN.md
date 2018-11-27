# Blockchain browser and wallet usage documentation
## Deployment
1. First, pull the code from github
 
```shell
$ git clone git@github.com:U-Network/explorer.git
```
 
2. Install npm dependency
```shell
$ cd ./U-Network/explorer
$ npm install
```
 
3. Modify the browser configuration file
Set the parameters of `rpcUrl` and `ipcFile`
```shell
$ vim ./U-Network/explorer/app/config.json
// MacOS reference configuration information is as follows
{
  "rpcUrl" : "http://127.0.0.1:12321",
  "ipcFile" : "~/Library/.ethereum/unetwork.ipc"
}
```
 
4. Run the browser locally
```shell
$ cd ./U-Network/explorer
$ npm start
 
> UNetworkWalletExplorer@0.1.0 start /home/wwwroot/explorer
> node server.js
 
app listening on port 8000!
```
 
## Wallet usage
1. Enter the blockchain browser page `http://localhost:12321`
 
2. Create your own account,
	- Click `Create New Wallet`
	- Set your password, and click `Submit to Create`
	- Click `Download Keystore File(UTC/JSON)` to save file
	- Save the private key
 
3. Login to wallet
	- Click `login my wallet`
	- Two ways of login:
	1) Click “select file” to choose the wallet file downloaded previously, and enter password to login
	2) Enter the saved private key to login 
4. Transfer
	- Enter target address to `To Address` 
	- Enter amount at `Amount to Send` and send
 
## Your token for free
 
1. Enter the blockchain browser page  `http://localhost:12321`
 
2. Login to wallet (refer to previous passage for details)
 
3. Generate token
	- Click `Generate My Token` and set parameters as required
	- Wait for block confirmation of the transaction
	- Once created, you can see the tokens you generated in the wallet token list
 
4. If you receive the token from another party, make the token appear in the wallet
	- Obtain the contract address of the token
	- Enter the contract address at `Get Balance of Token`
	- After completion, you can view the token in the wallet token list