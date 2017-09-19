package sdk

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"

	"UGCNetwork/account"
	. "UGCNetwork/common"
	. "UGCNetwork/core/asset"
	"UGCNetwork/core/contract"
	"UGCNetwork/core/signature"
	"UGCNetwork/core/transaction"
)

// sortedAccounts used for sequential constructing program hash for verification
type sortedAccounts []*account.Account

func (sa sortedAccounts) Len() int      { return len(sa) }
func (sa sortedAccounts) Swap(i, j int) { sa[i], sa[j] = sa[j], sa[i] }
func (sa sortedAccounts) Less(i, j int) bool {
	if sa[i].ProgramHash.CompareTo(sa[j].ProgramHash) > 0 {
		return false
	} else {
		return true
	}
}

type sortedCoinsItem struct {
	input *transaction.UTXOTxInput
	coin  *account.Coin
}

// sortedCoins used for spend minor coins first
type sortedCoins []*sortedCoinsItem

func (sc sortedCoins) Len() int      { return len(sc) }
func (sc sortedCoins) Swap(i, j int) { sc[i], sc[j] = sc[j], sc[i] }
func (sc sortedCoins) Less(i, j int) bool {
	if sc[i].coin.Output.Value > sc[j].coin.Output.Value {
		return false
	} else {
		return true
	}
}

func sortCoinsByValue(coins map[*transaction.UTXOTxInput]*account.Coin) sortedCoins {
	var coinList sortedCoins
	for in, c := range coins {
		tmp := &sortedCoinsItem{
			input: in,
			coin:  c,
		}
		coinList = append(coinList, tmp)
	}
	sort.Sort(coinList)
	return coinList
}

func MakeRegTransaction(wallet account.Client, name string, value float64) (*transaction.Transaction, error) {
	admin, err := wallet.GetDefaultAccount()
	if err != nil {
		return nil, err
	}
	issuer := admin
	asset := &Asset{name, "description", byte(MaxPrecision), AssetType(Share), UTXO}
	transactionContract, err := contract.CreateSignatureContract(admin.PubKey())
	if err != nil {
		fmt.Println("CreateSignatureContract failed")
		return nil, err
	}
	tx, _ := transaction.NewRegisterAssetTransaction(asset, AssetValuetoFixed64(value), issuer.PubKey(), transactionContract.ProgramHash)
	txAttr := transaction.NewTxAttribute(transaction.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	tx.Attributes = make([]*transaction.TxAttribute, 0)
	tx.Attributes = append(tx.Attributes, &txAttr)
	if err := signTransaction(issuer, tx); err != nil {
		fmt.Println("sign regist transaction failed")
		return nil, err
	}

	return tx, nil
}

func MakeIssueTransaction(wallet account.Client, assetID Uint256, address string, value float64) (*transaction.Transaction, error) {
	admin, err := wallet.GetDefaultAccount()
	if err != nil {
		return nil, err
	}
	programHash, err := ToScriptHash(address)
	if err != nil {
		return nil, err
	}
	issueTxOutput := &transaction.TxOutput{
		AssetID:     assetID,
		Value:       AssetValuetoFixed64(value),
		ProgramHash: programHash,
	}
	outputs := []*transaction.TxOutput{issueTxOutput}
	tx, _ := transaction.NewIssueAssetTransaction(outputs)
	txAttr := transaction.NewTxAttribute(transaction.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	tx.Attributes = make([]*transaction.TxAttribute, 0)
	tx.Attributes = append(tx.Attributes, &txAttr)
	if err := signTransaction(admin, tx); err != nil {
		fmt.Println("sign issue transaction failed")
		return nil, err
	}
	return tx, nil
}

func MakeTransferTransaction(wallet account.Client, assetID Uint256, address string, value float64) (*transaction.Transaction, error) {
	mainAccount, err := wallet.GetDefaultAccount()
	if err != nil {
		return nil, err
	}
	receiverProgramHash, err := ToScriptHash(address)
	if err != nil {
		return nil, err
	}
	coins := wallet.GetCoins()
	input := []*transaction.UTXOTxInput{}
	output := []*transaction.TxOutput{}
	expected := AssetValuetoFixed64(value)
	var transfer Fixed64

	sorted := sortCoinsByValue(coins)
	for _, coinItem := range sorted {
		if coinItem.coin.Output.AssetID == assetID {
			input = append(input, coinItem.input)
			if coinItem.coin.Output.Value > expected {
				OutOfChange := &transaction.TxOutput{
					AssetID:     assetID,
					Value:       coinItem.coin.Output.Value - expected,
					ProgramHash: mainAccount.ProgramHash,
				}
				output = append(output, OutOfChange)
				transfer += expected
				expected = 0
				break
			} else if coinItem.coin.Output.Value == expected {
				transfer += expected
				expected = 0
				break
			} else if coinItem.coin.Output.Value < expected {
				transfer += coinItem.coin.Output.Value
				expected = expected - coinItem.coin.Output.Value
			}
		}
	}
	OutOfTx := &transaction.TxOutput{
		AssetID:     assetID,
		Value:       transfer,
		ProgramHash: receiverProgramHash,
	}
	output = append(output, OutOfTx)

	if expected > 0 {
		return nil, errors.New("Token is not enough")
	}

	txn, err := transaction.NewTransferAssetTransaction(input, output)
	if err != nil {
		return nil, err
	}
	txAttr := transaction.NewTxAttribute(transaction.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	txn.Attributes = make([]*transaction.TxAttribute, 0)
	txn.Attributes = append(txn.Attributes, &txAttr)

	// get account
	var accounts sortedAccounts
	for ref, coin := range coins {
		for _, in := range input {
			if in.ReferTxID == ref.ReferTxID && in.ReferTxOutputIndex == ref.ReferTxOutputIndex {
				foundDuplicate := false
				for _, tmp := range accounts {
					// skip duplicated program hash
					if coin.Output.ProgramHash == tmp.ProgramHash {
						foundDuplicate = true
						break
					}
				}
				if !foundDuplicate {
					accounts = append(accounts, wallet.GetAccountByProgramHash(coin.Output.ProgramHash))
				}
			}
		}
	}

	ctx := newContractContextWithoutProgramHashes(txn, len(accounts))
	// get public keys and signatures
	sort.Sort(accounts)
	i := 0
	for _, account := range accounts {
		signature, _ := signature.SignBySigner(txn, account)
		contract, _ := contract.CreateSignatureContract(account.PublicKey)
		ctx.MyAdd(contract, i, signature)
		i++
	}

	txn.SetPrograms(ctx.GetPrograms())

	return txn, nil
}

func signTransaction(signer *account.Account, tx *transaction.Transaction) error {
	signature, err := signature.SignBySigner(tx, signer)
	if err != nil {
		fmt.Println("SignBySigner failed")
		return err
	}
	transactionContract, err := contract.CreateSignatureContract(signer.PubKey())
	if err != nil {
		fmt.Println("CreateSignatureContract failed")
		return err
	}
	transactionContractContext := newContractContextWithoutProgramHashes(tx, 1)
	if err := transactionContractContext.AddContract(transactionContract, signer.PubKey(), signature); err != nil {
		fmt.Println("SaveContract failed")
		return err
	}
	tx.SetPrograms(transactionContractContext.GetPrograms())
	return nil
}

func newContractContextWithoutProgramHashes(data signature.SignableData, length int) *contract.ContractContext {
	return &contract.ContractContext{
		Data:       data,
		Codes:      make([][]byte, length),
		Parameters: make([][][]byte, length),
	}
}
