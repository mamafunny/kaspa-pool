package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/pkg/errors"
)

func PutShare(ctx context.Context, address string, bluescore, nonce uint64, diff int) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		_, err := conn.Exec(context.Background(), `INSERT into shares(wallet, bluescore, timestamp, nonce, diff) 
		VALUES ($1, $2, $3, $4, $5)`,
			address, bluescore, time.Now().UTC(), nonce, diff)
		if err != nil {
			return errors.Wrapf(err, "failed to record share for worker %s", address)
		}
		return nil
	})
}

func GetHashrate(pg *pgx.Conn, lookback time.Duration) (model.EffortMap, error) {
	return nil, nil
}

func GetSharesByWallet(ctx context.Context, before time.Time, lookback uint64) (model.EffortMap, error) {
	var effort model.EffortMap
	return effort, DoQuery(ctx, func(conn *pgx.Conn) error {
		res, err := conn.Query(context.Background(),
			`SELECT subq.wallet, SUM(subq.diff) 
			FROM (SELECT wallet, timestamp, diff FROM shares WHERE timestamp <= $1 ORDER BY timestamp LIMIT $2) 
		as subq GROUP BY 1`, before.UTC(), lookback)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch shares from database")
		}
		defer res.Close()
		effort = model.EffortMap{}
		for {
			if !res.Next() {
				break
			}
			wallet := ""
			count := uint64(0)
			if err := res.Scan(&wallet, &count); err != nil {
				return errors.Wrap(err, "failed unmarshalling data")
			}
			effort[model.KaspaWalletAddr(wallet)] = count
		}

		return nil
	})
}
