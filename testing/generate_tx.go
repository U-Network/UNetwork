package testing

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"log"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

var GenerateTx = &cobra.Command{
	Use:     "generate_tx",
	Short:   "Generate transaction for testing TPS.",
	Long:    "This is the UNetwork generated transaction flag used to test the transaction processing system.",
	Example: `./cmd generate_tx toPrivateKey 10 URL or ./cmd generate_tx toPrivateKey count URL`,
	Run:     GenerateTransaction,
}

var (
	quitFl int32
)

type Queue struct {
}

func GenerateTransaction(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		log.Println("see Example: ", cmd.Example)
		return
	}

	count, err := strconv.Atoi(args[1])
	if err != nil {
		log.Println("args[1] is not number error: ", err.Error())
		return
	}
	priv, err = crypto.HexToECDSA(args[0])
	if err != nil {
		log.Println("from private key error: ", err.Error())
		return
	}

	for i := 0; i < runtime.NumCPU()*2; i++ {
		go taskLoop(priv, uint64(count), args[2])
	}
	sigc := make(chan os.Signal)
	signal.Notify(sigc, syscall.SIGINT)
	defer signal.Stop(sigc)
	<-sigc
	if atomic.SwapInt32(&quitFl, 1) == 0 {
		<-time.NewTicker(time.Duration(5)).C
	} else {
		return
	}
}

func taskLoop(toPriv *ecdsa.PrivateKey, count uint64, URL string) {
	client, err := ethclient.Dial(URL)
	if err != nil {
		log.Println("Create client error: ", err.Error())
		return
	}

	var flag uint64
	for {
		fromPriv, _ := ethcrypto.GenerateKey()
		auth := bind.NewKeyedTransactor(fromPriv)
		auth.GasLimit = uint64(3000000)
		auth.GasPrice = big.NewInt(0)
		auth.Value = big.NewInt(10)
		auth.Nonce = big.NewInt(int64(0))

		destinationAccount := bind.NewKeyedTransactor(fromPriv)
		tx := types.NewTransaction(uint64(auth.Nonce.Int64()), destinationAccount.From, big.NewInt(0), auth.GasLimit, auth.GasPrice, nil)
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

		flag++
		if atomic.LoadInt32(&quitFl) != 0 {
			return
		}
	}
}
