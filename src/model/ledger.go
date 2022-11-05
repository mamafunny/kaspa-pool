package model

import (
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/pb"
)

type LedgerEntryStatus string
type KaspaWalletAddr string

const KasDigitMultipler = 100000000 // multiplier from kas to non-decimal used in the db

const ( // needs to match `ledger_entry_status` in pg
	LedgerEntryStatusOwed      LedgerEntryStatus = "owed"
	LedgerEntryStatusSubmitted LedgerEntryStatus = "submitted"
	LedgerEntryStatusConfirmed LedgerEntryStatus = "confirmed"
	LedgerEntryStatusError     LedgerEntryStatus = "error"
)

type LedgerEntry struct {
	Status   LedgerEntryStatus
	Wallet   KaspaWalletAddr
	Amount   uint64
	Daascore uint64
	TxId     *string
}
type EffortMap map[KaspaWalletAddr]uint64

type AggregatedLedgerEntry struct {
	Wallet KaspaWalletAddr
	Amount uint64
}

type MinerPayoutTransction struct {
	TxId   string
	Amount uint64
	Wallet KaspaWalletAddr
	Raw    *pb.SendResponse
}

type RawKaspaTransaction *appmessage.RPCTransaction
