package transaction

import (

	"UNetwork/common"
	"UNetwork/core/asset"
	"UNetwork/core/code"
	"UNetwork/core/contract/program"
	"UNetwork/core/forum"
	"UNetwork/core/transaction/payload"
	"UNetwork/crypto"
	"UNetwork/smartcontract/types"
	"math/rand"
	"strconv"

)

//initial a new transaction with asset registration payload
func NewRegisterAssetTransaction(asset *asset.Asset, amount common.Fixed64, issuer *crypto.PubKey, conroller common.Uint160) (*Transaction, error) {
	assetRegPayload := &payload.RegisterAsset{
		Asset:      asset,
		Amount:     amount,
		Issuer:     issuer,
		Controller: conroller,
	}

	return &Transaction{
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		TxType:   RegisterAsset,
		Payload:  assetRegPayload,
		Programs: []*program.Program{},
	}, nil
}

//initial a new transaction with asset registration payload
func NewBookKeeperTransaction(pubKey *crypto.PubKey, isAdd bool, cert []byte, issuer *crypto.PubKey) (*Transaction, error) {
	bookKeeperPayload := &payload.BookKeeper{
		PubKey: pubKey,
		Action: payload.BookKeeperAction_SUB,
		Cert:   cert,
		Issuer: issuer,
	}

	if isAdd {
		bookKeeperPayload.Action = payload.BookKeeperAction_ADD
	}

	return &Transaction{
		TxType:        BookKeeper,
		Payload:       bookKeeperPayload,
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		Programs: []*program.Program{},
	}, nil
}

func NewIssueAssetTransaction(outputs []*TxOutput) (*Transaction, error) {
	assetRegPayload := &payload.IssueAsset{}

	return &Transaction{
		TxType:  IssueAsset,
		Payload: assetRegPayload,
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		BalanceInputs: []*BalanceTxInput{},
		Outputs:       outputs,
		Programs:      []*program.Program{},
	}, nil
}

func NewTransferAssetTransaction(inputs []*UTXOTxInput, outputs []*TxOutput) (*Transaction, error) {
	assetRegPayload := &payload.TransferAsset{}

	return &Transaction{
		TxType:  TransferAsset,
		Payload: assetRegPayload,
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		UTXOInputs:    inputs,
		BalanceInputs: []*BalanceTxInput{},
		Outputs:       outputs,
		Programs:      []*program.Program{},
	}, nil
}

//initial a new transaction with record payload
func NewRecordTransaction(recordType string, recordData []byte) (*Transaction, error) {
	recordPayload := &payload.Record{
		RecordType: recordType,
		RecordData: recordData,
	}

	return &Transaction{
		TxType:  Record,
		Payload: recordPayload,
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Programs:      []*program.Program{},
	}, nil
}

func NewPrivacyPayloadTransaction(fromPrivKey []byte, fromPubkey *crypto.PubKey, toPubkey *crypto.PubKey, payloadType payload.EncryptedPayloadType, data []byte) (*Transaction, error) {
	privacyPayload := &payload.PrivacyPayload{
		PayloadType: payloadType,
		EncryptType: payload.ECDH_AES256,
		EncryptAttr: &payload.EcdhAes256{
			FromPubkey: fromPubkey,
			ToPubkey:   toPubkey,
		},
	}
	privacyPayload.Payload, _ = privacyPayload.EncryptAttr.Encrypt(data, fromPrivKey)

	return &Transaction{
		TxType:  PrivacyPayload,
		Payload: privacyPayload,
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Programs:      []*program.Program{},
	}, nil
}
func NewDataFileTransaction(path string, fileName string, note string, issuer *crypto.PubKey) (*Transaction, error) {
	DataFilePayload := &payload.DataFile{
		IPFSPath: path,
		Filename: fileName,
		Note:     note,
		Issuer:   issuer,
	}

	return &Transaction{
		TxType:  DataFile,
		Payload: DataFilePayload,
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Programs:      []*program.Program{},
	}, nil
}

func NewLockAssetTransaction(programHash common.Uint160, assetID common.Uint256, amount common.Fixed64, height uint32) (*Transaction, error) {
	lockAssetPayload := &payload.LockAsset{
		ProgramHash:  programHash,
		AssetID:      assetID,
		Amount:       amount,
		UnlockHeight: height,
	}

	return &Transaction{
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		TxType:   LockAsset,
		Payload:  lockAssetPayload,
		Programs: []*program.Program{},
	}, nil
}

//initial a new transaction with publish payload
func NewDeployTransaction(fc *code.FunctionCode, programHash common.Uint160, name, codeversion, author, email, desp string, language types.LangType) (*Transaction, error) {
	DeployCodePayload := &payload.DeployCode{
		Code:        fc,
		Name:        name,
		CodeVersion: codeversion,
		Author:      author,
		Email:       email,
		Description: desp,
		Language:    language,
		ProgramHash: programHash,
	}

	return &Transaction{
		TxType:  DeployCode,
		Payload: DeployCodePayload,
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Programs:      []*program.Program{},
	}, nil
}

//initial a new transaction with invoke payload
func NewInvokeTransaction(fc []byte, codeHash common.Uint160, programhash common.Uint160) (*Transaction, error) {
	InvokeCodePayload := &payload.InvokeCode{
		Code:        fc,
		CodeHash:    codeHash,
		ProgramHash: programhash,
	}

	return &Transaction{
		TxType:  InvokeCode,
		Payload: InvokeCodePayload,
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Programs:      []*program.Program{},
	}, nil
}

func NewRegisterUserTrasaction(username string, userhash common.Uint160) (*Transaction, error) {
	registerUserPayload := &payload.RegisterUser{
		UserName:        username,
		UserProgramHash: userhash,
		Reputation:      100 * 100000000,
	}

	return &Transaction{
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		TxType:   RegisterUser,
		Payload:  registerUserPayload,
		Programs: []*program.Program{},
	}, nil
}

func NewPostArticleTrasaction(articleHash common.Uint256, author string) (*Transaction, error) {
	postArticlePayload := &payload.PostArticle{
		ContentHash: articleHash,
		Author:      author,
	}

	return &Transaction{
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		TxType:   PostArticle,
		Payload:  postArticlePayload,
		Programs: []*program.Program{},
	}, nil
}

func NewReplyArticleTrasaction(postHash common.Uint256, contentHash common.Uint256, replier string) (*Transaction, error) {
	replyArticlePayload := &payload.ReplyArticle{
		PostHash:    postHash,
		ContentHash: contentHash,
		Replier:     replier,
	}

	return &Transaction{
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		TxType:   ReplyArticle,
		Payload:  replyArticlePayload,
		Programs: []*program.Program{},
	}, nil
}

func NewLikeArticleTrasaction(articleHash common.Uint256, me string, likeType forum.LikeType) (*Transaction, error) {
	LikeArticlePayload := &payload.LikeArticle{
		PostTxnHash: articleHash,
		Liker:       me,
		LikeType:    likeType,
	}

	return &Transaction{
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		TxType:   LikeArticle,
		Payload:  LikeArticlePayload,
		Programs: []*program.Program{},
	}, nil
}

func NewWithdrawalTrasaction(payee string, recipient common.Uint160, asset common.Uint256, amount common.Fixed64) (*Transaction, error) {
	WithdrawalPayload := &payload.Withdrawal{
		Payee: payee,
	}

	return &Transaction{
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Outputs: []*TxOutput{
			{
				AssetID:     asset,
				Value:       amount,
				ProgramHash: recipient,
			},
		},
		Attributes: []*TxAttribute{
			{
				Usage: Nonce,
				Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
			},
		},
		TxType:   Withdrawal,
		Payload:  WithdrawalPayload,
		Programs: []*program.Program{},
	}, nil
}
