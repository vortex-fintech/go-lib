package postgres

import (
	"errors"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		err  error
	}{
		{name: "empty url", cfg: Config{}, err: errEmptyURL},
		{name: "negative max", cfg: Config{URL: "postgres://u:p@h:5432/db", MaxConns: -1}, err: errNegativeMaxConns},
		{name: "negative min", cfg: Config{URL: "postgres://u:p@h:5432/db", MinConns: -1}, err: errNegativeMinConns},
		{name: "min exceeds max", cfg: Config{URL: "postgres://u:p@h:5432/db", MaxConns: 2, MinConns: 3}, err: errMinConnsExceedsMaxConns},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.validate()
			if !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
}

func TestConfigValidate_OK(t *testing.T) {
	t.Parallel()

	cfg := Config{URL: "postgres://u:p@h:5432/db?sslmode=disable", MaxConns: 8, MinConns: 2}
	if err := cfg.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDBConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  DBConfig
		err  error
	}{
		{name: "missing host", cfg: DBConfig{Port: "5432", DBName: "db"}, err: errHostRequired},
		{name: "missing port", cfg: DBConfig{Host: "localhost", DBName: "db"}, err: errPortRequired},
		{name: "missing db", cfg: DBConfig{Host: "localhost", Port: "5432"}, err: errDBNameRequired},
		{name: "negative open", cfg: DBConfig{Host: "localhost", Port: "5432", DBName: "db", MaxOpenConns: -1}, err: errNegativeMaxConns},
		{name: "negative idle", cfg: DBConfig{Host: "localhost", Port: "5432", DBName: "db", MaxIdleConns: -1}, err: errNegativeMinConns},
		{name: "idle exceeds open", cfg: DBConfig{Host: "localhost", Port: "5432", DBName: "db", MaxOpenConns: 2, MaxIdleConns: 3}, err: errMinConnsExceedsMaxConns},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.validate()
			if !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
}

func TestDBConfigValidate_OK(t *testing.T) {
	t.Parallel()

	cfg := DBConfig{Host: "localhost", Port: "5432", DBName: "db", MaxOpenConns: 8, MaxIdleConns: 2}
	if err := cfg.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
