//go:build unit
// +build unit

package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/db/postgres"
)

func TestNewPostgresClient_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
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
		ConnMaxIdleTime: 2 * time.Minute,
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

func TestNewPostgresClient_BuildsDSN_AndPings(t *testing.T) {
	origOpen := postgres.OpenSQL
	t.Cleanup(func() { postgres.OpenSQL = origOpen })

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var capturedDriver, capturedDSN string
	postgres.OpenSQL = func(driverName, dsn string) (*sql.DB, error) {
		capturedDriver = driverName
		capturedDSN = dsn
		return db, nil
	}

	cfg := postgres.DBConfig{
		Host:            "h",
		Port:            "5432",
		User:            "u",
		Password:        "p",
		DBName:          "d",
		SSLMode:         "disable",
		MaxOpenConns:    7,
		MaxIdleConns:    3,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: 30 * time.Second,
	}

	mock.ExpectPing()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	gotDB, err := postgres.NewPostgresClient(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, gotDB)

	require.Equal(t, "postgres", capturedDriver)
	for _, part := range []string{
		"host=h",
		"port=5432",
		"user=u",
		"password=p",
		"dbname=d",
		"sslmode=disable",
	} {
		require.True(t, strings.Contains(capturedDSN, part), "dsn missing part: %s; got=%s", part, capturedDSN)
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewPostgresClient_OpenError(t *testing.T) {
	origOpen := postgres.OpenSQL
	t.Cleanup(func() { postgres.OpenSQL = origOpen })

	postgres.OpenSQL = func(driverName, dsn string) (*sql.DB, error) {
		return nil, errors.New("open failed")
	}

	cfg := postgres.DBConfig{
		Host:            "h",
		Port:            "5432",
		User:            "u",
		Password:        "p",
		DBName:          "d",
		SSLMode:         "disable",
		MaxOpenConns:    1,
		MaxIdleConns:    0,
		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 0,
	}

	_, err := postgres.NewPostgresClient(context.Background(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "open failed")
}

func TestNewPostgresClient_PingError(t *testing.T) {
	origOpen := postgres.OpenSQL
	t.Cleanup(func() { postgres.OpenSQL = origOpen })

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	postgres.OpenSQL = func(driverName, dsn string) (*sql.DB, error) {
		return db, nil
	}

	mock.ExpectPing().WillReturnError(errors.New("ping failed"))

	cfg := postgres.DBConfig{
		Host:            "h",
		Port:            "5432",
		User:            "u",
		Password:        "p",
		DBName:          "d",
		SSLMode:         "disable",
		MaxOpenConns:    1,
		MaxIdleConns:    0,
		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 0,
	}

	_, err = postgres.NewPostgresClient(context.Background(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ping failed")
	require.NoError(t, mock.ExpectationsWereMet())
}
