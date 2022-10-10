package cashier

import "context"

type MockCashierClient struct {
	config CashierConfig

	transactions []*Transaction
}

func NewMockCashier(cfg CashierConfig) (Cashier, error) {
	return &MockCashierClient{
		config: cfg,
	}, nil
}

func (cc *MockCashierClient) Send(ctx context.Context, address string, amount uint64) error {
	cc.transactions = append(cc.transactions, &Transaction{
		To:     address,
		From:   string(cc.config.PoolWallet),
		Amount: amount,
	})

	return nil
}
