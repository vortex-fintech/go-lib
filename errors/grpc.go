package errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

// ToGRPC — ErrorResponse → gRPC error с ErrorInfo + BadRequest (если нужно).
func (e ErrorResponse) ToGRPC() error {
	st := status.New(e.Code, e.Message)

	// ErrorInfo: reason + metadata
	if e.Reason != "" || len(e.Details) > 0 {
		ei := &errdetails.ErrorInfo{
			Reason:   string(e.Reason),
			Metadata: map[string]string{},
		}
		for k, v := range e.Details {
			ei.Metadata[k] = v
		}
		if st2, err := st.WithDetails(ei); err == nil {
			st = st2
		}
	}

	// BadRequest: field violations (только для InvalidArgument)
	if len(e.Violations) > 0 && e.Code == codes.InvalidArgument {
		br := &errdetails.BadRequest{
			FieldViolations: make([]*errdetails.BadRequest_FieldViolation, 0, len(e.Violations)),
		}
		for _, v := range e.Violations {
			desc := v.Description
			if desc == "" {
				desc = v.Reason
			}
			br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       v.Field,
				Description: desc,
			})
		}
		if st2, err := st.WithDetails(br); err == nil {
			st = st2
		}
	}

	return st.Err()
}

// FromGRPC — обратное преобразование gRPC error → ErrorResponse.
func FromGRPC(err error) ErrorResponse {
	st, ok := status.FromError(err)
	if !ok {
		return Unknown()
	}
	out := New(st.Message(), st.Code(), nil)
	for _, d := range st.Details() {
		switch x := d.(type) {
		case *errdetails.ErrorInfo:
			if x.GetReason() != "" {
				out.Reason = Reason(x.GetReason())
			}
			if md := x.GetMetadata(); len(md) > 0 {
				out = out.WithDetails(md)
			}
		case *errdetails.BadRequest:
			if len(x.FieldViolations) > 0 {
				vs := make([]FieldViolation, 0, len(x.FieldViolations))
				for _, fv := range x.FieldViolations {
					vs = append(vs, FieldViolation{
						Field:       fv.GetField(),
						Description: fv.GetDescription(),
					})
				}
				out.Violations = vs
			}
		}
	}
	return out
}
