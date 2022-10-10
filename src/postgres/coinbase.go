package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/pkg/errors"
)

func GetCoinbaseTip(ctx context.Context, wallet model.KaspaWalletAddr) (uint64, error) {
	var daaTip uint64
	return daaTip, DoQuery(ctx, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `SELECT daascore from coinbase_payments cp 
										WHERE cp.wallet = $1 
										ORDER BY daascore DESC LIMIT 1`,
			wallet)
		return row.Scan(&daaTip)
	})
}

func PutCoinbasePayment(ctx context.Context, payment *model.CoinbasePayment) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		_, err := conn.Exec(context.Background(),
			`INSERT into coinbase_payments(tx, tx_idx, wallet, amount, daascore)
				VALUES ($1, $2, $3, $4, $5)`,
			payment.TxId, payment.TxIndex, payment.Wallet, payment.Amount, payment.Daascore)
		if err != nil {
			return errors.Wrapf(err, "failed to insert coinbase payment")
		}
		return nil
	})
}
