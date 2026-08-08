package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Aayush9029/aads/internal/api"
	"github.com/Aayush9029/aads/internal/commands"
	"github.com/Aayush9029/aads/internal/config"
)

func TestOperationClassifiesRetrySemantics(t *testing.T) {
	tests := []struct {
		name    string
		request commands.Request
		want    api.Operation
	}{
		{name: "GET", request: commands.Request{Method: http.MethodGet, Operation: "campaigns.list"}, want: api.OperationRead},
		{name: "find", request: commands.Request{Method: http.MethodPost, Operation: "campaigns.find"}, want: api.OperationFind},
		{name: "eligibility find", request: commands.Request{Method: http.MethodPost, Operation: "apps.eligibility"}, want: api.OperationFind},
		{name: "report", request: commands.Request{Method: http.MethodPost, Operation: "reports.campaigns"}, want: api.OperationReport},
		{name: "impression share create is unsafe", request: commands.Request{Method: http.MethodPost, Operation: "impression-share.create"}, want: api.OperationCreate},
		{name: "impression share read", request: commands.Request{Method: http.MethodGet, Operation: "impression-share.get"}, want: api.OperationRead},
		{name: "geo detail POST is replay safe", request: commands.Request{Method: http.MethodPost, Operation: "geo.get"}, want: api.OperationRead},
		{name: "create", request: commands.Request{Method: http.MethodPost, Operation: "campaigns.create"}, want: api.OperationCreate},
		{name: "update", request: commands.Request{Method: http.MethodPut, Operation: "campaigns.update"}, want: api.OperationUpdate},
		{name: "delete", request: commands.Request{Method: http.MethodDelete, Operation: "campaigns.delete"}, want: api.OperationDelete},
		{name: "bulk delete", request: commands.Request{Method: http.MethodPost, Operation: "keywords.delete"}, want: api.OperationBulkDelete},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := operation(test.request); got != test.want {
				t.Errorf("operation() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPaginationValuesReadsQueryAndSelector(t *testing.T) {
	tests := []struct {
		name       string
		request    commands.Request
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "query",
			request:    commands.Request{Query: url.Values{"limit": {"50"}, "offset": {"100"}}},
			wantLimit:  50,
			wantOffset: 100,
		},
		{
			name: "top-level body",
			request: commands.Request{Body: map[string]any{"pagination": map[string]any{
				"limit": json.Number("25"), "offset": float64(75),
			}}},
			wantLimit: 25, wantOffset: 75,
		},
		{
			name: "report selector overrides query",
			request: commands.Request{
				Query: url.Values{"limit": {"10"}, "offset": {"20"}},
				Body: map[string]any{"selector": map[string]any{"pagination": map[string]any{
					"limit": 3, "offset": json.Number("9"),
				}}},
			},
			wantLimit: 3, wantOffset: 9,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limit, offset := paginationValues(test.request)
			if limit != test.wantLimit || offset != test.wantOffset {
				t.Errorf("paginationValues() = (%d, %d), want (%d, %d)", limit, offset, test.wantLimit, test.wantOffset)
			}
		})
	}
}

func TestWithPaginationClonesInput(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		original := commands.Request{Query: url.Values{"limit": {"10"}, "offset": {"5"}, "state": {"ENABLED"}}}
		page := withPagination(original, 1000, 2005)
		if page.Query.Get("limit") != "1000" || page.Query.Get("offset") != "2005" || page.Query.Get("state") != "ENABLED" {
			t.Errorf("page query = %v", page.Query)
		}
		if original.Query.Get("limit") != "10" || original.Query.Get("offset") != "5" {
			t.Errorf("original query was mutated: %v", original.Query)
		}
	})

	t.Run("find body", func(t *testing.T) {
		body := map[string]any{"conditions": []any{"unchanged"}, "pagination": map[string]any{"limit": 20, "offset": 7}}
		original := commands.Request{Query: url.Values{"expand": {"true"}}, Body: body}
		page := withPagination(original, 40, 47)
		pageBody := page.Body.(map[string]any)
		pagination := pageBody["pagination"].(map[string]any)
		if pagination["limit"] != 40 || pagination["offset"] != 47 {
			t.Errorf("page pagination = %#v", pagination)
		}
		originalPagination := body["pagination"].(map[string]any)
		if originalPagination["limit"] != 20 || originalPagination["offset"] != 7 {
			t.Errorf("original body was mutated: %#v", body)
		}
		if page.Query.Get("expand") != "true" {
			t.Errorf("page query = %v", page.Query)
		}
	})

	t.Run("report selector", func(t *testing.T) {
		body := map[string]any{
			"startTime": "2026-01-01",
			"selector": map[string]any{
				"conditions": []any{"unchanged"},
				"pagination": map[string]any{"limit": 2, "offset": 4},
			},
		}
		original := commands.Request{Body: body}
		page := withPagination(original, 2, 6)
		pageBody := page.Body.(map[string]any)
		selector := pageBody["selector"].(map[string]any)
		pagination := selector["pagination"].(map[string]any)
		if pagination["limit"] != 2 || pagination["offset"] != 6 {
			t.Errorf("selector pagination = %#v", pagination)
		}
		if _, exists := pageBody["pagination"]; exists {
			t.Errorf("report pagination was added at the top level: %#v", pageBody)
		}
		originalSelector := body["selector"].(map[string]any)
		originalPagination := originalSelector["pagination"].(map[string]any)
		if originalPagination["offset"] != 4 {
			t.Errorf("original selector was mutated: %#v", originalSelector)
		}
	})

}

type fixedTokenSource struct{}

func (fixedTokenSource) Token(context.Context) (string, error) { return "test-token", nil }
func (fixedTokenSource) Invalidate()                           {}

func newAdapter(t *testing.T, handler http.HandlerFunc) *apiAdapter {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := api.NewClient(
		&config.Config{OrgID: "123"},
		api.WithBaseURL(server.URL),
		api.WithTokenSource(fixedTokenSource{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return &apiAdapter{client: client}
}

func TestAPIAdapterSingleRequest(t *testing.T) {
	adapter := newAdapter(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/campaigns/find" || request.URL.Query().Get("limit") != "20" {
			t.Errorf("request = %s %s", request.Method, request.URL.String())
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("X-AP-Context"); got != "orgId=123" {
			t.Errorf("X-AP-Context = %q", got)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if !jsonEqual(body, []byte(`{"selector":"value"}`)) {
			t.Errorf("body = %s", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		io.WriteString(writer, `{"data":[{"id":1}],"pagination":{"totalResults":1},"future":"kept"}`)
	})

	result, err := adapter.Do(context.Background(), commands.Request{
		Method: http.MethodPost, Path: "/campaigns/find", Query: url.Values{"limit": {"20"}},
		Body: map[string]any{"selector": "value"}, Operation: "campaigns.find", OrgContext: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(result.Data.(json.RawMessage), []byte(`[{"id":1}]`)) {
		t.Errorf("data = %s", result.Data)
	}
	if !jsonEqual(result.Envelope.(json.RawMessage), []byte(`{"data":[{"id":1}],"pagination":{"totalResults":1},"future":"kept"}`)) {
		t.Errorf("envelope = %s", result.Envelope)
	}
}

func TestAPIAdapterOmitsOrgContextForACL(t *testing.T) {
	adapter := newAdapter(t, func(writer http.ResponseWriter, request *http.Request) {
		if value := request.Header.Get("X-AP-Context"); value != "" {
			t.Errorf("X-AP-Context = %q, want omitted", value)
		}
		io.WriteString(writer, `{"data":[]}`)
	})
	_, err := adapter.Do(context.Background(), commands.Request{Method: http.MethodGet, Path: "/acls", Operation: "acls.list"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIAdapterCollectsQueryPagesWithoutMutatingRequest(t *testing.T) {
	var offsets []int
	adapter := newAdapter(t, func(writer http.ResponseWriter, request *http.Request) {
		limit, _ := strconv.Atoi(request.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(request.URL.Query().Get("offset"))
		offsets = append(offsets, offset)
		if limit != 2 {
			t.Errorf("limit = %d, want 2", limit)
		}
		var data string
		switch offset {
		case 5:
			data = `[{"id":5},{"id":6}]`
		case 7:
			data = `[{"id":7}]`
		default:
			t.Errorf("unexpected offset %d", offset)
			data = `[]`
		}
		io.WriteString(writer, `{"data":`+data+`,"pagination":{"totalResults":8,"startIndex":`+strconv.Itoa(offset)+`,"itemsPerPage":2},"grandTotals":{"spend":"10.00"}}`)
	})

	request := commands.Request{
		Method: http.MethodGet, Path: "/campaigns", Query: url.Values{"limit": {"2"}, "offset": {"5"}},
		Operation: "campaigns.list", OrgContext: true, AllPages: true,
	}
	result, err := adapter.Do(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(offsets, []int{5, 7}) {
		t.Errorf("offsets = %v, want [5 7]", offsets)
	}
	if request.Query.Get("offset") != "5" {
		t.Errorf("request query was mutated: %v", request.Query)
	}
	if !jsonEqual(result.Data.(json.RawMessage), []byte(`[{"id":5},{"id":6},{"id":7}]`)) {
		t.Errorf("combined data = %s", result.Data)
	}
	envelope := result.Envelope.(json.RawMessage)
	for _, wanted := range []string{`"grandTotals"`, `"spend":"10.00"`, `"totalResults":8`} {
		if !strings.Contains(string(envelope), wanted) {
			t.Errorf("envelope %s does not contain %s", envelope, wanted)
		}
	}
}

func TestAPIAdapterCollectsReportPagesInsideSelector(t *testing.T) {
	var offsets []int
	adapter := newAdapter(t, func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		decoder := json.NewDecoder(request.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, exists := body["pagination"]; exists {
			t.Errorf("report body has top-level pagination: %#v", body)
		}
		selector := body["selector"].(map[string]any)
		pagination := selector["pagination"].(map[string]any)
		offset, _ := strconv.Atoi(pagination["offset"].(json.Number).String())
		limit, _ := strconv.Atoi(pagination["limit"].(json.Number).String())
		offsets = append(offsets, offset)
		if limit != 1 {
			t.Errorf("limit = %d, want 1", limit)
		}
		io.WriteString(writer, `{"data":{"reportingDataResponse":{"row":[{"row":`+strconv.Itoa(offset)+`}],"grandTotals":{"spend":"12.00"}}},"pagination":{"totalResults":4,"startIndex":`+strconv.Itoa(offset)+`,"itemsPerPage":1}}`)
	})

	body := map[string]any{
		"startTime": "2026-01-01", "endTime": "2026-01-02",
		"selector": map[string]any{"pagination": map[string]any{"limit": 1, "offset": 2}},
	}
	request := commands.Request{
		Method: http.MethodPost, Path: "/reports/campaigns", Body: body,
		Operation: "reports.campaigns", OrgContext: true, AllPages: true,
	}
	result, err := adapter.Do(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(offsets, []int{2, 3}) {
		t.Errorf("offsets = %v, want [2 3]", offsets)
	}
	if !jsonEqual(result.Data.(json.RawMessage), []byte(`{"reportingDataResponse":{"row":[{"row":2},{"row":3}],"grandTotals":{"spend":"12.00"}}}`)) {
		t.Errorf("combined data = %s", result.Data)
	}
	originalSelector := body["selector"].(map[string]any)
	originalPagination := originalSelector["pagination"].(map[string]any)
	if originalPagination["offset"] != 2 {
		t.Errorf("original body was mutated: %#v", body)
	}
}

func TestAPIAdapterUsesCustomReportPageLimit(t *testing.T) {
	var offsets []int
	adapter := newAdapter(t, func(writer http.ResponseWriter, request *http.Request) {
		if got := request.URL.Query().Get("limit"); got != "50" {
			t.Errorf("limit = %q, want 50", got)
		}
		offset, _ := strconv.Atoi(request.URL.Query().Get("offset"))
		offsets = append(offsets, offset)
		count := 50
		if offset == 50 {
			count = 1
		}
		items := make([]map[string]any, count)
		for index := range items {
			items[index] = map[string]any{"id": offset + index + 1}
		}
		json.NewEncoder(writer).Encode(map[string]any{"data": items, "pagination": map[string]any{"totalResults": 51, "startIndex": offset, "itemsPerPage": count}})
	})
	result, err := adapter.Do(context.Background(), commands.Request{Method: http.MethodGet, Path: "/custom-reports", Operation: "impression-share.list", OrgContext: true, AllPages: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(offsets, []int{0, 50}) {
		t.Errorf("offsets = %v, want [0 50]", offsets)
	}
	var items []any
	if err := json.Unmarshal(result.Data.(json.RawMessage), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 51 {
		t.Errorf("items = %d, want 51", len(items))
	}
}

func TestAPIAdapterEmptyBodyReturnsEmptyResult(t *testing.T) {
	adapter := newAdapter(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	result, err := adapter.Do(context.Background(), commands.Request{Method: http.MethodDelete, Path: "/campaigns/1", Operation: "campaigns.delete", OrgContext: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data != nil || result.Envelope != nil {
		t.Errorf("result = %#v, want empty", result)
	}
}

func jsonEqual(left, right []byte) bool {
	var leftValue any
	var rightValue any
	return json.Unmarshal(left, &leftValue) == nil && json.Unmarshal(right, &rightValue) == nil && reflect.DeepEqual(leftValue, rightValue)
}
