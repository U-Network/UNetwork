package global

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"io/ioutil"
	"os"
	"path/filepath"
)

func SaveEthPrivateKey(peernum int, dir string) error {
	for i := 1; i <= peernum; i++ {
		pathdir := filepath.Join(dir, fmt.Sprintf("config%d", i))
		filename := filepath.Join(pathdir, "eth_privatekey.json")
		content, err := GenEthPrivatekey()
		if err != nil {
			return err
		}
		SaveContent(content, filename)
		//logger.Info("Generated genesis file", "path", genFile)
	}
	return nil
}

func GenEthPrivatekey() (string, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), crand.Reader)
	if err != nil {
		return "", err
	}
	Address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	Addressstr := hex.EncodeToString(Address[:])
	keyBytes := math.PaddedBigBytes(privateKeyECDSA.D, 32)
	Prvstr := hex.EncodeToString(keyBytes[:])
	content := "{\"Address\":\"" + Addressstr + "\",\"Privatekey\":\"" + Prvstr + "\"}\n"
	return content, nil
}

func SaveContent(content string, filename string) error {

	dstFile, err := os.Create(filename)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	defer dstFile.Close()
	dstFile.WriteString(content)
	return nil
}

func ReadFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetEthAddressfromfile(filename string) (string, error) {
	type ethPrvkey struct {
		Address    string
		Privatekey string
	}
	epkey := ethPrvkey{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(data, &epkey)
	if err != nil {
		return "", err
	}
	return epkey.Address, nil
}
