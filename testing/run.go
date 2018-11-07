package testing

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"io"
	"log"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
)

var (
	quit     = make(chan struct{})
	priv     *ecdsa.PrivateKey
	formAddr common.Address
	client   *ethclient.Client
	gNonce   uint64
)

var TestCmd = &cobra.Command{
	Use:     "run",
	Short:   "Running test cases",
	Long:    "Read the configuration run test case in config.json",
	Example: `./cmd run fromPrivatekey path URL or ./cmd run b074ccb81649d52e3f7f813459f5e9a9db6c85bab523f6d8519c5c4397beaffe ./config.json http://localhost:8545`,
	Run:     Run,
}

var BalanceCmd = &cobra.Command{
	Use:     "balance",
	Short:   "Running test cases",
	Long:    "Read the configuration run test case in config.json",
	Example: `./cmd balance ./comfig.json URL or ./cmd balance ./comfig.json http://localhost:8545`,
	Run:     GetBalance,
}

func ExecuteGetBalance(data *ToJson, flag uint32, wg *sync.WaitGroup) {
	var err error
	var balance *big.Int
	by := common.Hex2Bytes(data.Address)
	balance, err = client.BalanceAt(context.Background(), common.BytesToAddress(by), nil)
	if err != nil {
		log.Println("ExecuteGetBalance error: ", err.Error(), ", Serial number:", flag)
		wg.Done()
	} else {
		log.Printf("Address: %s, balance: %d\n", data.Address, balance)
		wg.Done()
	}
}

func GetBalance(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		log.Printf("use %s: ", cmd.Example)
		return
	}

	f, err := os.Open(args[0])
	if err != nil {
		log.Println("Read config.json error : ", err.Error())
	}
	defer f.Close()

	client, err = ethclient.Dial(args[1])
	if err != nil {
		log.Println("Create client error: ", err.Error())
		return
	}

	r := bufio.NewReader(f)
	var line string
	var flag uint32
	var wg sync.WaitGroup
	for {
		flag++
		line, err = r.ReadString('}')
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		var data *ToJson = new(ToJson)
		err = json.Unmarshal([]byte(line), data)
		if err != nil {
			log.Println("Unmarshal json err: ", err.Error())
			continue
		}
		wg.Add(1)
		go ExecuteGetBalance(data, flag, &wg)
	}
	wg.Wait()

}

func ExecuteTransaction(data *ToJson, flag uint32) {
	var err error
	var nonce uint64

	if atomic.LoadUint64(&gNonce) == 0 {
		nonce, err = client.NonceAt(context.Background(), formAddr, nil)
		if err != nil {
			log.Println("Unable to generate account random number on the network: ", err.Error(), "Serial number:", flag)
		}
		atomic.SwapUint64(&gNonce, nonce)
	} else {
		atomic.AddUint64(&gNonce, 1)
	}

	log.Println("nonce :", gNonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Printf("Failed to obtain suggested gas price from grid: %+v, Serial number: %d\n", err, flag)
	}

	auth := bind.NewKeyedTransactor(priv)
	auth.GasLimit = uint64(3000000)
	auth.GasPrice = gasPrice.Mul(gasPrice, big.NewInt(2))
	//auth.GasPrice = gasPrice
	auth.Value = big.NewInt(10)
	auth.Nonce = big.NewInt(int64(gNonce))

	var toPriv *ecdsa.PrivateKey
	toPriv, err = crypto.HexToECDSA(data.PrivKey)
	if err != nil {
		log.Println("from private key error: ", err.Error(), "Serial number: ", flag)
	}
	destinationAccount := bind.NewKeyedTransactor(toPriv)
	tx := types.NewTransaction(uint64(auth.Nonce.Int64()), destinationAccount.From, big.NewInt(1), auth.GasLimit, auth.GasPrice, nil)
	signedTx, err := auth.Signer(types.HomesteadSigner{}, auth.From, tx)
	if err != nil {
		fmt.Printf("Failed to sign the transaction to send ether a new account from a record buyer: %+v, Serial number: %d\n", err, flag)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Println("SendTransaction err: ", err, ", Serial number: ", flag)
	} else {
		log.Println("SendTransaction success Serial number:", flag)
	}
	//log.Println("from Address", common.Bytes2Hex(auth.From[:]), ",to Address: ", data.Address, ",Nonce :", auth.Nonce, ", GasPrice:", auth.GasPrice)
}

func Run(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		log.Printf("use %s: ", cmd.Example)
		return
	}
	f, err := os.Open(args[1])
	if err != nil {
		log.Println("Read config.json error : ", err.Error())
		return
	}
	defer f.Close()

	priv, err = crypto.HexToECDSA(args[0])
	if err != nil {
		log.Println("from private key error: ", err.Error())
		return
	}
	fromPubkey := priv.Public()
	fromPubkeyECDSA, _ := fromPubkey.(*ecdsa.PublicKey)
	formAddr = crypto.PubkeyToAddress(*fromPubkeyECDSA)

	client, err = ethclient.Dial(args[2])
	if err != nil {
		log.Println("Create client error: ", err.Error())
		return
	}

	r := bufio.NewReader(f)
	var line string
	var flag uint32
	//var wg sync.WaitGroup
	for {
		flag++
		line, err = r.ReadString('}')
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		var data *ToJson = new(ToJson)
		err = json.Unmarshal([]byte(line), data)
		if err != nil {
			log.Println("Unmarshal json err: ", err.Error())
			continue
		}

		ExecuteTransaction(data, flag)
	}
}
