package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type PayoutMap map[string]uint64

func PutPayable(ctx context.Context, balances PayoutMap, bluescore uint64) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		rows := [][]any{}
		now := time.Now().UTC()
		for k, v := range balances {
			// payee, amount, updated, bluescore,
			rows = append(rows, []any{
				k, v, now, bluescore,
			})
		}

		_, err := conn.CopyFrom(context.Background(), pgx.Identifier{"ledger"},
			[]string{"payee", "amount", "updated", "bluescore"}, pgx.CopyFromRows(rows))
		if err != nil {
			return errors.Wrap(err, "failed to write to ledger")
		}
		return nil
	})
}
