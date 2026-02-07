package errors

import (
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/durationpb"
)

// GRPCRateLimited — готовый gRPC error с RetryInfo + ErrorInfo(reason).
func GRPCRateLimited(retryAfter time.Duration) error {
	st := status.New(codes.ResourceExhausted, "Rate limited")

	ri := &errdetails.RetryInfo{RetryDelay: durationpb.New(retryAfter)}
	ei := &errdetails.ErrorInfo{
		Reason:   "rate_limited",
		Metadata: map[string]string{"retry_after_ms": fmt.Sprintf("%d", retryAfter.Milliseconds())},
	}

	st2, err := st.WithDetails(ri, ei)
	if err != nil {
		return st.Err()
	}
	return st2.Err()
}
