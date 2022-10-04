package poolworker

import (
	"context"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

func configurePostgres(cfg PostgresConfig) (*pgx.Conn, error) {
	pg, err := pgx.Connect(pgx.ConnConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		Database: cfg.Database,
	})
	if err != nil {
		return nil, err
	}
	if err := pg.Ping(context.Background()); err != nil {
		return nil, errors.Wrap(err, "failed to ping postgres")
	}

	return pg, nil
}
