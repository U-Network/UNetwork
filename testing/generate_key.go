package testing

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strconv"
)

var DefaultFile = "./config.json"
var Generate = &cobra.Command {
	Use:     "generate",
	Short:   "generate private key and address",
	Long:    "This is the UNetwork used to generate the private key and address command to generate the private key to write to ./config.json",
	Example: `./cmd generate 10 or ./cmd generate n`,
	Run:     GenerateKey,
}

func GenerateKey(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Println("use: ", cmd.Example)
		return
	}
	n, _ := strconv.Atoi(args[0])
	if n <= 0 {
		log.Printf("use: %s Greater than 0", cmd.Example)
		return
	}
	f, err := os.Create(DefaultFile)
	if err != nil {
		log.Printf("Generate err: %s", err.Error())
		return
	}
	defer f.Close()
	writefile(bufio.NewWriter(f), n)
}

type ToJson struct {
	PrivKey string
	PubKey  string
	Address string
}

func writefile(w *bufio.Writer, n int) {
	for i := 0; i < n; i++ {
		s := CreateKey()
		w.WriteString(s)
	}
}

func CreateKey() string {
	key, _ := ethcrypto.GenerateKey()
	Pubkey := key.Public()
	PubkeyECDSA, _ := Pubkey.(*ecdsa.PublicKey)
	addr := ethcrypto.PubkeyToAddress(*PubkeyECDSA)

	var data *ToJson = &ToJson{
		PrivKey: hex.EncodeToString(ethcrypto.FromECDSA(key)),
		PubKey:  hex.EncodeToString(ethcrypto.FromECDSAPub(PubkeyECDSA)),
		Address: common.Bytes2Hex(addr[:]),
	}
	by, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		log.Printf("CreateKey error : %s", err.Error())
	}
	return string(by) + "\n "
}
