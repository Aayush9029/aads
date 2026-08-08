package types

import "encoding/json"

type APIResponse[T any] struct {
	Data             *T                         `json:"data,omitempty"`
	Pagination       *PageDetail                `json:"pagination,omitempty"`
	Error            *APIError                  `json:"error,omitempty"`
	RawData          json.RawMessage            `json:"-"`
	RawPagination    json.RawMessage            `json:"-"`
	RawError         json.RawMessage            `json:"-"`
	AdditionalFields map[string]json.RawMessage `json:"-"`
}

func (response *APIResponse[T]) UnmarshalJSON(data []byte) error {
	response.Data = nil
	response.Pagination = nil
	response.Error = nil
	response.RawData = nil
	response.RawPagination = nil
	response.RawError = nil
	response.AdditionalFields = nil

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if raw, ok := fields["data"]; ok {
		response.RawData = append(response.RawData[:0], raw...)
		if string(raw) != "null" {
			var value T
			if err := json.Unmarshal(raw, &value); err != nil {
				return err
			}
			response.Data = &value
		}
	}
	if raw, ok := fields["pagination"]; ok {
		response.RawPagination = append(response.RawPagination[:0], raw...)
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &response.Pagination); err != nil {
				return err
			}
		}
	}
	if raw, ok := fields["error"]; ok {
		response.RawError = append(response.RawError[:0], raw...)
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &response.Error); err != nil {
				return err
			}
		}
	}
	delete(fields, "data")
	delete(fields, "pagination")
	delete(fields, "error")
	if len(fields) > 0 {
		response.AdditionalFields = fields
	}
	return nil
}

func (response *APIResponse[T]) SetData(data T) {
	response.Data = &data
	response.RawData = nil
}

func (response APIResponse[T]) MarshalJSON() ([]byte, error) {
	fields := cloneRawFields(response.AdditionalFields)
	if len(response.RawData) > 0 {
		fields["data"] = response.RawData
	} else if response.Data != nil {
		data, err := json.Marshal(response.Data)
		if err != nil {
			return nil, err
		}
		fields["data"] = data
	}
	if len(response.RawPagination) > 0 {
		fields["pagination"] = response.RawPagination
	} else if response.Pagination != nil {
		data, err := json.Marshal(response.Pagination)
		if err != nil {
			return nil, err
		}
		fields["pagination"] = data
	}
	if len(response.RawError) > 0 {
		fields["error"] = response.RawError
	} else if response.Error != nil {
		data, err := json.Marshal(response.Error)
		if err != nil {
			return nil, err
		}
		fields["error"] = data
	}
	return json.Marshal(fields)
}

type APIListResponse[T any] struct {
	Data             []T                        `json:"data,omitempty"`
	Pagination       *PageDetail                `json:"pagination,omitempty"`
	Error            *APIError                  `json:"error,omitempty"`
	RawData          json.RawMessage            `json:"-"`
	RawPagination    json.RawMessage            `json:"-"`
	RawError         json.RawMessage            `json:"-"`
	AdditionalFields map[string]json.RawMessage `json:"-"`
}

func (response *APIListResponse[T]) UnmarshalJSON(data []byte) error {
	response.Data = nil
	response.Pagination = nil
	response.Error = nil
	response.RawData = nil
	response.RawPagination = nil
	response.RawError = nil
	response.AdditionalFields = nil

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if raw, ok := fields["data"]; ok {
		response.RawData = append(response.RawData[:0], raw...)
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &response.Data); err != nil {
				return err
			}
		}
	}
	if raw, ok := fields["pagination"]; ok {
		response.RawPagination = append(response.RawPagination[:0], raw...)
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &response.Pagination); err != nil {
				return err
			}
		}
	}
	if raw, ok := fields["error"]; ok {
		response.RawError = append(response.RawError[:0], raw...)
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &response.Error); err != nil {
				return err
			}
		}
	}
	delete(fields, "data")
	delete(fields, "pagination")
	delete(fields, "error")
	if len(fields) > 0 {
		response.AdditionalFields = fields
	}
	return nil
}

func (response *APIListResponse[T]) SetData(data []T) {
	response.Data = data
	response.RawData = nil
}

func (response APIListResponse[T]) MarshalJSON() ([]byte, error) {
	fields := cloneRawFields(response.AdditionalFields)
	if len(response.RawData) > 0 {
		fields["data"] = response.RawData
	} else if response.Data != nil {
		data, err := json.Marshal(response.Data)
		if err != nil {
			return nil, err
		}
		fields["data"] = data
	}
	if len(response.RawPagination) > 0 {
		fields["pagination"] = response.RawPagination
	} else if response.Pagination != nil {
		data, err := json.Marshal(response.Pagination)
		if err != nil {
			return nil, err
		}
		fields["pagination"] = data
	}
	if len(response.RawError) > 0 {
		fields["error"] = response.RawError
	} else if response.Error != nil {
		data, err := json.Marshal(response.Error)
		if err != nil {
			return nil, err
		}
		fields["error"] = data
	}
	return json.Marshal(fields)
}

func cloneRawFields(source map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(source)+3)
	for key, value := range source {
		result[key] = value
	}
	return result
}

type APIError struct {
	Errors []APIErrorDetail `json:"errors,omitempty"`
}

type APIErrorDetail struct {
	MessageCode string `json:"messageCode"`
	Message     string `json:"message"`
	Field       string `json:"field,omitempty"`
}

type IntegerResponse struct {
	Data       int32       `json:"data"`
	Pagination *PageDetail `json:"pagination,omitempty"`
	Error      *APIError   `json:"error,omitempty"`
}

type VoidResponse struct {
	Data       map[string]json.RawMessage `json:"data,omitempty"`
	Pagination *PageDetail                `json:"pagination,omitempty"`
	Error      *APIError                  `json:"error,omitempty"`
}
