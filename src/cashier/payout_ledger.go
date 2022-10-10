package cashier

import (
	"context"
	"fmt"
	"time"

	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func DoLedgerUpdate(ctx context.Context, cfg CashierConfig, logger *zap.Logger) error {
	confirmedBlocks, err := postgres.GetConfirmedBlocks(ctx, 100)
	if err != nil {
		return errors.Wrap(err, "failed fetching confirmed blocks for ledger update")
	}
	logger.Info(fmt.Sprintf("fetched %d confirmed blocks for processing", len(confirmedBlocks)))
	for _, block := range confirmedBlocks {
		if err := GenerateLedgerEntriesForBlock(ctx, block, cfg.PPLNSWindow); err != nil {
			logger.Error("failed generating ledger entries for block", zap.Any("block", confirmedBlocks), zap.Error(err))
		}
	}

	return nil
}

func GenerateLedgerEntriesForBlock(ctx context.Context, block *model.ConfirmedBlock, shareWindow uint64) error {
	shares, err := postgres.GetSharesByWallet(ctx, time.Unix(block.Header.Timestamp, 0), shareWindow)
	if err != nil {
		return errors.Wrap(err, "failed getting shares for block effort")
	}

	ledgerEntries := DeterminePayouts(block.CoinbasePayment.Amount, block.Header.DAAScore, shares)
	if err := postgres.PutPayable(ctx, block.Hash, ledgerEntries); err != nil {
		return errors.Wrap(err, "failed to put payble entries to ledger")
	}

	return nil
}

func DeterminePayouts(payout uint64, daascore uint64, effort model.EffortMap) []model.LedgerEntry {
	payoutFloat := float64(payout)
	totalEffort := float64(0)
	for _, v := range effort {
		totalEffort += float64(v)
	}

	var entries []model.LedgerEntry
	for k, v := range effort {
		entries = append(entries, model.LedgerEntry{
			Wallet:   k,
			Status:   model.LedgerEntryStatusOwed,
			Amount:   uint64(((float64)(v) / totalEffort) * payoutFloat),
			Daascore: daascore,
			TxId:     nil,
		})
	}
	return entries
}
