//go:build integration
// +build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/db/postgres"
)

func TestNewPostgresClient_Integration(t *testing.T) {
	cfg := postgres.DBConfig{
		Host:            "localhost",
		Port:            "5433",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
	}

	db, err := postgres.NewPostgresClient(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, db)
}
