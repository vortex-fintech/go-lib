//go:build unit
// +build unit

package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/db/postgres"
)

func TestNewPostgresClient_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectPing()

	cfg := postgres.DBConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "user",
		Password:        "pass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 10 * time.Minute,
	}

	original := postgres.OpenSQL
	postgres.OpenSQL = func(driverName, dataSourceName string) (*sql.DB, error) {
		return db, nil
	}
	defer func() { postgres.OpenSQL = original }()

	conn, err := postgres.NewPostgresClient(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.NoError(t, mock.ExpectationsWereMet())
}
