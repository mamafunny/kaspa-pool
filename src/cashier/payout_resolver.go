package cashier

import (
	"context"

	"github.com/onemorebsmith/kaspa-pool/src/kaspaapi"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func DoPayouts(ctx context.Context, cfg CashierConfig, logger *zap.Logger) error {
	entries, err := postgres.GetLedgerEntriesByStatus(ctx, model.LedgerEntryStatusOwed, 1024)
	if err != nil {
		return errors.Wrap(err, "failed fetching pending payouts from ledger")
	}
	kaapi := kaspaapi.NewKaspawalletApi(cfg.RPCServer, logger)
	for _, entry := range entries {
		kaapi.SendKas(ctx, entry.Wallet, model.KaspaWalletAddr(cfg.PoolWallet), entry.Amount, cfg.Password)
	}
	return nil
}
