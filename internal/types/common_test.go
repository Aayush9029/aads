package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMoneyKeepsDecimalText(t *testing.T) {
	for _, input := range []string{
		`{"amount":"12345678901234567890.01","currency":"USD"}`,
		`{"amount":12345678901234567890.01,"currency":"USD"}`,
	} {
		var money Money
		if err := json.Unmarshal([]byte(input), &money); err != nil {
			t.Fatal(err)
		}
		if money.Amount != "12345678901234567890.01" {
			t.Fatalf("amount = %q", money.Amount)
		}

		encoded, err := json.Marshal(money)
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != `{"amount":"12345678901234567890.01","currency":"USD"}` {
			t.Fatalf("encoded money = %s", encoded)
		}
	}
}

func TestMoneyValidation(t *testing.T) {
	tests := []struct {
		name    string
		money   Money
		wantErr bool
	}{
		{name: "integer", money: Money{Amount: "250", Currency: "USD"}},
		{name: "two decimals", money: Money{Amount: "0.25", Currency: "CAD"}},
		{name: "too many decimals", money: Money{Amount: "1.001", Currency: "USD"}, wantErr: true},
		{name: "negative", money: Money{Amount: "-1", Currency: "USD"}, wantErr: true},
		{name: "lowercase currency", money: Money{Amount: "1", Currency: "usd"}, wantErr: true},
		{name: "non-letter currency", money: Money{Amount: "1", Currency: "U1D"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.money.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestSelectorValidationAndJSON(t *testing.T) {
	selector := Selector{
		Conditions: []*Condition{{
			Field:      "name",
			IgnoreCase: true,
			Operator:   OperatorStartsWith,
			Values:     []string{"trip"},
		}},
		Fields: []string{"id", "name"},
		OrderBy: []*Sorting{{
			Field:     "name",
			SortOrder: SortAscending,
		}},
		Pagination: &Pagination{Offset: 20, Limit: 1000},
	}

	if err := selector.Validate(); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(selector)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"ignoreCase":true`, `"operator":"STARTSWITH"`, `"limit":1000`} {
		if !strings.Contains(string(encoded), fragment) {
			t.Fatalf("selector JSON %s does not contain %s", encoded, fragment)
		}
	}
}

func TestSelectorRejectsInvalidRules(t *testing.T) {
	tests := []Selector{
		{Conditions: []*Condition{{Field: "name", Operator: "UNKNOWN"}}},
		{OrderBy: []*Sorting{{Field: "name", SortOrder: "UP"}}},
		{Pagination: &Pagination{Limit: 1001}},
		{Pagination: &Pagination{Offset: -1}},
		{Conditions: []*Condition{nil}},
	}
	for _, selector := range tests {
		if err := selector.Validate(); err == nil {
			t.Fatalf("Validate() accepted %#v", selector)
		}
	}
}
