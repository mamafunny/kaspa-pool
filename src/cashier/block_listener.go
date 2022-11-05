package cashier

import (
	"context"
	"fmt"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/onemorebsmith/kaspa-pool/src/kaspaapi"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func BackfillFromUnconfirmed(ctx context.Context, ka *kaspaapi.KaspaApi, logger *zap.Logger) error {
	blocks, err := postgres.GetUnconfirmedBlocks(ctx, 1024)
	if err != nil {
		return err
	}

	for _, b := range blocks {
		logger.Info(fmt.Sprintf("attempting transaction index backfill from block %s", b.Hash))
		err := ka.GetBlocksSince(ctx, b.Hash, func(b *appmessage.RPCBlock) {
			indexBlock(ctx, logger, b)
		})
		if err == nil {
			logger.Info(fmt.Sprintf("backfill from block %s successful", b.Hash))
			return nil // success
		}
	}
	return nil
}

func Backfill(ctx context.Context, ka *kaspaapi.KaspaApi, logger *zap.Logger, lowHash string) error {
	return ka.GetBlocksSince(ctx, lowHash, func(b *appmessage.RPCBlock) {
		indexBlock(ctx, logger, b)
	})
}

func indexBlock(ctx context.Context, logger *zap.Logger, rpcBlock *appmessage.RPCBlock) {
	var transactions []model.RawKaspaTransaction
	for _, t := range rpcBlock.Transactions {
		transactions = append(transactions, t)
	}
	if err := postgres.PutTransactions(ctx, transactions); err != nil {
		logger.Error(err.Error())
	}
}

func StartListener(ctx context.Context, ka *kaspaapi.KaspaApi, logger *zap.Logger) {
	logger = logger.Named("Transaction Listener")
	logger.Info("Starting transaction indexer")
	go func() {
		logger.Info("Starting backfill")
		if err := BackfillFromUnconfirmed(ctx, ka, logger); err != nil {
			logger.Warn(errors.Wrap(err, "failed transaction backfill").Error())
		}
	}()

	err := ka.StartBlockAddedListener(ctx, func(b *appmessage.RPCBlock) {
		indexBlock(ctx, logger, b)
	})
	if err != nil {
		logger.Error(errors.Wrap(err, "failed starting transaction listener").Error())
	}
}
