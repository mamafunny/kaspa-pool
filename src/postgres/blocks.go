package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

func InsertBlock(pg *pgx.Conn, address string, bluescore, nonce uint64) error {
	_, err := pg.Exec(context.Background(), `INSERT into blocks(wallet, bluescore, nonce, timestamp) 
	VALUES ($1, $2, $3, $4)`,
		address, bluescore, nonce, time.Now())
	if err != nil {
		return errors.Wrapf(err, "failed to record share for worker %s", address)
	}
	return nil
}
