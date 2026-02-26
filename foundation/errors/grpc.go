package errors

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

const violationReasonMetadataPrefix = "_errors.violation_reason."

func (e ErrorResponse) ToGRPC() error {
	st := status.New(e.Code, e.Message)

	metadata := cloneDetails(e.Details)
	for _, v := range e.Violations {
		if v.Field == "" || v.Reason == "" {
			continue
		}
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata[violationReasonMetadataPrefix+v.Field] = v.Reason
	}

	// ErrorInfo: reason + metadata + domain (if provided).
	if e.Reason != "" || len(metadata) > 0 || e.Domain != "" {
		ei := &errdetails.ErrorInfo{
			Reason:   string(e.Reason),
			Domain:   e.Domain,
			Metadata: metadata,
		}
		if st2, err := st.WithDetails(ei); err == nil {
			st = st2
		}
	}

	// BadRequest details for InvalidArgument.
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

func FromGRPC(err error) ErrorResponse {
	st, ok := status.FromError(err)
	if !ok {
		return Unknown()
	}
	out := New(st.Message(), st.Code(), nil)
	var violationReasons map[string]string
	for _, d := range st.Details() {
		switch x := d.(type) {
		case *errdetails.ErrorInfo:
			if x.GetReason() != "" {
				out.Reason = Reason(x.GetReason())
			}
			if dom := x.GetDomain(); dom != "" {
				out.Domain = dom
			}
			if md := x.GetMetadata(); len(md) > 0 {
				details := make(map[string]string, len(md))
				for k, v := range md {
					if strings.HasPrefix(k, violationReasonMetadataPrefix) {
						field := strings.TrimPrefix(k, violationReasonMetadataPrefix)
						if field == "" {
							continue
						}
						if violationReasons == nil {
							violationReasons = map[string]string{}
						}
						violationReasons[field] = v
						continue
					}
					details[k] = v
				}
				if len(details) > 0 {
					out = out.WithDetails(details)
				}
			}
		case *errdetails.BadRequest:
			if len(x.FieldViolations) > 0 {
				vs := make([]FieldViolation, 0, len(x.FieldViolations))
				for _, fv := range x.FieldViolations {
					field := fv.GetField()
					violation := FieldViolation{Field: field, Description: fv.GetDescription()}
					if reason, ok := violationReasons[field]; ok {
						violation.Reason = reason
					}
					vs = append(vs, violation)
				}
				out.Violations = vs
			}
		}
	}
	return out
}
