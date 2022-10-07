package cashier

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"go.uber.org/zap"
)

const coinbase_payment_window = 32

func ResolverThread(ctx context.Context, logger *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping block resolution thread, context cancelled")
			return
		case <-ticker.C:
			blocks, err := ResolveBlocks(ctx, logger)
			if err != nil {
				logger.Error("error resolving blocks", zap.Error(err))
				continue
			}
			CommitResolvedBlocks(ctx, logger, blocks)
			if err != nil {
				logger.Error("error committing resolved blocks", zap.Error(err))
				continue
			}
		}
	}
}

func ResolveBlocks(ctx context.Context, logger *zap.Logger) ([]*model.ConfirmedBlock, error) {
	logger.Info("resolving blocks")
	blocks, err := postgres.GetUnconfirmedBlocks(ctx, 10)
	if err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("fetched %d blocks to resolve", len(blocks)))
	var resolvedBlocks []*model.ConfirmedBlock
	for _, block := range blocks {
		logger.Info(fmt.Sprintf("resolving block %s - daa: %d", block.Hash, block.Header.DAAScore))
		cp, err := ResolveCoinbasePayment(ctx, block.Wallet, block.Header.DAAScore, coinbase_payment_window)
		if err != nil {
			logger.Warn(fmt.Sprintf("failed to resolve block %s - daa: %d", block.Hash, block.Header.DAAScore), zap.Error(err))
			continue
		}
		daaDiff := cp.Daascore - block.Header.DAAScore
		logger.Info(fmt.Sprintf("resolved block %s <-> %s, daa diff: %d", block.Hash, cp.TxId, daaDiff))
		resolvedBlocks = append(resolvedBlocks, &model.ConfirmedBlock{
			UnconfirmedBlock: *block,
			CoinbasePayment:  cp,
		})
	}

	return resolvedBlocks, nil
}

func ResolveCoinbasePayment(ctx context.Context, wallet string, daascore uint64, window int) (*model.CoinbasePayment, error) {
	var result *model.CoinbasePayment
	// To determine the coinbase payment we're going to take the closest wallet utxo for the payee
	// wallet to the unconfirmed block's daacore, within a tight window
	return result, postgres.DoQuery(ctx, func(conn *pgx.Conn) error {

		row := conn.QueryRow(ctx, `SELECT tx, wallet, amount, daascore from coinbase_payments cp 
										WHERE cp.wallet = $1 AND
											  cp.daascore > $2 AND 
											  cp.daascore <= ($2 + $3)
										ORDER BY daascore ASC LIMIT 1`,
			wallet, daascore, window)

		result = &model.CoinbasePayment{}
		return row.Scan(&result.TxId, &result.Wallet, &result.Amount, &result.Daascore)
	})
}

func CommitResolvedBlocks(ctx context.Context, logger *zap.Logger, resolved []*model.ConfirmedBlock) {
	for _, block := range resolved {
		if err := postgres.UpdateBlockCoinbaseTransaction(ctx, block); err != nil {
			logger.Error("failed to update resolved block", zap.Error(err))
		}
	}
}
