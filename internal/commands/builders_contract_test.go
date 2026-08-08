package commands

import (
	"reflect"
	"strings"
	"testing"
)

func TestCommandQueriesAreScopedToOfficialParameters(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		values  Values
		globals Globals
		want    map[string]string
	}{
		{name: "app search", key: "apps search", values: Values{"query": "notes", "return-owned-apps": "true", "limit": "25", "offset": "50", "all": "true"}, want: map[string]string{"query": "notes", "returnOwnedApps": "true", "limit": "25", "offset": "50"}},
		{name: "campaign list fields", key: "campaigns list", values: Values{"limit": "10", "offset": "20"}, globals: Globals{Fields: "id,name"}, want: map[string]string{"limit": "10", "offset": "20", "fields": "id,name"}},
		{name: "campaign get fields", key: "campaigns get", values: Values{"id": "101"}, globals: Globals{Fields: "id,name"}, want: map[string]string{"fields": "id,name"}},
		{name: "ad group list fields", key: "adgroups list", values: Values{"campaign-id": "101", "limit": "10", "offset": "20"}, globals: Globals{Fields: "id,name"}, want: map[string]string{"limit": "10", "offset": "20", "fields": "id,name"}},
		{name: "ad group get fields", key: "adgroups get", values: Values{"campaign-id": "101", "id": "202"}, globals: Globals{Fields: "id,name"}, want: map[string]string{"fields": "id,name"}},
		{name: "creative include deleted assets", key: "creatives get", values: Values{"id": "707", "include-deleted-creative-set-assets": "true"}, want: map[string]string{"includeDeletedCreativeSetAssets": "true"}},
		{name: "product page filters", key: "product-pages list", values: Values{"adam-id": "303", "name": "Launch", "states": "RUNNING,PAUSED"}, want: map[string]string{"name": "Launch", "states": "RUNNING,PAUSED"}},
		{name: "product page locale filters", key: "product-pages locales", values: Values{"adam-id": "303", "id": "page-one", "device-classes": "IPHONE", "expand": "true", "language-codes": "en-US", "languages": "en"}, want: map[string]string{"deviceClasses": "IPHONE", "expand": "true", "languageCodes": "en-US", "languages": "en"}},
		{name: "supported countries filter", key: "product-pages countries", values: Values{"countries-or-regions": "US,CA"}, want: map[string]string{"countriesOrRegions": "US,CA"}},
		{name: "geo search", key: "geo search", values: Values{"query": "Toronto", "country-code": "CA", "entity": "Locality", "limit": "10", "offset": "20", "all": "true"}, want: map[string]string{"query": "Toronto", "countrycode": "CA", "entity": "Locality", "limit": "10", "offset": "20"}},
		{name: "geo lookup", key: "geo get", values: Values{"geo-id": "geo/toronto", "entity": "LOCALITY", "limit": "10", "offset": "20", "all": "true"}, want: map[string]string{"limit": "10", "offset": "20"}},
		{name: "custom report list", key: "impression-share list", values: Values{"field": "modificationTime", "sort-order": "DESCENDING", "limit": "10", "offset": "20", "all": "true"}, want: map[string]string{"field": "modificationTime", "sortOrder": "DESCENDING", "limit": "10", "offset": "20"}},
		{name: "create name stays in body", key: "campaigns create", values: Values{"name": "Launch", "adam-id": "303", "daily-budget": "100", "countries": "US", "ad-channel-type": "SEARCH", "billing-event": "TAPS", "supply-sources": "APPSTORE_SEARCH_RESULTS"}, globals: Globals{Currency: "USD"}},
		{name: "selector field stays in body", key: "campaigns find", values: Values{"field": "status", "op": "equals", "values": "ENABLED"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := buildForTest(t, test.key, test.values, test.globals, contractInput{})
			if got := len(request.Query); got != len(test.want) {
				t.Fatalf("query = %v, want exactly %v", request.Query, test.want)
			}
			for key, want := range test.want {
				if got := request.Query.Get(key); got != want {
					t.Errorf("query[%q] = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestAllBulkBodiesAreRawNonemptyArrays(t *testing.T) {
	array := []any{map[string]any{"id": "505"}}
	input := contractInput{"array": array, "object": map[string]any{"id": "505"}, "empty": []any{}}
	tests := []struct {
		key    string
		values Values
	}{
		{key: "keywords create", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
		{key: "keywords update", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
		{key: "negatives campaign-create", values: Values{"campaign-id": "101", "from-json": "array"}},
		{key: "negatives campaign-update", values: Values{"campaign-id": "101", "from-json": "array"}},
		{key: "negatives adgroup-create", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
		{key: "negatives adgroup-update", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
		{key: "geo get", values: Values{"from-json": "array"}},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			request := buildForTest(t, test.key, test.values, Globals{Currency: "USD"}, input)
			if !reflect.DeepEqual(request.Body, array) {
				t.Errorf("Body = %#v, want raw array %#v", request.Body, array)
			}
			for _, source := range []string{"object", "empty"} {
				values := cloneValues(test.values)
				values["from-json"] = source
				if _, err := buildForTestError(test.key, values, Globals{Currency: "USD"}, input); err == nil {
					t.Errorf("Build(from-json=%s) succeeded, want nonempty-array error", source)
				}
			}
		})
	}

	deleteTests := []struct {
		key    string
		values Values
	}{
		{key: "keywords delete", values: Values{"campaign-id": "101", "adgroup-id": "202", "ids": "505, 506"}},
		{key: "negatives campaign-delete", values: Values{"campaign-id": "101", "ids": "505, 506"}},
		{key: "negatives adgroup-delete", values: Values{"campaign-id": "101", "adgroup-id": "202", "ids": "505, 506"}},
	}
	for _, test := range deleteTests {
		t.Run(test.key, func(t *testing.T) {
			request := buildForTest(t, test.key, test.values, Globals{}, contractInput{})
			if want := []int64{505, 506}; !reflect.DeepEqual(request.Body, want) {
				t.Errorf("Body = %#v, want raw IDs %#v", request.Body, want)
			}
		})
	}
}

func TestTypedCreateBodies(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		values Values
		want   any
	}{
		{
			name: "campaign", key: "campaigns create",
			values: Values{"name": "Launch", "adam-id": "303", "daily-budget": "100.25", "countries": "us, ca", "ad-channel-type": "SEARCH", "billing-event": "TAPS", "supply-sources": "APPSTORE_SEARCH_RESULTS, APPSTORE_PRODUCT_PAGES_BROWSE"},
			want:   map[string]any{"name": "Launch", "adamId": int64(303), "dailyBudgetAmount": map[string]any{"amount": "100.25", "currency": "USD"}, "countriesOrRegions": []string{"US", "CA"}, "adChannelType": "SEARCH", "billingEvent": "TAPS", "supplySources": []string{"APPSTORE_SEARCH_RESULTS", "APPSTORE_PRODUCT_PAGES_BROWSE"}},
		},
		{
			name: "ad group", key: "adgroups create",
			values: Values{"campaign-id": "101", "name": "Broad", "default-bid": "1.25", "search-match": "true", "pricing-model": "CPC", "start-time": "2026-08-08T12:30:00Z"},
			want:   map[string]any{"name": "Broad", "defaultBidAmount": map[string]any{"amount": "1.25", "currency": "USD"}, "automatedKeywordsOptIn": true, "pricingModel": "CPC", "startTime": "2026-08-08T12:30:00Z"},
		},
		{
			name: "targeting keyword", key: "keywords create",
			values: Values{"campaign-id": "101", "adgroup-id": "202", "text": "focus timer", "bid": "2.50"},
			want:   []any{map[string]any{"text": "focus timer", "matchType": "BROAD", "bidAmount": map[string]any{"amount": "2.50", "currency": "USD"}}},
		},
		{
			name: "campaign negative", key: "negatives campaign-create",
			values: Values{"campaign-id": "101", "text": "free"},
			want:   []any{map[string]any{"text": "free", "matchType": "BROAD"}},
		},
		{
			name: "ad", key: "ads create",
			values: Values{"campaign-id": "101", "adgroup-id": "202", "creative-id": "707", "name": "Launch", "status": "enabled"},
			want:   map[string]any{"creativeId": int64(707), "name": "Launch", "status": "ENABLED"},
		},
		{
			name: "creative", key: "creatives create",
			values: Values{"adam-id": "303", "product-page-id": "page-one", "name": "Launch", "type": "custom_product_page"},
			want:   map[string]any{"adamId": int64(303), "productPageId": "page-one", "name": "Launch", "type": "CUSTOM_PRODUCT_PAGE"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := buildForTest(t, test.key, test.values, Globals{Currency: "USD"}, contractInput{})
			if !reflect.DeepEqual(request.Body, test.want) {
				t.Errorf("Body = %#v\nwant = %#v", request.Body, test.want)
			}
		})
	}
}

func TestObjectJSONMutationsRequireNonemptyObjects(t *testing.T) {
	tests := []struct {
		key    string
		values Values
	}{
		{key: "campaigns create", values: Values{"from-json": "invalid"}},
		{key: "campaigns update", values: Values{"id": "101", "from-json": "invalid"}},
		{key: "budgetorders create", values: Values{"from-json": "invalid"}},
		{key: "budgetorders update", values: Values{"id": "404", "from-json": "invalid"}},
		{key: "adgroups create", values: Values{"campaign-id": "101", "from-json": "invalid"}},
		{key: "adgroups update", values: Values{"campaign-id": "101", "id": "202", "from-json": "invalid"}},
		{key: "ads create", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "invalid"}},
		{key: "ads update", values: Values{"campaign-id": "101", "adgroup-id": "202", "id": "606", "from-json": "invalid"}},
		{key: "creatives create", values: Values{"from-json": "invalid"}},
		{key: "impression-share create", values: Values{"from-json": "invalid"}},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			for name, body := range map[string]any{"array": []any{map[string]any{"id": 1}}, "empty": map[string]any{}} {
				values := cloneValues(test.values)
				values["from-json"] = name
				if _, err := buildForTestError(test.key, values, Globals{Currency: "USD"}, contractInput{name: body}); err == nil {
					t.Errorf("Build(from-json=%s) succeeded, want nonempty-object error", name)
				}
			}
		})
	}
}

func TestMoneyAndCountryValidation(t *testing.T) {
	baseCampaign := Values{"name": "x", "adam-id": "303", "daily-budget": "1", "countries": "US", "ad-channel-type": "SEARCH", "billing-event": "TAPS", "supply-sources": "APPSTORE_SEARCH_RESULTS"}
	for _, amount := range []string{"0", "-1", "NaN", "Inf", "1e3", "1.001"} {
		t.Run("money/"+amount, func(t *testing.T) {
			values := cloneValues(baseCampaign)
			values["daily-budget"] = amount
			if _, err := buildForTestError("campaigns create", values, Globals{Currency: "USD"}, contractInput{}); err == nil {
				t.Errorf("Build(daily-budget=%q) succeeded, want positive decimal money error", amount)
			}
		})
	}
	for _, currency := range []string{"US", "US1", "ZZZ"} {
		t.Run("currency/"+currency, func(t *testing.T) {
			if _, err := buildForTestError("campaigns create", baseCampaign, Globals{Currency: currency}, contractInput{}); err == nil {
				t.Errorf("Build(currency=%q) succeeded, want ISO 4217 currency error", currency)
			}
		})
	}
	for _, countries := range []string{"USA", "U1", "ZZ"} {
		t.Run("countries/"+countries, func(t *testing.T) {
			values := cloneValues(baseCampaign)
			values["countries"] = countries
			if _, err := buildForTestError("campaigns create", values, Globals{Currency: "USD"}, contractInput{}); err == nil {
				t.Errorf("Build(countries=%q) succeeded, want ISO 3166 country error", countries)
			}
		})
	}
}

func TestTypedCreateValidation(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		values Values
		want   string
	}{
		{name: "campaign requires name", key: "campaigns create", values: Values{}, want: "--name is required"},
		{name: "campaign status enum", key: "campaigns create", values: Values{"name": "x", "adam-id": "303", "daily-budget": "1", "countries": "US", "ad-channel-type": "SEARCH", "billing-event": "TAPS", "supply-sources": "APPSTORE_SEARCH_RESULTS", "status": "running"}, want: "--status must be one of"},
		{name: "campaign channel enum", key: "campaigns create", values: Values{"name": "x", "adam-id": "303", "daily-budget": "1", "countries": "US", "ad-channel-type": "VIDEO", "billing-event": "TAPS", "supply-sources": "APPSTORE_SEARCH_RESULTS"}, want: "--ad-channel-type must be one of"},
		{name: "campaign billing enum", key: "campaigns create", values: Values{"name": "x", "adam-id": "303", "daily-budget": "1", "countries": "US", "ad-channel-type": "SEARCH", "billing-event": "CLICKS", "supply-sources": "APPSTORE_SEARCH_RESULTS"}, want: "--billing-event must be one of"},
		{name: "campaign supply enum", key: "campaigns create", values: Values{"name": "x", "adam-id": "303", "daily-budget": "1", "countries": "US", "ad-channel-type": "SEARCH", "billing-event": "TAPS", "supply-sources": "APPSTORE_PRODUCT_PAGES"}, want: "--supply-sources must be one of"},
		{name: "ad group pricing enum", key: "adgroups create", values: Values{"campaign-id": "101", "name": "x", "default-bid": "1", "pricing-model": "CPA", "start-time": "2026-08-08"}, want: "--pricing-model must be one of"},
		{name: "keyword match enum", key: "keywords create", values: Values{"campaign-id": "101", "adgroup-id": "202", "text": "x", "match-type": "PHRASE"}, want: "--match-type must be one of"},
		{name: "ad status enum", key: "ads create", values: Values{"campaign-id": "101", "adgroup-id": "202", "creative-id": "707", "name": "x", "status": "RUNNING"}, want: "--status must be one of"},
		{name: "creative type enum", key: "creatives create", values: Values{"adam-id": "303", "product-page-id": "page", "name": "x", "type": "CREATIVE_SET"}, want: "--type must be one of"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildForTestError(test.key, test.values, Globals{Currency: "USD"}, contractInput{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Build() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestCampaignUpdateEnvelope(t *testing.T) {
	t.Run("typed flags wrap campaign once", func(t *testing.T) {
		request := buildForTest(t, "campaigns update", Values{"id": "101", "name": "Renamed", "status": "paused"}, Globals{Currency: "USD"}, contractInput{})
		want := map[string]any{"campaign": map[string]any{"name": "Renamed", "status": "PAUSED"}}
		if !reflect.DeepEqual(request.Body, want) {
			t.Errorf("Body = %#v, want %#v", request.Body, want)
		}
	})
	t.Run("complete JSON request passes through", func(t *testing.T) {
		body := map[string]any{"campaign": map[string]any{"name": "Renamed"}, "clearGeoTargetingOnCountryOrRegionChange": true}
		request := buildForTest(t, "campaigns update", Values{"id": "101", "from-json": "update"}, Globals{}, contractInput{"update": body})
		if !reflect.DeepEqual(request.Body, body) {
			t.Errorf("Body = %#v, want unchanged %#v", request.Body, body)
		}
	})
}

func TestSelectors(t *testing.T) {
	t.Run("shorthand condition and pagination stay in body", func(t *testing.T) {
		request := buildForTest(t, "campaigns find", Values{"field": "status", "op": "equals", "values": "ENABLED, PAUSED", "limit": "30", "offset": "10", "all": "true"}, Globals{}, contractInput{})
		want := map[string]any{
			"conditions": []any{map[string]any{"field": "status", "operator": "EQUALS", "values": []string{"ENABLED", "PAUSED"}}},
			"pagination": map[string]any{"limit": 30, "offset": 10},
		}
		if !reflect.DeepEqual(request.Body, want) {
			t.Errorf("Body = %#v, want %#v", request.Body, want)
		}
		if len(request.Query) != 0 {
			t.Errorf("Query = %v, selector flags must not leak into the URL", request.Query)
		}
		if !request.AllPages {
			t.Error("AllPages = false, want true")
		}
	})
	t.Run("alias conflict", func(t *testing.T) {
		_, err := buildForTestError("campaigns find", Values{"selector-json": "selector", "selector": "selector"}, Globals{}, contractInput{"selector": map[string]any{}})
		if err == nil || !strings.Contains(err.Error(), "either --selector-json or --selector") {
			t.Fatalf("Build() error = %v, want alias conflict", err)
		}
	})
	t.Run("all shorthand fields are required together", func(t *testing.T) {
		_, err := buildForTestError("campaigns find", Values{"field": "status", "op": "equals"}, Globals{}, contractInput{})
		if err == nil || !strings.Contains(err.Error(), "must be used together") {
			t.Fatalf("Build() error = %v, want shorthand completeness error", err)
		}
	})
	t.Run("keyword alias adds string ad group condition", func(t *testing.T) {
		request := buildForTest(t, "keywords find", Values{"campaign-id": "101", "adgroup-id": "202"}, Globals{}, contractInput{})
		want := []any{map[string]any{"field": "adGroupId", "operator": "EQUALS", "values": []string{"202"}}}
		if got := request.Body.(map[string]any)["conditions"]; !reflect.DeepEqual(got, want) {
			t.Errorf("conditions = %#v, want %#v", got, want)
		}
	})
}

func TestGeoLookupBody(t *testing.T) {
	request := buildForTest(t, "geo get", Values{"geo-id": "geo/toronto", "entity": "locality", "limit": "20", "offset": "40", "all": "true"}, Globals{}, contractInput{})
	want := []any{map[string]any{"id": "geo/toronto", "entity": "Locality"}}
	if !reflect.DeepEqual(request.Body, want) {
		t.Errorf("Body = %#v, want %#v", request.Body, want)
	}
	if got := request.Query.Get("entity"); got != "" {
		t.Errorf("entity query = %q, want empty because entity belongs only in the body", got)
	}
	if request.Query.Get("limit") != "20" || request.Query.Get("offset") != "40" || !request.AllPages {
		t.Errorf("pagination = query %v all=%v, want limit=20 offset=40 all=true", request.Query, request.AllPages)
	}
}

func TestReportsBuildAllOfficialBodies(t *testing.T) {
	for _, key := range []string{"reports campaigns", "reports adgroups", "reports keywords", "reports searchterms", "reports ads"} {
		t.Run(key, func(t *testing.T) {
			values := Values{"start-time": "2026-07-01", "end-time": "2026-07-31", "granularity": "daily", "group-by": "countryOrRegion, deviceClass", "time-zone": "ortz", "return-records-with-no-metrics": "true", "limit": "50", "offset": "10", "all": "true"}
			if key != "reports campaigns" {
				values["campaign-id"] = "101"
			}
			request := buildForTest(t, key, values, Globals{}, contractInput{})
			want := map[string]any{
				"startTime": "2026-07-01", "endTime": "2026-07-31", "granularity": "DAILY",
				"returnRowTotals": false, "returnGrandTotals": false, "returnRecordsWithNoMetrics": true,
				"timeZone": "ORTZ", "groupBy": []string{"countryOrRegion", "deviceClass"},
				"selector": map[string]any{
					"orderBy":    []any{map[string]any{"field": "localSpend", "sortOrder": "DESCENDING"}},
					"pagination": map[string]any{"limit": 50, "offset": 10},
				},
			}
			if !reflect.DeepEqual(request.Body, want) {
				t.Errorf("Body = %#v\nwant = %#v", request.Body, want)
			}
			if len(request.Query) != 0 || !request.AllPages {
				t.Errorf("query/all = %v/%v, report pagination belongs in selector and all must be true", request.Query, request.AllPages)
			}
		})
	}
}

func TestReportValidationAndJSONPassThrough(t *testing.T) {
	tests := []struct {
		name   string
		values Values
		input  contractInput
		want   string
	}{
		{name: "requires start", values: Values{"end-time": "2026-07-31"}, want: "--start-time is required"},
		{name: "valid dates", values: Values{"start-time": "07/01/2026", "end-time": "2026-07-31"}, want: "YYYY-MM-DD"},
		{name: "ordered dates", values: Values{"start-time": "2026-08-01", "end-time": "2026-07-31"}, want: "cannot be after"},
		{name: "granularity", values: Values{"start-time": "2026-07-01", "end-time": "2026-07-31", "granularity": "YEARLY"}, want: "--granularity must be one of"},
		{name: "time zone", values: Values{"start-time": "2026-07-01", "end-time": "2026-07-31", "time-zone": "America/Toronto"}, want: "--time-zone must be one of"},
		{name: "selector alias conflict", values: Values{"start-time": "2026-07-01", "end-time": "2026-07-31", "selector-json": "selector", "selector": "selector"}, input: contractInput{"selector": map[string]any{}}, want: "either --selector-json or --selector"},
		{name: "empty complete request", values: Values{"from-json": "empty"}, input: contractInput{"empty": map[string]any{}}, want: "selector object"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildForTestError("reports campaigns", test.values, Globals{}, test.input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Build() error = %v, want containing %q", err, test.want)
			}
		})
	}

	body := map[string]any{"startTime": "2026-07-01", "endTime": "2026-07-31", "selector": map[string]any{"conditions": []any{}}}
	request := buildForTest(t, "reports campaigns", Values{"from-json": "report"}, Globals{}, contractInput{"report": body})
	if !reflect.DeepEqual(request.Body, body) {
		t.Errorf("complete report request = %#v, want unchanged %#v", request.Body, body)
	}

	t.Run("totals without granularity", func(t *testing.T) {
		request := buildForTest(t, "reports campaigns", Values{"start-time": "2026-07-01", "end-time": "2026-07-31", "granularity": "NONE", "return-row-totals": "true", "return-grand-totals": "true"}, Globals{}, contractInput{})
		body := request.Body.(map[string]any)
		if _, exists := body["granularity"]; exists {
			t.Errorf("granularity = %#v, want omitted for NONE", body["granularity"])
		}
		if body["returnRowTotals"] != true || body["returnGrandTotals"] != true {
			t.Errorf("totals = row %#v grand %#v, want true/true", body["returnRowTotals"], body["returnGrandTotals"])
		}
	})
	t.Run("totals conflict with granularity", func(t *testing.T) {
		_, err := buildForTestError("reports campaigns", Values{"start-time": "2026-07-01", "end-time": "2026-07-31", "granularity": "DAILY", "return-row-totals": "true"}, Globals{}, contractInput{})
		if err == nil || !strings.Contains(err.Error(), "totals cannot be combined") {
			t.Fatalf("Build() error = %v, want totals/granularity conflict", err)
		}
	})
}

func TestImpressionShareValidation(t *testing.T) {
	t.Run("daily dates are optional", func(t *testing.T) {
		request := buildForTest(t, "impression-share create", Values{"name": "Daily share"}, Globals{}, contractInput{})
		want := map[string]any{"name": "Daily share", "granularity": "DAILY"}
		if !reflect.DeepEqual(request.Body, want) {
			t.Errorf("Body = %#v, want %#v", request.Body, want)
		}
	})
	t.Run("daily", func(t *testing.T) {
		request := buildForTest(t, "impression-share create", Values{"name": "July share", "start-time": "2026-07-01", "end-time": "2026-07-31", "selector": "selector"}, Globals{}, contractInput{"selector": map[string]any{"conditions": []any{}}})
		want := map[string]any{"name": "July share", "granularity": "DAILY", "startTime": "2026-07-01", "endTime": "2026-07-31", "selector": map[string]any{"conditions": []any{}}}
		if !reflect.DeepEqual(request.Body, want) {
			t.Errorf("Body = %#v, want %#v", request.Body, want)
		}
	})
	t.Run("weekly", func(t *testing.T) {
		request := buildForTest(t, "impression-share create", Values{"name": "Weekly share", "granularity": "weekly", "date-range": "last_2_weeks"}, Globals{}, contractInput{})
		want := map[string]any{"name": "Weekly share", "granularity": "WEEKLY", "dateRange": "LAST_2_WEEKS"}
		if !reflect.DeepEqual(request.Body, want) {
			t.Errorf("Body = %#v, want %#v", request.Body, want)
		}
	})
	t.Run("weekly defaults to last week", func(t *testing.T) {
		request := buildForTest(t, "impression-share create", Values{"name": "Weekly share", "granularity": "weekly"}, Globals{}, contractInput{})
		want := map[string]any{"name": "Weekly share", "granularity": "WEEKLY", "dateRange": "LAST_WEEK"}
		if !reflect.DeepEqual(request.Body, want) {
			t.Errorf("Body = %#v, want %#v", request.Body, want)
		}
	})
	tests := []struct {
		name   string
		values Values
		want   string
	}{
		{name: "name required", values: Values{"start-time": "2026-07-01", "end-time": "2026-07-31"}, want: "--name is required"},
		{name: "name length", values: Values{"name": strings.Repeat("x", 51), "start-time": "2026-07-01", "end-time": "2026-07-31"}, want: "50"},
		{name: "daily both dates", values: Values{"name": "x", "start-time": "2026-07-01"}, want: "must be used together"},
		{name: "date lower bound", values: Values{"name": "x", "start-time": "2020-04-11", "end-time": "2020-04-12"}, want: "2020-04-12"},
		{name: "ordered dates", values: Values{"name": "x", "start-time": "2026-08-01", "end-time": "2026-07-31"}, want: "cannot be after"},
		{name: "daily max range", values: Values{"name": "x", "start-time": "2026-07-01", "end-time": "2026-08-01"}, want: "30 days"},
		{name: "weekly forbids dates", values: Values{"name": "x", "granularity": "WEEKLY", "date-range": "LAST_WEEK", "start-time": "2026-07-01"}, want: "instead of start and end"},
		{name: "weekly range enum", values: Values{"name": "x", "granularity": "WEEKLY", "date-range": "LAST_YEAR"}, want: "--date-range must be one of"},
		{name: "selector alias conflict", values: Values{"name": "x", "start-time": "2026-07-01", "end-time": "2026-07-31", "selector-json": "selector", "selector": "selector"}, want: "either --selector-json or --selector"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildForTestError("impression-share create", test.values, Globals{}, contractInput{"selector": map[string]any{}})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Build() error = %v, want containing %q", err, test.want)
			}
		})
	}

	t.Run("complete JSON request must be nonempty object", func(t *testing.T) {
		invalidBodies := contractInput{"array": []any{map[string]any{}}, "empty": map[string]any{}}
		for source, body := range invalidBodies {
			_, err := buildForTestError("impression-share create", Values{"from-json": source}, Globals{}, contractInput{source: body})
			if err == nil {
				t.Errorf("Build(%s) succeeded, want object validation error", source)
			}
		}
	})
}

func TestMutationAndDeleteMetadata(t *testing.T) {
	mutations := map[string]bool{
		"campaigns create": true, "campaigns update": true, "campaigns delete": true,
		"budgetorders create": true, "budgetorders update": true,
		"adgroups create": true, "adgroups update": true, "adgroups delete": true,
		"keywords create": true, "keywords update": true, "keywords delete": true, "keywords delete-one": true,
		"negatives campaign-create": true, "negatives campaign-update": true, "negatives campaign-delete": true,
		"negatives adgroup-create": true, "negatives adgroup-update": true, "negatives adgroup-delete": true,
		"ads create": true, "ads update": true, "ads delete": true,
		"creatives create": true, "impression-share create": true,
	}
	deletes := map[string]bool{
		"campaigns delete": true, "adgroups delete": true, "keywords delete": true, "keywords delete-one": true,
		"negatives campaign-delete": true, "negatives adgroup-delete": true, "ads delete": true,
	}
	for _, command := range Registry() {
		wantMutation := mutations[command.Key()]
		if command.Mutation != wantMutation {
			t.Errorf("%s Mutation = %v, want %v", command.Key(), command.Mutation, wantMutation)
		}
		if hasFlag(command, "dry-run") != wantMutation {
			t.Errorf("%s --dry-run present = %v, want %v", command.Key(), hasFlag(command, "dry-run"), wantMutation)
		}
		wantDelete := deletes[command.Key()]
		if command.Delete != wantDelete {
			t.Errorf("%s Delete = %v, want %v", command.Key(), command.Delete, wantDelete)
		}
		if hasFlag(command, "yes") != wantDelete {
			t.Errorf("%s --yes present = %v, want %v", command.Key(), hasFlag(command, "yes"), wantDelete)
		}
		if wantDelete && command.SuccessText == "" {
			t.Errorf("%s has no deletion success text", command.Key())
		}
	}
}

func TestPathExpansionEscapesOpaqueProductPageIDs(t *testing.T) {
	request := buildForTest(t, "product-pages get", Values{"adam-id": "303", "id": "launch/US ?"}, Globals{}, contractInput{})
	if want := "/apps/303/product-pages/launch%2FUS%20%3F"; request.Path != want {
		t.Errorf("Path = %q, want %q", request.Path, want)
	}
	for _, test := range []struct {
		key    string
		values Values
	}{{key: "creatives get", values: Values{"id": "../707"}}, {key: "apps details", values: Values{"adam-id": "../303"}}, {key: "campaigns get", values: Values{"id": "0"}}} {
		_, err := buildForTestError(test.key, test.values, Globals{}, contractInput{})
		if err == nil {
			t.Errorf("%s accepted invalid numeric path values %v", test.key, test.values)
		}
	}
}

func hasFlag(command Command, name string) bool {
	for _, flag := range command.Flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}

func buildForTest(t *testing.T, key string, values Values, globals Globals, input contractInput) Request {
	t.Helper()
	request, err := buildForTestError(key, values, globals, input)
	if err != nil {
		t.Fatalf("%s Build() error = %v", key, err)
	}
	return request
}

func buildForTestError(key string, values Values, globals Globals, input contractInput) (Request, error) {
	parts := strings.SplitN(key, " ", 2)
	command, ok := Lookup(parts[0], parts[1])
	if !ok {
		return Request{}, &testError{"command not registered: " + key}
	}
	values = cloneValues(values)
	if err := ValidateFlags(command, values); err != nil {
		return Request{}, err
	}
	return command.Build(values, globals, input)
}

type testError struct{ message string }

func (err *testError) Error() string { return err.message }
