package types

import (
	"encoding/json"
	"testing"
)

func TestAPIResponsePreservesUnknownPayloadAndEnvelopeFields(t *testing.T) {
	input := []byte(`{
		"data":{"id":42,"name":"Trip Trek","newAppleField":{"amount":"1.25"}},
		"pagination":null,
		"error":null,
		"requestId":"abc"
	}`)

	var response APIResponse[Campaign]
	if err := json.Unmarshal(input, &response); err != nil {
		t.Fatal(err)
	}
	if response.Data == nil || response.Data.ID != 42 || response.Data.Name != "Trip Trek" {
		t.Fatalf("typed data = %#v", response.Data)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"data", "pagination", "error", "requestId"} {
		if _, ok := roundTrip[key]; !ok {
			t.Fatalf("round trip omitted %q: %s", key, encoded)
		}
	}
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(roundTrip["data"], &rawData); err != nil {
		t.Fatal(err)
	}
	if _, ok := rawData["newAppleField"]; !ok {
		t.Fatalf("round trip lost future field: %s", encoded)
	}
}

func TestAPIListResponseProvidesTypedAndRawData(t *testing.T) {
	input := []byte(`{
		"data":[{"orgId":1,"futureRole":"ANALYST"},{"orgId":2}],
		"pagination":{"totalResults":2,"startIndex":0,"itemsPerPage":2},
		"error":null
	}`)

	var response APIListResponse[UserACL]
	if err := json.Unmarshal(input, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 2 || response.Data[1].OrgID != 2 {
		t.Fatalf("typed data = %#v", response.Data)
	}
	if response.Pagination == nil || response.Pagination.TotalResults != 2 {
		t.Fatalf("pagination = %#v", response.Pagination)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := envelope.Data[0]["futureRole"]; !ok {
		t.Fatalf("round trip lost future list field: %s", encoded)
	}
}

func TestSetDataReplacesPreservedWireData(t *testing.T) {
	var response APIListResponse[UserACL]
	if err := json.Unmarshal([]byte(`{"data":[{"orgId":1,"futureRole":"ANALYST"}]}`), &response); err != nil {
		t.Fatal(err)
	}
	response.SetData([]UserACL{{OrgID: 2}})

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 1 || string(envelope.Data[0]["orgId"]) != "2" {
		t.Fatalf("data was not replaced: %s", encoded)
	}
	if _, ok := envelope.Data[0]["futureRole"]; ok {
		t.Fatalf("stale raw data survived SetData: %s", encoded)
	}
}

func TestAPIErrorShape(t *testing.T) {
	input := []byte(`{"error":{"errors":[{"messageCode":"INVALID_JSON_REQUEST","message":"Invalid request","field":"selector"}]}}`)
	var response APIResponse[Campaign]
	if err := json.Unmarshal(input, &response); err != nil {
		t.Fatal(err)
	}
	if response.Error == nil || len(response.Error.Errors) != 1 {
		t.Fatalf("error = %#v", response.Error)
	}
	if response.Error.Errors[0].Field != "selector" {
		t.Fatalf("error detail = %#v", response.Error.Errors[0])
	}
}
