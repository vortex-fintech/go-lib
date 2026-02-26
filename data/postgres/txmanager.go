package postgres

import "context"

// TxManager is a minimal transaction management abstraction.
// Higher layers can depend on this interface instead of *Client.
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	WithTxRO(ctx context.Context, fn func(ctx context.Context) error) error
}

// AdvancedTxManager is an extended contract for advanced scenarios
// (retryable serializable, savepoints, explicit tx options).
type AdvancedTxManager interface {
	TxManager
	WithTxOpts(ctx context.Context, cfg TxConfig, fn func(ctx context.Context) error) error
	WithSerializable(ctx context.Context, maxRetries int, fn func(ctx context.Context) error) error
	WithSerializableRO(ctx context.Context, deferrable bool, fn func(ctx context.Context) error) error
	WithSavepoint(ctx context.Context, fn func(ctx context.Context) error) error
}
