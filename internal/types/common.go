package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	SortAscending  = "ASCENDING"
	SortDescending = "DESCENDING"

	OperatorBetween     = "BETWEEN"
	OperatorContains    = "CONTAINS"
	OperatorContainsAll = "CONTAINS_ALL"
	OperatorContainsAny = "CONTAINS_ANY"
	OperatorEndsWith    = "ENDSWITH"
	OperatorEquals      = "EQUALS"
	OperatorGreaterThan = "GREATER_THAN"
	OperatorIn          = "IN"
	OperatorLessThan    = "LESS_THAN"
	OperatorLike        = "LIKE"
	OperatorStartsWith  = "STARTSWITH"
)

var decimalAmount = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]{1,2})?$`)

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func (m *Money) UnmarshalJSON(data []byte) error {
	m.Amount = ""
	m.Currency = ""
	var object struct {
		Amount   json.RawMessage `json:"amount"`
		Currency string          `json:"currency"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}

	if len(object.Amount) != 0 && !bytes.Equal(object.Amount, []byte("null")) {
		if object.Amount[0] == '"' {
			if err := json.Unmarshal(object.Amount, &m.Amount); err != nil {
				return fmt.Errorf("decode money amount: %w", err)
			}
		} else {
			m.Amount = string(object.Amount)
		}
	}
	m.Currency = object.Currency
	return nil
}

func (m Money) Validate() error {
	if !decimalAmount.MatchString(m.Amount) {
		return errors.New("amount must be a non-negative decimal with at most two fractional digits")
	}
	if len(m.Currency) != 3 || strings.ToUpper(m.Currency) != m.Currency {
		return errors.New("currency must be a three-letter uppercase code")
	}
	for _, character := range m.Currency {
		if character < 'A' || character > 'Z' {
			return errors.New("currency must be a three-letter uppercase code")
		}
	}
	return nil
}

type Selector struct {
	Conditions []*Condition `json:"conditions,omitempty"`
	Fields     []string     `json:"fields,omitempty"`
	OrderBy    []*Sorting   `json:"orderBy,omitempty"`
	Pagination *Pagination  `json:"pagination,omitempty"`
}

type Condition struct {
	Field      string   `json:"field"`
	IgnoreCase bool     `json:"ignoreCase,omitempty"`
	Operator   string   `json:"operator"`
	Values     []string `json:"values"`
}

func (condition Condition) Validate() error {
	if strings.TrimSpace(condition.Field) == "" {
		return errors.New("condition field cannot be empty")
	}
	valid := map[string]bool{
		OperatorBetween: true, OperatorContains: true, OperatorContainsAll: true,
		OperatorContainsAny: true, OperatorEndsWith: true, OperatorEquals: true,
		OperatorGreaterThan: true, OperatorIn: true, OperatorLessThan: true,
		OperatorLike: true, OperatorStartsWith: true,
	}
	if !valid[condition.Operator] {
		return fmt.Errorf("unsupported condition operator %q", condition.Operator)
	}
	return nil
}

type Sorting struct {
	Field      string `json:"field"`
	IgnoreCase bool   `json:"ignoreCase,omitempty"`
	SortOrder  string `json:"sortOrder"`
}

func (sorting Sorting) Validate() error {
	if strings.TrimSpace(sorting.Field) == "" {
		return errors.New("sorting field cannot be empty")
	}
	if sorting.SortOrder != SortAscending && sorting.SortOrder != SortDescending {
		return fmt.Errorf("unsupported sort order %q", sorting.SortOrder)
	}
	return nil
}

type Pagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func (p Pagination) Validate() error {
	if p.Offset < 0 {
		return errors.New("pagination offset cannot be negative")
	}
	if p.Limit < 0 || p.Limit > 1000 {
		return errors.New("pagination limit must be between 0 and 1000")
	}
	return nil
}

func (selector Selector) Validate() error {
	for _, condition := range selector.Conditions {
		if condition == nil {
			return errors.New("selector contains a null condition")
		}
		if err := condition.Validate(); err != nil {
			return err
		}
	}
	for _, sorting := range selector.OrderBy {
		if sorting == nil {
			return errors.New("selector contains a null sorting rule")
		}
		if err := sorting.Validate(); err != nil {
			return err
		}
	}
	if selector.Pagination != nil {
		return selector.Pagination.Validate()
	}
	return nil
}

type PageDetail struct {
	TotalResults int64 `json:"totalResults"`
	StartIndex   int   `json:"startIndex"`
	ItemsPerPage int   `json:"itemsPerPage"`
}

type LOCInvoiceDetails struct {
	BillingContactEmail string `json:"billingContactEmail,omitempty"`
	BuyerEmail          string `json:"buyerEmail,omitempty"`
	BuyerName           string `json:"buyerName,omitempty"`
	ClientName          string `json:"clientName,omitempty"`
	OrderNumber         string `json:"orderNumber,omitempty"`
}
