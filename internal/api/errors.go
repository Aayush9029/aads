package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrorDetail is one structured error returned by Apple Ads.
type ErrorDetail struct {
	MessageCode string                     `json:"messageCode,omitempty"`
	Message     string                     `json:"message,omitempty"`
	Field       string                     `json:"field,omitempty"`
	Unknown     map[string]json.RawMessage `json:"-"`
}

func (detail *ErrorDetail) UnmarshalJSON(data []byte) error {
	type plain ErrorDetail
	if err := json.Unmarshal(data, (*plain)(detail)); err != nil {
		return err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	delete(fields, "messageCode")
	delete(fields, "message")
	delete(fields, "field")
	detail.Unknown = fields
	return nil
}

// ErrorEnvelope is the error member used in successful Apple response
// envelopes as well as non-2xx bodies.
type ErrorEnvelope struct {
	Errors  []ErrorDetail
	Unknown map[string]json.RawMessage
}

func (envelope *ErrorEnvelope) UnmarshalJSON(data []byte) error {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if raw, ok := fields["errors"]; ok {
		if err := json.Unmarshal(raw, &envelope.Errors); err != nil {
			return err
		}
		delete(fields, "errors")
	}
	envelope.Unknown = fields
	return nil
}

// APIError contains the HTTP status and unmodified body for failed requests.
type APIError struct {
	StatusCode int
	Errors     []ErrorDetail
	RawBody    json.RawMessage
	RetryAfter time.Duration
	RequestID  string
}

func (apiError *APIError) Error() string {
	details := make([]string, 0, len(apiError.Errors))
	for _, detail := range apiError.Errors {
		message := detail.Message
		if message == "" {
			message = detail.MessageCode
		}
		if detail.Field != "" {
			message += fmt.Sprintf(" (field: %s)", detail.Field)
		}
		if message != "" {
			details = append(details, message)
		}
	}
	if len(details) == 0 {
		body := strings.TrimSpace(string(apiError.RawBody))
		if body == "" {
			body = http.StatusText(apiError.StatusCode)
		}
		if body == "" {
			body = "request failed"
		}
		details = append(details, body)
	}
	message := fmt.Sprintf("Apple Ads API error %d: %s", apiError.StatusCode, strings.Join(details, "; "))
	if apiError.RequestID != "" {
		message += fmt.Sprintf(" (request ID: %s)", apiError.RequestID)
	}
	return message
}

func newAPIError(statusCode int, body []byte, requestID string) *APIError {
	apiError := &APIError{
		StatusCode: statusCode,
		RawBody:    append(json.RawMessage(nil), body...),
		RequestID:  requestID,
	}
	var direct struct {
		Errors []ErrorDetail `json:"errors"`
		Error  *struct {
			Errors []ErrorDetail `json:"errors"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &direct) == nil {
		apiError.Errors = direct.Errors
		if len(apiError.Errors) == 0 && direct.Error != nil {
			apiError.Errors = direct.Error.Errors
		}
	}
	return apiError
}
