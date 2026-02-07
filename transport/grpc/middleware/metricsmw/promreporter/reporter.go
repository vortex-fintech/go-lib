package promreporter

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
)

type RPCMetrics interface {
	ObserveRPC(service, method, code string, seconds float64)
	IncError(typ, service, method string)
}

type Reporter struct {
	M RPCMetrics
}

func (r Reporter) ObserveRPCFull(ctx context.Context, fullMethod string, code codes.Code, secs float64) {
	if r.M == nil {
		return
	}
	svc, mth := SplitGRPCMethod(fullMethod)
	r.M.ObserveRPC(svc, mth, code.String(), secs)
	if code != codes.OK {
		r.M.IncError("grpc", svc, mth)
	}
}

func SplitGRPCMethod(full string) (service, method string) {
	if full == "" {
		return "unknown", "unknown"
	}
	full = strings.TrimPrefix(full, "/")
	parts := strings.Split(full, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1]
	}
	if i := strings.LastIndex(full, "."); i > 0 && i+1 < len(full) {
		return full[:i], full[i+1:]
	}
	return "unknown", full
}
