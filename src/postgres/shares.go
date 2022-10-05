package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

func PutShare(pg *pgx.Conn, address string, bluescore, nonce uint64, diff int) error {
	_, err := pg.Exec(context.Background(), `INSERT into shares(wallet, bluescore, timestamp, nonce, diff) 
	VALUES ($1, $2, $3, $4, $5)`,
		address, bluescore, time.Now().UTC(), nonce, diff)
	if err != nil {
		return errors.Wrapf(err, "failed to record share for worker %s", address)
	}
	return nil
}

type EffortMap map[string]uint64

func GetSharesByWallet(pg *pgx.Conn, before time.Time, lookback int) (EffortMap, error) {
	res, err := pg.Query(context.Background(),
		`SELECT subq.wallet, SUM(subq.diff) 
			FROM (SELECT wallet, timestamp, diff FROM shares WHERE timestamp <= $1 ORDER BY timestamp LIMIT $2) 
		as subq GROUP BY 1`, before.UTC(), lookback)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch shares from database")
	}
	effort := EffortMap{}
	defer res.Close()
	for {
		if !res.Next() {
			break
		}
		wallet := ""
		count := uint64(0)
		if err := res.Scan(&wallet, &count); err != nil {
			return nil, errors.Wrap(err, "failed unmarshalling data")
		}
		effort[wallet] = count
	}

	return effort, nil
}
