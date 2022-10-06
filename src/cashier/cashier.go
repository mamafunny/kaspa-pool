package cashier

import (
	"context"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/client"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/pb"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
)

type Cashier interface {
	Send(ctx context.Context, address string, amount uint64) error
}

type CashierClient struct {
	config  CashierConfig
	client  pb.KaspawalletdClient
	cleanup func()
}

type CashierConfig struct {
	DaemonAddress  string `yaml:"daemon_address"`
	PoolAddress    string `yaml:"pool_wallet_address"`
	Password       string `yaml:"password"`
	Mock           bool   `yaml:"use_mock"`
	PostgresConfig string `yaml:"postgres"`
}

func NewCashierClient(cfg CashierConfig) (Cashier, error) {
	if cfg.Mock {
		return NewMockCashier(cfg)
	}

	client, deferred, err := client.Connect(cfg.DaemonAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed connecting to kaspawallet daemon")
	}
	postgres.ConfigurePostgres(cfg.PostgresConfig)

	return &CashierClient{
		config:  cfg,
		client:  client,
		cleanup: deferred,
	}, nil
}

func (cc *CashierClient) Send(ctx context.Context, address string, amount uint64) error {
	resp, err := cc.client.Send(ctx, &pb.SendRequest{
		ToAddress: address,
		From: []string{
			cc.config.PoolAddress,
		},
		Amount:                   amount,
		Password:                 cc.config.Password,
		UseExistingChangeAddress: true,
	})
	if err != nil {
		return errors.Wrap(err, "failed sending to address")
	}
	_ = resp
	return nil
}

func DeterminePayouts(payout uint64, effort postgres.EffortMap) postgres.PayoutMap {
	totalEffort := uint64(0)
	for _, v := range effort {
		totalEffort += v
	}

	out := postgres.PayoutMap{}
	for k, v := range effort {
		out[k] = uint64(((float64)(v) / (float64)(totalEffort)) * float64(payout))
	}
	return out
}
