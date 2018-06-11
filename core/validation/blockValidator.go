package validation

import (
	"UNetwork/core/ledger"
	tx "UNetwork/core/transaction"
	. "UNetwork/errors"
	"fmt"
)

func VerifyBlock(block *ledger.Block, ld *ledger.Ledger, completely bool) error {
	if block.Blockdata.Height == 0 {
		return nil
	}
	err := VerifyBlockData(block.Blockdata, ld)
	if err != nil {
		return err
	}

	flag, err := VerifySignableData(block)
	if flag == false || err != nil {
		return err
	}

	if block.Transactions == nil {
		return NewErr(fmt.Sprintf("No Transactions Exist in Block."))
	}
	if block.Transactions[0].TxType != tx.BookKeeping {
		return NewErr(fmt.Sprintf("Blockdata Verify failed first Transacion in block is not BookKeeping type."))
	}
	for index, v := range block.Transactions {
		if v.TxType == tx.BookKeeping && index != 0 {
			return NewErr(fmt.Sprintf("This Block Has BookKeeping transaction after first transaction in block."))
		}
	}

	//verfiy block's transactions
	if completely {
		/*
			//TODO: NextBookKeeper Check.
			bookKeeperaddress, err := ledger.GetBookKeeperAddress(ld.Blockchain.GetBookKeepersByTXs(block.Transactions))
			if err != nil {
				return NewErr(fmt.Sprintf("GetBookKeeperAddress Failed."))
			}
			if block.Blockdata.NextBookKeeper != bookKeeperaddress {
				return NewErr(fmt.Sprintf("BookKeeper is not validate."))
			}
		*/
		for _, txVerify := range block.Transactions {
			if errCode := VerifyTransaction(txVerify); errCode != ErrNoError {
				return NewErr(fmt.Sprintf("VerifyTransaction failed when verifiy block"))
			}
			if errCode := VerifyTransactionWithLedger(txVerify, ledger.DefaultLedger); errCode != ErrNoError {
				return NewErr(fmt.Sprintf("VerifyTransactionWithLedger failed when verifiy block"))
			}
		}
		if err := VerifyTransactionWithBlock(block.Transactions); err != nil {
			return NewErr(fmt.Sprintf("VerifyTransactionWithBlock failed when verifiy block"))
		}
	}

	return nil
}

func VerifyHeader(bd *ledger.Header, ledger *ledger.Ledger) error {
	return VerifyBlockData(bd.Blockdata, ledger)
}

func VerifyBlockData(bd *ledger.Blockdata, ledger *ledger.Ledger) error {
	if bd.Height == 0 {
		return nil
	}

	prevHeader, err := ledger.Blockchain.GetHeader(bd.PrevBlockHash)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[BlockValidator], Cannnot find prevHeader..")
	}
	if prevHeader == nil {
		return NewDetailErr(NewErr("[BlockValidator] error"), ErrNoCode, "[BlockValidator], Cannnot find previous block.")
	}

	if prevHeader.Blockdata.Height+1 != bd.Height {
		return NewDetailErr(NewErr("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block height is incorrect.")
	}

	if prevHeader.Blockdata.Timestamp >= bd.Timestamp {
		return NewDetailErr(NewErr("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block timestamp is incorrect.")
	}

	return nil
}
