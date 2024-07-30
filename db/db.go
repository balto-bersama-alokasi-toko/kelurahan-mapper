package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDb(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable", "postgres", "I>6F^,IC9~bB34Y,",
		"34.128.79.15", 5432, "balto_db")

	conn, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Println("Success connect Database")

	return conn, nil
}
