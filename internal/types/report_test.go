package types

import (
	"encoding/json"
	"testing"
)

func TestReportingResponseV5MetricsAndFlexibleMetadata(t *testing.T) {
	input := []byte(`{
		"reportingDataResponse":{
			"row":[{
				"metadata":{"campaignId":123,"futureDimension":{"name":"value"}},
				"total":{
					"impressions":1000,
					"taps":45,
					"tapInstalls":12,
					"totalInstalls":15,
					"tapInstallRate":0.266666,
					"localSpend":{"amount":"12345678901234567890.01","currency":"USD"}
				},
				"granularity":[{"date":"2026-08-01","impressions":500}]
			}],
			"grandTotals":{"other":false,"total":{"impressions":1000}}
		}
	}`)

	var response ReportingResponse
	if err := json.Unmarshal(input, &response); err != nil {
		t.Fatal(err)
	}
	row := response.ReportingDataResponse.Row[0]
	if row.Total.TapInstalls != 12 || row.Total.TotalInstalls != 15 {
		t.Fatalf("metrics = %#v", row.Total)
	}
	if row.Total.LocalSpend.Amount != "12345678901234567890.01" {
		t.Fatalf("spend = %#v", row.Total.LocalSpend)
	}
	if _, ok := row.Metadata["futureDimension"]; !ok {
		t.Fatalf("metadata = %#v", row.Metadata)
	}
	if row.Granularity[0].Date != "2026-08-01" || row.Granularity[0].Impressions != 500 {
		t.Fatalf("granularity = %#v", row.Granularity)
	}
}

func TestReportingRequestIncludesRequiredSelector(t *testing.T) {
	request := ReportingRequest{
		StartTime:         "2026-08-01",
		EndTime:           "2026-08-07",
		Granularity:       "DAILY",
		ReturnGrandTotals: true,
		Selector: &Selector{
			Pagination: &Pagination{Offset: 0, Limit: 1000},
		},
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["selector"]; !ok {
		t.Fatalf("request omitted selector: %s", encoded)
	}
}

func TestImpressionShareRequestAndLifecycleResponse(t *testing.T) {
	request := CustomReportRequest{
		Name:        "impression share",
		StartTime:   "2026-07-01",
		EndTime:     "2026-07-31",
		Granularity: "DAILY",
		Selector: &SovSelector{Conditions: []*SovCondition{{
			Field:    "countryOrRegion",
			Operator: "IN",
			Values:   []string{"US", "CA"},
		}}},
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, encoded, []byte(`{
		"name":"impression share",
		"startTime":"2026-07-01",
		"endTime":"2026-07-31",
		"granularity":"DAILY",
		"selector":{"conditions":[{"field":"countryOrRegion","operator":"IN","values":["US","CA"]}]}
	}`))

	responseInput := []byte(`{
		"id":986235,
		"name":"impression share",
		"state":"COMPLETED",
		"downloadUri":"https://example.com/report.csv",
		"dimensions":["appName","adamId","countryOrRegion","searchTerm"],
		"metrics":["lowImpressionShare","highImpressionShare","rank","searchPopularity"]
	}`)
	var response CustomReportResponse
	if err := json.Unmarshal(responseInput, &response); err != nil {
		t.Fatal(err)
	}
	if response.State != "COMPLETED" || response.DownloadURI == "" || len(response.Metrics) != 4 {
		t.Fatalf("response = %#v", response)
	}
}
