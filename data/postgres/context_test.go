package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type contextRunnerStub struct{}

func (contextRunnerStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (contextRunnerStub) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

func (contextRunnerStub) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

func TestContextWithRunner_NilContext(t *testing.T) {
	ctx := ContextWithRunner(nil, contextRunnerStub{})
	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}

	r := MustRunnerFromContext(ctx)
	if _, ok := r.(contextRunnerStub); !ok {
		t.Fatalf("expected contextRunnerStub, got %T", r)
	}
}

func TestRunnerFromContext_NilInputs(t *testing.T) {
	if r := RunnerFromContext(nil, nil); r != nil {
		t.Fatalf("expected nil runner, got %T", r)
	}

	if r := RunnerFromContext(context.Background(), nil); r != nil {
		t.Fatalf("expected nil runner, got %T", r)
	}
}

func TestRunnerFromContext_UsesStoredRunner(t *testing.T) {
	want := contextRunnerStub{}
	ctx := ContextWithRunner(context.Background(), want)

	r := RunnerFromContext(ctx, nil)
	if _, ok := r.(contextRunnerStub); !ok {
		t.Fatalf("expected contextRunnerStub, got %T", r)
	}
}

func TestRunnerFromContext_UsesFallback(t *testing.T) {
	r := RunnerFromContext(context.Background(), &Client{})
	if r == nil {
		t.Fatalf("expected fallback runner")
	}
	if _, ok := r.(poolRunner); !ok {
		t.Fatalf("expected poolRunner, got %T", r)
	}
}
