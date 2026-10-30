package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func openDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return nil, fmt.Errorf("parse PostgreSQL connection settings: %w", err)
		}

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = pool.Ping(pingCtx)
		cancel()
		if err != nil {
			pool.Close()
			lastErr = err
			if isPermanentConnectionError(err) {
				return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
			}
			if attempt == 9 {
				break
			}
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("connect to PostgreSQL (last attempt: %v): %w", lastErr, ctx.Err())
			case <-time.After(time.Duration(attempt+1) * time.Second):
			}
			continue
		}

		schema, err := os.ReadFile("schema.sql")
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("read database schema: %w", err)
		}
		if _, err := pool.Exec(ctx, string(schema)); err != nil {
			pool.Close()
			return nil, fmt.Errorf("initialize database schema: %w", err)
		}
		return pool, nil
	}
	return nil, fmt.Errorf("initialize PostgreSQL: %w", lastErr)
}

func isPermanentConnectionError(err error) bool {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return false
	}
	switch postgresError.Code {
	case "28000", "28P01", "3D000":
		return true
	default:
		return false
	}
}
