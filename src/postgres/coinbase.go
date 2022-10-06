package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

func PutCoinbasePayment(ctx context.Context,
	tx string, wallet string, amount uint64, daascore uint64) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		_, err := conn.Exec(context.Background(), `INSERT into coinbase_payments(tx, wallet, amount, daascore)
			VALUES ($1, $2, $3, $4)`, time.Now().UTC(), tx, wallet, amount, daascore)
		if err != nil {
			return errors.Wrapf(err, "failed to record block to databse")
		}
		return nil
	})
}

type CoinbasePayment struct {
	TxId     string
	Wallet   string
	Amount   uint64
	Daascore uint64
}

func GetCoinbasePayments(ctx context.Context, daascore uint64, window int) (*CoinbasePayment, error) {
	var result *CoinbasePayment
	return result, DoQuery(ctx, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `SELECT tx, wallet, amount, daascore from coinbase_payments cp 
										where cp.daascore > $1 AND cp.daascore <= ($1 + $2 )
										ORDER BY daascore ASC LIMIT 1`,
			daascore, window)

		result = &CoinbasePayment{}
		return row.Scan(&result.TxId, &result.Wallet, &result.Amount, &result.Daascore)
	})
}
