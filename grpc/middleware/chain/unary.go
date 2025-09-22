package chain

import (
	cb "github.com/vortex-fintech/go-lib/grpc/middleware/circuitbreaker"
	ctxcancel "github.com/vortex-fintech/go-lib/grpc/middleware/contextcancel"
	errorsmw "github.com/vortex-fintech/go-lib/grpc/middleware/errorsmw"
	"google.golang.org/grpc"
)

// Options задаёт порядок и состав цепочки unary-интерсепторов.
// Итоговая последовательность вызовов:
//
//	Pre → (ctxcancel) → (circuitbreaker) → (errors) → Post
type Options struct {
	// Пользовательские перехватчики, исполняются раньше/позже встроенных.
	Pre  []grpc.UnaryServerInterceptor
	Post []grpc.UnaryServerInterceptor

	// Встроенные
	CircuitBreaker   *cb.Interceptor // если nil — не включаем
	DisableCtxCancel bool            // по умолчанию false => включено
	DisableErrors    bool            // по умолчанию false => включено
}

// Default возвращает grpc.ServerOption с собранной цепочкой перехватчиков.
func Default(opts Options) grpc.ServerOption {
	var chain []grpc.UnaryServerInterceptor

	// Pre (например, метрики) — первыми
	if len(opts.Pre) > 0 {
		chain = append(chain, opts.Pre...)
	}

	if !opts.DisableCtxCancel {
		chain = append(chain, ctxcancel.Unary())
	}
	if opts.CircuitBreaker != nil {
		chain = append(chain, opts.CircuitBreaker.Unary())
	}
	if !opts.DisableErrors {
		chain = append(chain, errorsmw.Unary())
	}

	// Post — последними
	if len(opts.Post) > 0 {
		chain = append(chain, opts.Post...)
	}

	return grpc.ChainUnaryInterceptor(chain...)
}
