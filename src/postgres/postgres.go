package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

var connectionString string

func ConfigurePostgres(connString string) {
	connectionString = connString
}

func GetConnection(ctx context.Context) (*pgx.Conn, error) {
	pg, err := pgx.Connect(ctx, connectionString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create connection to pg")
	}
	return pg, nil
}

func ConfigureDockerConnection() {
	ConfigurePostgres("postgres://postgres:postgres@localhost:5432/kaspapool")
}

func DoQuery(ctx context.Context, handler func(conn *pgx.Conn) error) error {
	conn, err := GetConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	return handler(conn)
}

func DoExec(ctx context.Context, command string) error {
	conn, err := GetConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(context.Background(), command)
	return err
}

func DoExecOrDie(ctx context.Context, command string) {
	conn, err := GetConnection(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(context.Background(), command)
	if err != nil {
		panic(err)
	}
}
