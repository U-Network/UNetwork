package account

import (
	"UGCNetwork/core/transaction"
	"io"
)

type CoinState byte

//const (
//	Unconfirmed     CoinState = 0
//	Unspent         CoinState = 1
//	Spending        CoinState = 2
//	Spent           CoinState = 3
//	SpentAndClaimed CoinState = 4
//)

type Coin struct {
	//input  *transaction.UTXOTxInput
	Output *transaction.TxOutput
	//CoinState
}

func (coin *Coin) Serialize(w io.Writer) error {
	//coin.input.Serialize(w)
	coin.Output.Serialize(w)
	//w.Write([]byte{byte(coin.CoinState)})
	return nil
}

func (coin *Coin) Deserialize(r io.Reader) error {
	//coin.input = new(transaction.UTXOTxInput)
	//if err := coin.input.Deserialize(r); err != nil {
	//	return err
	//}

	coin.Output = new(transaction.TxOutput)
	coin.Output.Deserialize(r)
	//
	//state, err := serialization.ReadUint8(r)
	//if err != nil {
	//	return err
	//}
	//coin.CoinState = CoinState(state)

	return nil
}
