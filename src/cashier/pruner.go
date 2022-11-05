package cashier

import (
	"context"
	"fmt"
	"time"

	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func StartPruner(ctx context.Context, delay time.Duration, logger *zap.Logger) error {
	ticker := time.NewTicker(delay)
	logger = logger.Named("pruner")
	for {
		select {
		case <-ticker.C:
			if err := PruneShares(ctx, 1000000); err != nil {
				logger.Error(err.Error())
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func PruneShares(ctx context.Context, keep uint64) error {
	pruner := fmt.Sprintf(`DELETE FROM SHARES WHERE bluescore < 
			(SELECT MIN(bluescore) FROM 
				(select s.bluescore FROM shares s ORDER BY bluescore DESC LIMIT %d) as raw)`, keep)
	return errors.Wrap(postgres.DoExec(ctx, pruner), "failed pruning shares")
}
