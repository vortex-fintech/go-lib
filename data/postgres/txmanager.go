package postgres

import "context"

// TxManager — абстракция управления транзакциями.
// Любой слой выше infrastructure может зависеть от этого интерфейса,
// а не от конкретного *Client.
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	WithTxRO(ctx context.Context, fn func(ctx context.Context) error) error
}
