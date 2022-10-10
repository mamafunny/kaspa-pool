package cashier

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/onemorebsmith/kaspa-pool/src/kaspaapi"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func UTXOThread(ctx context.Context, logger *zap.Logger, cfg CashierConfig) {
	ticker := time.NewTicker(30 * time.Second)
	logger = logger.With(zap.String("component", "UTXOThread"))
	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping block resolution thread, context cancelled")
			return
		case <-ticker.C:
			if err := DoUTXOResolve(ctx, cfg, logger); err != nil {
				logger.Error(err.Error())
				continue
			}
		}
	}
}

func DoUTXOResolve(ctx context.Context, cfg CashierConfig, logger *zap.Logger) error {
	kaapi, err := kaspaapi.NewKaspaAPI(cfg.RPCServer, logger)
	if err != nil {
		return errors.Wrap(err, "error connecting to kaspad -- utxo update skipped")
	}
	defer kaapi.Close()
	if err := UpdateWalletUTXOLedger(context.Background(), logger, kaapi, model.KaspaWalletAddr(cfg.PoolWallet)); err != nil {
		return errors.Wrap(err, "failed updating utxo ledger")
	}
	return nil
}

func UpdateWalletUTXOLedger(ctx context.Context, logger *zap.Logger, kapi *kaspaapi.KaspaApi, wallet model.KaspaWalletAddr) error {
	coinbaseTip, err := postgres.GetCoinbaseTip(ctx, wallet)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Info("no utxo entries in ledger for wallet, doing full fetch", zap.String("wallet", string(wallet)))
			coinbaseTip = 0
		} else {
			return errors.Wrap(err, "failed fetching utxo tip for wallet")
		}
	}
	logger.Info(fmt.Sprintf("fetching utxos > daa %d", coinbaseTip))
	utxos, err := kapi.GetCoinbaseUTXOsForWallet(wallet, coinbaseTip)
	if err != nil {
		return err
	}

	for _, v := range utxos {
		logger.Info("adding coinbase utxo", zap.Any("utxo", v))
		if err := postgres.PutCoinbasePayment(ctx, v); err != nil {
			logger.Error("failed writing coinbase payment", zap.Error(err), zap.Any("payment", v))
		}
	}

	return nil
}
