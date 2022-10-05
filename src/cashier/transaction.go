package cashier

type Transaction struct {
	To     string
	From   string
	Amount uint64
	TxId   string
}
