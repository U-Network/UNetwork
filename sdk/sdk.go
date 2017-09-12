package sdk

import (
	"fmt"
	"math/rand"
	"strconv"

	"UGCNetwork/account"
	. "UGCNetwork/common"
	. "UGCNetwork/core/asset"
	"UGCNetwork/core/contract"
	"UGCNetwork/core/signature"
	"UGCNetwork/core/transaction"
	"errors"
)

const (
	REFERTXHASHLEN = 64
)

func MakeRegTransaction(wallet account.Client, name string, value Fixed64) (*transaction.Transaction, error) {
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
	tx, _ := transaction.NewRegisterAssetTransaction(asset, value, issuer.PubKey(), transactionContract.ProgramHash)
	txAttr := transaction.NewTxAttribute(transaction.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	tx.Attributes = make([]*transaction.TxAttribute, 0)
	tx.Attributes = append(tx.Attributes, &txAttr)
	if err := signTransaction(issuer, tx); err != nil {
		fmt.Println("sign regist transaction failed")
		return nil, err
	}

	return tx, nil
}

func MakeIssueTransaction(wallet account.Client, assetID Uint256, programHash Uint160, value Fixed64) (*transaction.Transaction, error) {
	admin, err := wallet.GetDefaultAccount()
	if err != nil {
		return nil, err
	}

	issueTxOutput := &transaction.TxOutput{
		AssetID:     assetID,
		Value:       value,
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

func MakeTransferTransaction(wallet account.Client, assetID Uint256, programhash Uint160, value Fixed64) (*transaction.Transaction, error) {
	mainAccount, err := wallet.GetDefaultAccount()
	if err != nil {
		return nil, err
	}
	coins := wallet.GetCoins()
	input := []*transaction.UTXOTxInput{}
	output := []*transaction.TxOutput{}
	expected := value
	for ref, coin := range coins {
		if coin.Output.ProgramHash == programhash {
			input = append(input, ref)
			if coin.Output.Value > expected {
				transfered := transaction.TxOutput{
					AssetID:     assetID,
					Value:       expected,
					ProgramHash: programhash,
				}
				getChanged := transaction.TxOutput{
					AssetID:     assetID,
					Value:       coin.Output.Value - expected,
					ProgramHash: mainAccount.ProgramHash,
				}
				output = append(output, &transfered, &getChanged)
				expected = 0
				break
			} else if coin.Output.Value == expected {
				transfered := transaction.TxOutput{
					AssetID:     assetID,
					Value:       expected,
					ProgramHash: programhash,
				}
				output = append(output, &transfered)
				expected = 0
				break
			} else if coin.Output.Value < expected {
				transfered := transaction.TxOutput{
					AssetID:     assetID,
					Value:       coin.Output.Value,
					ProgramHash: programhash,
				}
				output = append(output, &transfered)
				expected = expected - coin.Output.Value
			}
		}
	}
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

	if err := signTransaction(mainAccount, txn); err != nil {
		fmt.Println("sign transfer transaction failed")
		return nil, err
	}
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
	transactionContractContext := newContractContextWithoutProgramHashes(tx)
	if err := transactionContractContext.AddContract(transactionContract, signer.PubKey(), signature); err != nil {
		fmt.Println("SaveContract failed")
		return err
	}
	tx.SetPrograms(transactionContractContext.GetPrograms())
	return nil
}

func newContractContextWithoutProgramHashes(data signature.SignableData) *contract.ContractContext {
	return &contract.ContractContext{
		Data:       data,
		Codes:      make([][]byte, 1),
		Parameters: make([][][]byte, 1),
	}
}
