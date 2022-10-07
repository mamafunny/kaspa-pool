package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/pkg/errors"
)

func PutCoinbasePayment(ctx context.Context, payment *model.CoinbasePayment) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		_, err := conn.Exec(context.Background(), `INSERT into coinbase_payments(tx, wallet, amount, daascore)
			VALUES ($1, $2, $3, $4)`, time.Now().UTC(),
			payment.TxId, payment.Wallet, payment.Amount, payment.Daascore)
		if err != nil {
			return errors.Wrapf(err, "failed to record block to databse")
		}
		return nil
	})
}
