package errors

import (
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/durationpb"
)

// GRPCRateLimited returns a ready gRPC error with RetryInfo + ErrorInfo(reason).
func GRPCRateLimited(retryAfter time.Duration) error {
	if retryAfter < 0 {
		retryAfter = 0
	}

	st := status.New(codes.ResourceExhausted, "Rate limited")

	ri := &errdetails.RetryInfo{RetryDelay: durationpb.New(retryAfter)}
	ei := &errdetails.ErrorInfo{
		Reason:   "rate_limited",
		Metadata: map[string]string{"retry_after_ms": strconv.FormatInt(retryAfter.Milliseconds(), 10)},
	}

	st2, err := st.WithDetails(ri, ei)
	if err != nil {
		return st.Err()
	}
	return st2.Err()
}
