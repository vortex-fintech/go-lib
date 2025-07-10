package errors

import (
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type ErrorResponse struct {
	Code    codes.Code        `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e ErrorResponse) ToString() string {
	if len(e.Details) > 0 {
		detailsJSON, _ := json.Marshal(map[string]any{
			"code":    e.Code.String(),
			"message": e.Message,
			"details": e.Details,
		})
		return string(detailsJSON)
	}
	return fmt.Sprintf(`{"code":"%s","message":"%s"}`, e.Code.String(), e.Message)
}

func (e ErrorResponse) Error() string {
	return e.ToString()
}

func (e ErrorResponse) ToGRPC() error {
	st := status.New(e.Code, e.Message)

	if len(e.Details) > 0 {
		m := make(map[string]any, len(e.Details))
		for k, v := range e.Details {
			m[k] = v
		}

		if detailsStruct, err := structpb.NewStruct(m); err == nil {
			if stWithDetails, err := st.WithDetails(detailsStruct); err == nil {
				return stWithDetails.Err()
			}
		}
	}

	return st.Err()
}
