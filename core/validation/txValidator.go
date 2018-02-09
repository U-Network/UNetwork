package validation

import (
	"errors"
	"fmt"
	"math"

	. "UNetwork/common"
	"UNetwork/common/config"
	"UNetwork/common/log"
	"UNetwork/core/asset"
	"UNetwork/core/forum"
	"UNetwork/core/ledger"
	tx "UNetwork/core/transaction"
	"UNetwork/core/transaction/payload"
	"UNetwork/crypto"
	. "UNetwork/errors"

)

func VerifyTransaction(txn *tx.Transaction) ErrCode {

	if err := CheckDuplicateInput(txn); err != nil {
		log.Warn("[VerifyTransaction],", err)
		return ErrDuplicateInput
	}

	if err := CheckTransactionOutput(txn); err != nil {
		log.Warn("[VerifyTransaction],", err)
		return ErrInvalidOutput
	}

	if err := CheckAssetPrecision(txn); err != nil {
		log.Warn("[VerifyTransaction],", err)
		return ErrAssetPrecision
	}

	if err := CheckTransactionBalance(txn); err != nil {
		log.Warn("[VerifyTransaction],", err)
		return ErrTransactionBalance
	}

	if err := CheckAttributeProgram(txn); err != nil {
		log.Warn("[VerifyTransaction],", err)
		return ErrAttributeProgram
	}

	if err := CheckTransactionPayload(txn); err != nil {
		log.Warn("[VerifyTransaction],", err)
		return ErrTransactionPayload
	}

	if err := CheckTransactionContracts(txn); err != nil {
		log.Warn("[VerifyTransaction],", err)
		return ErrTransactionContracts
	}

	return ErrNoError
}

// VerifyTransactionWithBlock verifys a transaction with current transaction pool in memory
func VerifyTransactionWithBlock(TxPool []*tx.Transaction) error {
	//initial
	txnlist := make(map[Uint256]*tx.Transaction, 0)
	var txPoolInputs []string
	//sum all inputs in TxPool
	for _, Tx := range TxPool {
		for _, UTXOinput := range Tx.UTXOInputs {
			txPoolInputs = append(txPoolInputs, UTXOinput.ToString())
		}
	}
	//start check
	for _, txn := range TxPool {
		//1.check weather have duplicate transaction.
		if _, exist := txnlist[txn.Hash()]; exist {
			return errors.New("[VerifyTransactionWithBlock], duplicate transaction exist in block.")
		} else {
			txnlist[txn.Hash()] = txn
		}
		//2.check Duplicate Utxo input
		if err := CheckDuplicateUtxoInBlock(txn, txPoolInputs); err != nil {
			return err
		}
		//3.check issue amount
		switch txn.TxType {
		case tx.IssueAsset:
			//TODO: use delta mode to improve performance
			results := txn.GetMergedAssetIDValueFromOutputs()
			for k, _ := range results {
				//Get the Asset amount when RegisterAsseted.
				trx, err := tx.TxStore.GetTransaction(k)
				if trx.TxType != tx.RegisterAsset {
					return errors.New("[VerifyTransaction], TxType is illegal.")
				}
				AssetReg := trx.Payload.(*payload.RegisterAsset)

				//Get the amount has been issued of this assetID
				var quantity_issued Fixed64
				if AssetReg.Amount < Fixed64(0) {
					continue
				} else {
					quantity_issued, err = tx.TxStore.GetQuantityIssued(k)
					if err != nil {
						return errors.New("[VerifyTransaction], GetQuantityIssued failed.")
					}
				}

				//calc the amounts in txPool which are also IssueAsset
				var txPoolAmounts Fixed64
				for _, t := range TxPool {
					if t.TxType == tx.IssueAsset {
						outputResult := t.GetMergedAssetIDValueFromOutputs()
						for txidInPool, txValueInPool := range outputResult {
							if txidInPool == k {
								txPoolAmounts = txPoolAmounts + txValueInPool
							}
						}
					}
				}

				//calc weather out off the amount when Registed.
				//AssetReg.Amount : amount when RegisterAsset of this assedID
				//quantity_issued : amount has been issued of this assedID
				//txPoolAmounts   : amount in transactionPool of this assedID of issue transaction.
				if AssetReg.Amount-quantity_issued < txPoolAmounts {
					return errors.New("[VerifyTransaction], Amount check error.")
				}
			}
		}

	}

	return nil
}

func CheckLockedAsset(txn *tx.Transaction, ledger *ledger.Ledger) error {
	// onlu check locked asset for transfer transaction
	if txn.TxType != tx.TransferAsset {
		return nil
	}

	// get spend asset amount for each program hash and asset ID pair
	result := make(map[Uint160]map[Uint256]Fixed64)
	inputAsset, err := txn.GetReference()
	if err != nil {
		return err
	}
	for _, referOutput := range inputAsset {
		if _, ok := result[referOutput.ProgramHash]; !ok {
			result[referOutput.ProgramHash] = make(map[Uint256]Fixed64)
		}
		if _, ok := result[referOutput.ProgramHash][referOutput.AssetID]; !ok {
			result[referOutput.ProgramHash][referOutput.AssetID] = referOutput.Value
		} else {
			result[referOutput.ProgramHash][referOutput.AssetID] += referOutput.Value
		}
	}
	for _, output := range txn.Outputs {
		if _, ok := result[output.ProgramHash]; ok {
			if _, find := result[output.ProgramHash][output.AssetID]; find {
				result[output.ProgramHash][output.AssetID] -= output.Value
			}
		}
	}

	// check if this transaction spends the locked asset
	for programHash, assets := range result {
		for assetID, spend := range assets {
			total, locked, err := ledger.Store.GetAvailableAsset(programHash, assetID)
			if err != nil {
				return err
			}
			if total < spend+locked {
				return errors.New("token is not enough, locked token can't be used.")
			}
		}
	}

	return nil
}

func VerifyTransactionWithLedger(txn *tx.Transaction, ledger *ledger.Ledger) ErrCode {

	if exist := ledger.Store.IsTxHashDuplicate(txn.Hash()); exist {
		log.Info("[VerifyTransactionWithLedger] duplicated transaction detected.")
		return ErrTxHashDuplicate
	}

	if IsDoubleSpend(txn, ledger) {
		log.Info("[VerifyTransactionWithLedger] double spend checking failed.")
		return ErrDoubleSpend
	}

	if err := CheckLockedAsset(txn, ledger); err != nil {
		log.Info("[VerifyTransactionWithLedger] .")
		return ErrLockedAsset
	}

	return ErrNoError
}

//validate the transaction of duplicate UTXO input
func CheckDuplicateInput(tx *tx.Transaction) error {
	if len(tx.UTXOInputs) == 0 {
		return nil
	}
	for i, utxoin := range tx.UTXOInputs {
		for j := 0; j < i; j++ {
			if utxoin.ReferTxID == tx.UTXOInputs[j].ReferTxID && utxoin.ReferTxOutputIndex == tx.UTXOInputs[j].ReferTxOutputIndex {
				return errors.New("invalid transaction")
			}
		}
	}
	return nil
}

func CheckTransactionOutput(txn *tx.Transaction) error {
	if txn.TxType == tx.Withdrawal {
		if len(txn.Outputs) != 1 {
			return errors.New("invaild transaction output num in withdrawal transaction")
		}
		_, err := txn.Outputs[0].ProgramHash.ToAddress()
		if err != nil {
			return errors.New("invaild withdrawal address")
		}

		//TODO: check asset ID
		payee := txn.Payload.(*payload.Withdrawal).Payee
		availableTokenInfo, err := ledger.DefaultLedger.Store.GetAvailableTokenInfo(payee)
		if err != nil {
			return err
		}
		if availableTokenInfo.Number <= Fixed64(0) {
			return errors.New("All tokens has been withdrawn")
		}
		if txn.Outputs[0].Value > availableTokenInfo.Number {
			return errors.New("available tokens is not enough")
		}
	}

	return nil
}

func CheckDuplicateUtxoInBlock(tx *tx.Transaction, txPoolInputs []string) error {
	var txInputs []string
	for _, t := range tx.UTXOInputs {
		txInputs = append(txInputs, t.ToString())
	}
	for _, i := range txInputs {
		for _, j := range txPoolInputs {
			if i == j {
				return errors.New("Duplicated UTXO inputs found in tx pool")
			}
		}
	}
	return nil
}

func IsDoubleSpend(tx *tx.Transaction, ledger *ledger.Ledger) bool {
	return ledger.IsDoubleSpend(tx)
}

func CheckAssetPrecision(Tx *tx.Transaction) error {
	if len(Tx.Outputs) == 0 {
		return nil
	}
	assetOutputs := make(map[Uint256][]*tx.TxOutput, len(Tx.Outputs))

	for _, v := range Tx.Outputs {
		assetOutputs[v.AssetID] = append(assetOutputs[v.AssetID], v)
	}
	for k, outputs := range assetOutputs {
		asset, err := ledger.DefaultLedger.GetAsset(k)
		if err != nil {
			return errors.New("The asset not exist in local blockchain.")
		}
		precision := asset.Precision
		for _, output := range outputs {
			if checkAmountPrecise(output.Value, precision) {
				return errors.New("The precision of asset is incorrect.")
			}
		}
	}
	return nil
}

func CheckTransactionBalance(Tx *tx.Transaction) error {
	for _, v := range Tx.Outputs {
		if v.Value <= Fixed64(0) {
			return errors.New("Invalid transaction UTXO output.")
		}
	}
	if Tx.TxType == tx.IssueAsset || Tx.TxType == tx.Withdrawal {
		if len(Tx.UTXOInputs) > 0 {
			return errors.New("Invalide Issue transaction.")
		}
		return nil
	}
	results, err := Tx.GetTransactionResults()
	if err != nil {
		return err
	}
	for k, v := range results {
		// if transaction fee is not configured, input amount must equal to output
		if fee, ok := config.Parameters.TransactionFee["Transfer"]; !ok {
			if v != 0 {
				return errors.New(fmt.Sprintf("AssetID %x in Transfer transactions %x, balance unmatched when fee is not set.", k, Tx.Hash()))
			}
		} else {
			switch fee {
			case 0.0:
				if v != 0 {
					return errors.New(fmt.Sprintf("AssetID %x in Transfer transactions %x, balance unmatched when fee is 0.", k, Tx.Hash()))
				}
			default:
				// due to non-zero transaction fee, the input amount must > output amount
				if v <= 0 {
					return errors.New(fmt.Sprintf("AssetID %x in Transfer transactions %x, output >= input.", k, Tx.Hash()))
				}
			}
		}
	}
	return nil
}

func CheckAttributeProgram(Tx *tx.Transaction) error {
	//TODO: implement CheckAttributeProgram
	return nil
}

func CheckTransactionContracts(Tx *tx.Transaction) error {
	flag, err := VerifySignableData(Tx)
	if flag && err == nil {
		return nil
	} else {
		return err
	}
}

func IsForumUserExist(user string) bool {
	userInfo, _ := ledger.DefaultLedger.Store.GetUserInfo(user)
	if userInfo == nil {
		return false
	}

	return true
}

func CheckForumPostTransaction(hash Uint256) error {
	txn, err := tx.TxStore.GetTransaction(hash)
	if err != nil {
		return errors.New(fmt.Sprintf("The main post hash '%s' is invaild", BytesToHexString(hash.ToArrayReverse())))
	}
	if txn.TxType != tx.PostArticle && txn.TxType != tx.ReplyArticle {
		return errors.New("The transaction type is invalid in main post")
	}

	return nil
}

func CheckForumLikeInfo(hash Uint256, liker string, likeType forum.LikeType) error {
	likeInfo, err := ledger.DefaultLedger.Store.GetLikeInfo(hash)
	if err != nil {
		return errors.New("The like info of main post is invalid")
	}
	flag := false
	for _, v := range likeInfo {
		if v.LikeType != likeType {
			continue
		}
		if liker == v.Liker {
			flag = true
			break
		}
	}
	if flag {
		return errors.New(fmt.Sprintf("The post has been click by user '%s'\n", liker))
	}

	return nil
}

func checkAmountPrecise(amount Fixed64, precision byte) bool {
	return amount.GetData()%int64(math.Pow(10, 8-float64(precision))) != 0
}

func checkIssuerInBookkeeperList(issuer *crypto.PubKey, bookKeepers []*crypto.PubKey) bool {
	for _, bk := range bookKeepers {
		r := crypto.Equal(issuer, bk)
		if r == true {
			log.Debug("issuer is in bookkeeperlist")
			return true
		}
	}
	log.Debug("issuer is NOT in bookkeeperlist")
	return false
}

func CheckTransactionPayload(Tx *tx.Transaction) error {
	switch pld := Tx.Payload.(type) {
	case *payload.BookKeeper:
		//Todo: validate bookKeeper Cert
		_ = pld.Cert
		bookKeepers, _, _ := ledger.DefaultLedger.Store.GetBookKeeperList()
		r := checkIssuerInBookkeeperList(pld.Issuer, bookKeepers)
		if r == false {
			return errors.New("The issuer isn't bookekeeper, can't add other in bookkeepers list.")
		}
		return nil
	case *payload.RegisterAsset:
		if pld.Asset.Precision < asset.MinPrecision || pld.Asset.Precision > asset.MaxPrecision {
			return errors.New("Invalide asset Precision.")
		}
		if checkAmountPrecise(pld.Amount, pld.Asset.Precision) {
			return errors.New("Invalid asset precision.")
		}
	case *payload.IssueAsset:
	case *payload.LockAsset:
		total, locked, err := ledger.DefaultLedger.Store.GetAvailableAsset(pld.ProgramHash, pld.AssetID)
		if err != nil {
			return errors.New("no available asset")
		}
		if locked+pld.Amount > total {
			return errors.New("no enough asset to be locked")
		}
		// TODO: add a height accept range
		if pld.UnlockHeight <= ledger.DefaultLedger.Store.GetHeight() {
			return errors.New("expired LockAsset transaction detected")
		}
	case *payload.TransferAsset:
	case *payload.BookKeeping:
	case *payload.PrivacyPayload:
	case *payload.Record:
	case *payload.DeployCode:
	case *payload.InvokeCode:
	case *payload.DataFile:
	case *payload.RegisterUser:
		username := pld.UserName
		if len(username) < MinUserNameLen || len(username) > MaxUserNameLen {
			return errors.New("invalid username")
		}
		if IsForumUserExist(pld.UserName) {
			return errors.New("user already exists")
		}
	case *payload.PostArticle:
		if !IsForumUserExist(pld.Author) {
			return errors.New("invalid post user")
		}
	case *payload.LikeArticle:
		liker := pld.Liker
		txnHash := pld.PostTxnHash
		liketype := pld.LikeType
		// check if the liker is a valid user registered on blockchain
		if !IsForumUserExist(liker) {
			return errors.New("invalid like user")
		}
		// check if the post user liked is valid
		if err := CheckForumPostTransaction(txnHash); err != nil {
			return err
		}
		// check if the post has been liked by current user
		if err := CheckForumLikeInfo(txnHash, liker, liketype); err != nil {
			return err
		}
	case *payload.ReplyArticle:
		if !IsForumUserExist(pld.Replier) {
			return errors.New("invalid reply user")
		}
		if err := CheckForumPostTransaction(pld.PostHash); err != nil {
			return err
		}
	case *payload.Withdrawal:
		if !IsForumUserExist(pld.Payee) {
			return errors.New("invalid withdraw user")
		}
	default:
		return errors.New("[txValidator],invalidate transaction payload type.")
	}
	return nil
}
