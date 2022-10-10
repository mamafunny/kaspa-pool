package model

type CoinbasePayment struct {
	TxId     string
	TxIndex  uint32
	Wallet   string
	Amount   uint64
	Daascore uint64
}
