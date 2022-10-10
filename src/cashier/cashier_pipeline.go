package cashier

import (
	"context"
	"time"

	"github.com/onemorebsmith/kaspa-pool/src/model"
	"go.uber.org/zap"
)

func StartPipeline(ctx context.Context, cfg CashierConfig, logger *zap.Logger) {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			DoPipelineOnce(ctx, cfg, logger)
		}
	}
}

func DoPipelineOnce(ctx context.Context, cfg CashierConfig, logger *zap.Logger) {
	if err := DoBlockResolve(ctx, model.KaspaWalletAddr(cfg.PoolWallet), logger); err != nil {
		logger.Error("error committing resolved blocks", zap.Error(err))
	}
	if err := DoUTXOResolve(ctx, cfg, logger); err != nil {
		logger.Error("error performing utxo resolve for pending blocks", zap.Error(err))
	}
	if err := DoLedgerUpdate(ctx, cfg, logger); err != nil {
		logger.Error("error performing utxo resolve for pending blocks", zap.Error(err))
	}
}
