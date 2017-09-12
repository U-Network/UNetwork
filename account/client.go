package account

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	. "UGCNetwork/common"
	"UGCNetwork/common/config"
	"UGCNetwork/common/log"
	"UGCNetwork/common/password"
	"UGCNetwork/core/contract"
	ct "UGCNetwork/core/contract"
	"UGCNetwork/core/ledger"
	sig "UGCNetwork/core/signature"
	"UGCNetwork/core/transaction"
	"UGCNetwork/crypto"
	. "UGCNetwork/errors"
	"UGCNetwork/events"
	"UGCNetwork/net/protocol"
	//"encoding/binary"
)

const (
	DefaultBookKeeperCount = 4
	WalletFileName         = "wallet.dat"
	MAINACCOUNT            = "main-account"
	SUBACCOUNT             = "sub-account"
)

type Client interface {
	Sign(context *ct.ContractContext) bool
	ContainsAccount(pubKey *crypto.PubKey) bool
	GetAccount(pubKey *crypto.PubKey) (*Account, error)
	GetAccountByProgramHash(programHash Uint160) *Account
	GetAccounts() []*Account
	GetDefaultAccount() (*Account, error)
	GetCoins() map[*transaction.UTXOTxInput]*Coin
}

type ClientImpl struct {
	mu sync.Mutex

	path      string
	iv        []byte
	masterKey []byte

	mainAccount Uint160
	accounts    map[Uint160]*Account
	contracts   map[Uint160]*ct.Contract
	coins       map[*transaction.UTXOTxInput]*Coin

	watchOnly     []Uint160
	currentHeight uint32

	FileStore
	isRunning bool

	newBlockSaved events.Subscriber
}

func Create(path string, passwordKey []byte) (*ClientImpl, error) {
	client := NewClient(path, passwordKey, true)
	if client == nil {
		return nil, errors.New("client nil")
	}
	account, err := client.CreateAccount()
	if err != nil {
		return nil, err
	}
	if err := client.CreateContract(account); err != nil {
		return nil, err
	}
	client.mainAccount = account.ProgramHash

	account1, err := client.CreateAccount()
	if err != nil {
		return nil, err
	}
	if err := client.CreateContract(account1); err != nil {
		return nil, err
	}

	return client, nil
}

func Open(path string, passwordKey []byte) (*ClientImpl, error) {
	client := NewClient(path, passwordKey, false)
	if client == nil {
		return nil, errors.New("client nil")
	}

	client.accounts = client.LoadAccounts()
	if client.accounts == nil {
		return nil, errors.New("Load accounts failure")
	}
	client.contracts = client.LoadContracts()
	if client.contracts == nil {
		return nil, errors.New("Load contracts failure")
	}

	loadedCoin, err := client.LoadCoins()
	if err != nil {
		return nil, err
	}
	for input, coin := range loadedCoin {
		client.coins[input] = coin
	}

	return client, nil
}

func Recover(path string, password []byte, privateKeyHex string) (*ClientImpl, error) {
	client := NewClient(path, password, true)
	if client == nil {
		return nil, errors.New("client nil")
	}

	privateKeyBytes, err := HexToBytes(privateKeyHex)
	if err != nil {
		return nil, err
	}

	// recover Account
	account, err := client.CreateAccountByPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}

	// recover contract
	if err := client.CreateContract(account); err != nil {
		return nil, err
	}

	return client, nil
}

func (client *ClientImpl) ProcessBlock(v interface{}) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if block, ok := v.(*ledger.Block); ok {
		blockHash := block.Hash()
		savedBlock, _ := ledger.DefaultLedger.GetBlockWithHash(blockHash)
		fmt.Println("ProcessBlock")

		var needUpdate bool
		// received coins
		for _, tx := range savedBlock.Transactions {
			for index, output := range tx.Outputs {
				if _, ok := client.contracts[output.ProgramHash]; ok {
					input := &transaction.UTXOTxInput{ReferTxID: tx.Hash(), ReferTxOutputIndex: uint16(index)}
					if _, ok := client.coins[input]; !ok {
						newCoin := &Coin{Output: output}
						client.coins[input] = newCoin
						needUpdate = true
					}
				}
			}
		}

		// spent coins
		for _, tx := range savedBlock.Transactions {
			for _, input := range tx.UTXOInputs {
				for k := range client.coins {
					if k.ReferTxOutputIndex == input.ReferTxOutputIndex && k.ReferTxID == input.ReferTxID {
						delete(client.coins, k)
						needUpdate = true
					}
				}
			}
		}

		// update wallet store
		if needUpdate {
			if err := client.SaveCoins(client.coins); err != nil {
				fmt.Println("save coin error")
			}
		}

		// update height
		client.currentHeight++

		//client.SaveStoredData("Height", )
	}
}

func NewClient(path string, password []byte, create bool) *ClientImpl {
	client := &ClientImpl{
		path:          path,
		accounts:      map[Uint160]*Account{},
		contracts:     map[Uint160]*ct.Contract{},
		coins:         map[*transaction.UTXOTxInput]*Coin{},
		currentHeight: 0,
		FileStore:     FileStore{path: path},
		isRunning:     true,
		newBlockSaved: nil,
	}
	//TODO: use isRunning instead
	if ledger.DefaultLedger != nil && ledger.DefaultLedger.Blockchain != nil {
		client.newBlockSaved = ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, client.ProcessBlock)
		client.currentHeight = ledger.DefaultLedger.GetLocalBlockChainHeight()
	}

	passwordKey := crypto.ToAesKey(password)
	if create {
		//create new client
		client.iv = make([]byte, 16)
		client.masterKey = make([]byte, 32)
		client.watchOnly = []Uint160{}

		//generate random number for iv/masterkey
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < 16; i++ {
			client.iv[i] = byte(r.Intn(256))
		}
		for i := 0; i < 32; i++ {
			client.masterKey[i] = byte(r.Intn(256))
		}

		//new client store (build DB)
		client.BuildDatabase(path)

		pwdhash := sha256.Sum256(passwordKey)
		if err := client.SaveStoredData("PasswordHash", pwdhash[:]); err != nil {
			log.Error(err)
			return nil
		}
		if err := client.SaveStoredData("IV", client.iv[:]); err != nil {
			log.Error(err)
			return nil
		}

		//if err := client.SaveHeight(client.currentHeight); err != nil {
		//	log.Error(err)
		//	return nil
		//}

		aesmk, err := crypto.AesEncrypt(client.masterKey[:], passwordKey, client.iv)
		if err != nil {
			log.Error(err)
			return nil
		}
		if err := client.SaveStoredData("MasterKey", aesmk); err != nil {
			log.Error(err)
			return nil
		}
	} else {
		if ok := client.verifyPasswordKey(passwordKey); !ok {
			return nil
		}
		var err error
		client.iv, err = client.LoadStoredData("IV")
		if err != nil {
			fmt.Println("error: failed to load iv")
			return nil
		}
		encryptedMasterKey, err := client.LoadStoredData("MasterKey")
		if err != nil {
			fmt.Println("error: failed to load master key")
			return nil
		}
		client.masterKey, err = crypto.AesDecrypt(encryptedMasterKey, passwordKey, client.iv)
		if err != nil {
			fmt.Println("error: failed to decrypt master key")
			return nil
		}
	}
	ClearBytes(passwordKey, len(passwordKey))

	return client
}

func (cl *ClientImpl) GetDefaultAccount() (*Account, error) {
	return cl.GetAccountByProgramHash(cl.mainAccount), nil
}

func (cl *ClientImpl) GetAccount(pubKey *crypto.PubKey) (*Account, error) {
	signatureRedeemScript, err := contract.CreateSignatureRedeemScript(pubKey)
	if err != nil {
		return nil, NewDetailErr(err, ErrNoCode, "CreateSignatureRedeemScript failed")
	}
	programHash, err := ToCodeHash(signatureRedeemScript)
	if err != nil {
		return nil, NewDetailErr(err, ErrNoCode, "ToCodeHash failed")
	}
	return cl.GetAccountByProgramHash(programHash), nil
}

func (cl *ClientImpl) GetAccountByProgramHash(programHash Uint160) *Account {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if account, ok := cl.accounts[programHash]; ok {
		return account
	}
	return nil
}

func (cl *ClientImpl) GetContract(programHash Uint160) *ct.Contract {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if contract, ok := cl.contracts[programHash]; ok {
		return contract
	}
	return nil
}

func (cl *ClientImpl) ChangePassword(oldPassword []byte, newPassword []byte) bool {
	// check password
	oldPasswordKey := crypto.ToAesKey(oldPassword)
	if !cl.verifyPasswordKey(oldPasswordKey) {
		fmt.Println("error: password verification failed")
		return false
	}

	// encrypt master key with new password
	newPasswordKey := crypto.ToAesKey(newPassword)
	newMasterKey, err := crypto.AesEncrypt(cl.masterKey, newPasswordKey, cl.iv)
	if err != nil {
		fmt.Println("error: set new password failed")
		return false
	}

	// update wallet file
	newPasswordHash := sha256.Sum256(newPasswordKey)
	if err := cl.SaveStoredData("PasswordHash", newPasswordHash[:]); err != nil {
		fmt.Println("error: wallet update failed(password hash)")
		return false
	}
	if err := cl.SaveStoredData("MasterKey", newMasterKey); err != nil {
		fmt.Println("error: wallet update failed (encrypted master key)")
		return false
	}
	ClearBytes(newPasswordKey, len(newPasswordKey))

	return true
}

func (cl *ClientImpl) ContainsAccount(pubKey *crypto.PubKey) bool {
	signatureRedeemScript, err := contract.CreateSignatureRedeemScript(pubKey)
	if err != nil {
		return false
	}
	programHash, err := ToCodeHash(signatureRedeemScript)
	if err != nil {
		return false
	}
	if cl.GetAccountByProgramHash(programHash) != nil {
		return true
	} else {
		return false
	}
}

func (cl *ClientImpl) Sign(context *ct.ContractContext) bool {
	log.Debug()
	fSuccess := false
	for i, hash := range context.ProgramHashes {
		contract := cl.GetContract(hash)
		if contract == nil {
			continue
		}
		account := cl.GetAccountByProgramHash(hash)
		if account == nil {
			continue
		}

		signature, err := sig.SignBySigner(context.Data, account)
		if err != nil {
			return fSuccess
		}
		err = context.AddContract(contract, account.PublicKey, signature)

		if err != nil {
			fSuccess = false
		} else {
			if i == 0 {
				fSuccess = true
			}
		}
	}
	return fSuccess
}

func (cl *ClientImpl) verifyPasswordKey(passwordKey []byte) bool {
	savedPasswordHash, err := cl.LoadStoredData("PasswordHash")
	if err != nil {
		fmt.Println("error: failed to load password hash")
		return false
	}
	if savedPasswordHash == nil {
		fmt.Println("error: saved password hash is nil")
		return false
	}
	passwordHash := sha256.Sum256(passwordKey)
	///ClearBytes(passwordKey, len(passwordKey))
	if !IsEqualBytes(savedPasswordHash, passwordHash[:]) {
		fmt.Println("error: password wrong")
		return false
	}
	return true
}

func (cl *ClientImpl) EncryptPrivateKey(prikey []byte) ([]byte, error) {
	enc, err := crypto.AesEncrypt(prikey, cl.masterKey, cl.iv)
	if err != nil {
		return nil, err
	}

	return enc, nil
}

func (cl *ClientImpl) DecryptPrivateKey(prikey []byte) ([]byte, error) {
	if prikey == nil {
		return nil, NewDetailErr(errors.New("The PriKey is nil"), ErrNoCode, "")
	}
	if len(prikey) != 96 {
		return nil, NewDetailErr(errors.New("The len of PriKeyEnc is not 96bytes"), ErrNoCode, "")
	}

	dec, err := crypto.AesDecrypt(prikey, cl.masterKey, cl.iv)
	if err != nil {
		return nil, err
	}

	return dec, nil
}

// CreateAccount create a new Account then save it
func (cl *ClientImpl) CreateAccount() (*Account, error) {
	account, err := NewAccount()
	if err != nil {
		return nil, err
	}
	if err := cl.SaveAccount(account); err != nil {
		return nil, err
	}

	return account, nil
}

func (cl *ClientImpl) DeleteAccount(programHash Uint160) error {
	// remove from memory
	delete(cl.accounts, programHash)
	// remove from db
	return cl.DeleteAccountData(ToHexString(programHash.ToArray()))
}

func (cl *ClientImpl) CreateAccountByPrivateKey(privateKey []byte) (*Account, error) {
	account, err := NewAccountWithPrivatekey(privateKey)
	if err != nil {
		return nil, err
	}

	if err := cl.SaveAccount(account); err != nil {
		return nil, err
	}

	return account, nil
}

// SaveAccount saves a Account to memory and db
func (cl *ClientImpl) SaveAccount(ac *Account) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// save Account to memory
	programHash := ac.ProgramHash
	cl.accounts[programHash] = ac

	decryptedPrivateKey := make([]byte, 96)
	temp, err := ac.PublicKey.EncodePoint(false)
	if err != nil {
		return err
	}
	for i := 1; i <= 64; i++ {
		decryptedPrivateKey[i-1] = temp[i]
	}
	for i := len(ac.PrivateKey) - 1; i >= 0; i-- {
		decryptedPrivateKey[96+i-len(ac.PrivateKey)] = ac.PrivateKey[i]
	}
	encryptedPrivateKey, err := cl.EncryptPrivateKey(decryptedPrivateKey)
	if err != nil {
		return err
	}
	ClearBytes(decryptedPrivateKey, 96)

	// save Account keys to db
	err = cl.SaveAccountData(programHash.ToArray(), encryptedPrivateKey)
	if err != nil {
		return err
	}

	return nil
}

// LoadAccounts loads all accounts from db to memory
func (cl *ClientImpl) LoadAccounts() map[Uint160]*Account {
	accounts := map[Uint160]*Account{}

	account, err := cl.LoadAccountData()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, a := range account {
		if a.Type == MAINACCOUNT {
			p, _ := HexToBytes(a.ProgramHash)
			cl.mainAccount, _ = Uint160ParseFromBytes(p)
		}
		encryptedKeyPair, _ := HexToBytes(a.PrivateKeyEncrypted)
		keyPair, err := cl.DecryptPrivateKey(encryptedKeyPair)
		if err != nil {
			log.Error(err)
			continue
		}
		privateKey := keyPair[64:96]
		ac, err := NewAccountWithPrivatekey(privateKey)
		accounts[ac.ProgramHash] = ac
	}

	return accounts
}

// CreateContract create a new contract then save it
func (cl *ClientImpl) CreateContract(account *Account) error {
	contract, err := contract.CreateSignatureContract(account.PubKey())
	if err != nil {
		return err
	}
	if err := cl.SaveContract(contract); err != nil {
		return err
	}
	return nil
}

func (cl *ClientImpl) DeleteContract(programHash Uint160) error {
	delete(cl.contracts, programHash)
	return cl.DeleteContractData(ToHexString(programHash.ToArray()))
}

// SaveContract saves a contract to memory and db
func (cl *ClientImpl) SaveContract(ct *contract.Contract) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.accounts[ct.ProgramHash] == nil {
		return NewDetailErr(errors.New("SaveContract(): contract.OwnerPubkeyHash not in []accounts"), ErrNoCode, "")
	}

	// save contract to memory
	cl.contracts[ct.ProgramHash] = ct

	// save contract to db
	return cl.SaveContractData(ct)
}

// LoadContracts loads all contracts from db to memory
func (cl *ClientImpl) LoadContracts() map[Uint160]*ct.Contract {
	contracts := map[Uint160]*ct.Contract{}

	contract, err := cl.LoadContractData()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, c := range contract {
		rawdata, _ := HexToBytes(c.RawData)
		rdreader := bytes.NewReader(rawdata)
		ct := new(ct.Contract)
		ct.Deserialize(rdreader)

		programHash, _ := HexToBytes(c.ProgramHash)
		programhash, _ := Uint160ParseFromBytes(programHash)
		ct.ProgramHash = programhash
		contracts[ct.ProgramHash] = ct
	}

	return contracts
}

func clientIsDefaultBookKeeper(publicKey string) bool {
	for _, bookKeeper := range config.Parameters.BookKeepers {
		if strings.Compare(bookKeeper, publicKey) == 0 {
			return true
		}
	}
	return false
}

func nodeType(typeName string) int {
	if "service" == config.Parameters.NodeType {
		return protocol.SERVICENODE
	} else {
		return protocol.VERIFYNODE
	}
}

func GetClient() Client {
	if !FileExisted(WalletFileName) {
		log.Fatal(fmt.Sprintf("No %s detected, please create a wallet by using command line.", WalletFileName))
		os.Exit(1)
	}
	passwd, err := password.GetAccountPassword()
	if err != nil {
		log.Fatal("Get password error.")
		os.Exit(1)
	}
	c, err := Open(WalletFileName, passwd)
	if err != nil {
		return nil
	}
	return c
}

func GetBookKeepers() []*crypto.PubKey {
	var pubKeys = []*crypto.PubKey{}
	sort.Strings(config.Parameters.BookKeepers)
	for _, key := range config.Parameters.BookKeepers {
		pubKey := []byte(key)
		pubKey, err := hex.DecodeString(key)
		// TODO Convert the key string to byte
		k, err := crypto.DecodePoint(pubKey)
		if err != nil {
			log.Error("Incorrectly book keepers key")
			return nil
		}
		pubKeys = append(pubKeys, k)
	}

	return pubKeys
}

func (client *ClientImpl) GetCoins() map[*transaction.UTXOTxInput]*Coin {
	client.mu.Lock()
	defer client.mu.Unlock()

	return client.coins
}

func (client *ClientImpl) GetAccounts() []*Account {
	client.mu.Lock()
	defer client.mu.Unlock()

	accounts := []*Account{}
	for _, v := range client.accounts {
		accounts = append(accounts, v)
	}
	return accounts
}
