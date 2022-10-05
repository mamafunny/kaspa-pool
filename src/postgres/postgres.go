package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

func ConfigurePostgres(connString string) (*pgx.Conn, error) {
	pg, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}
	if err := pg.Ping(context.Background()); err != nil {
		return nil, errors.Wrap(err, "failed to ping postgres")
	}

	return pg, nil
}

func DefaultDockerConnection() (*pgx.Conn, error) {
	pg, err := pgx.Connect(context.Background(), "postgres://postgres:postgres@localhost:5432/kaspapool")
	if err != nil {
		return nil, err
	}
	if err := pg.Ping(context.Background()); err != nil {
		return nil, errors.Wrap(err, "failed to ping postgres")
	}

	return pg, nil
}
