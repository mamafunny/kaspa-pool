package model

import "github.com/kaspanet/kaspad/app/appmessage"

type UnconfirmedBlock struct {
	appmessage.RPCBlock
	Hash   string
	Wallet string
}

type BlockStatusType string

const (
	BlockStatusUnconfirmed BlockStatusType = "unconfirmed"
	BlockStatusConfirmed   BlockStatusType = "confirmed"
	BlockStatusPaid        BlockStatusType = "paid"
	BlockStatusError       BlockStatusType = "error"
)

type CoinbasePayment struct {
	TxId     string
	Wallet   string
	Amount   uint64
	Daascore uint64
}

// ConfirmedBlock - a mined block and it's coinbase payment transaction
type ConfirmedBlock struct {
	UnconfirmedBlock
	CoinbasePayment *CoinbasePayment
}

func BlockArrayToMap(arr []*UnconfirmedBlock) map[string]*UnconfirmedBlock {
	mapped := map[string]*UnconfirmedBlock{}
	for _, v := range arr {
		mapped[v.Hash] = v
	}
	return mapped
}

func ConfirmedBlockArrayToMap(arr []*ConfirmedBlock) map[string]*ConfirmedBlock {
	mapped := map[string]*ConfirmedBlock{}
	for _, v := range arr {
		mapped[v.Hash] = v
	}
	return mapped
}
