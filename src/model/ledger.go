package model

type LedgerEntryStatus string
type KaspaWalletAddr string

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
