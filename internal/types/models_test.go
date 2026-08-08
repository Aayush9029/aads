package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCampaignCreateUsesV5BudgetAndBiddingFields(t *testing.T) {
	request := CampaignCreate{
		AdamID:             535500008,
		AdChannelType:      "DISPLAY",
		BillingEvent:       "TAPS",
		BiddingStrategy:    "MAX_CONVERSIONS",
		CountriesOrRegions: []string{"US", "CA"},
		DailyBudgetAmount:  &Money{Amount: "250.00", Currency: "USD"},
		Name:               "Trip Trek",
		SupplySources:      []string{"APPSTORE_SEARCH_RESULTS"},
		TargetCPA:          &Money{Amount: "10", Currency: "USD"},
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"adamId", "dailyBudgetAmount", "countriesOrRegions", "biddingStrategy", "targetCpa"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("campaign JSON omitted %q: %s", key, encoded)
		}
	}
	if _, ok := fields["budgetAmount"]; ok {
		t.Fatalf("campaign JSON used removed budgetAmount field: %s", encoded)
	}
}

func TestCampaignUpdateEnvelope(t *testing.T) {
	request := UpdateCampaignRequest{
		Campaign: &CampaignUpdate{
			Name:              "Renamed",
			DailyBudgetAmount: &Money{Amount: "75", Currency: "CAD"},
		},
		ClearGeoTargetingOnCountryOrRegionChange: true,
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"campaign":{"dailyBudgetAmount":{"amount":"75","currency":"CAD"},"name":"Renamed"},"clearGeoTargetingOnCountryOrRegionChange":true}`
	assertJSONEqual(t, encoded, []byte(want))
}

func TestBudgetOrderCreateEnvelope(t *testing.T) {
	request := BudgetOrderCreate{
		OrgIDs: []int64{40669820},
		BO: &BudgetOrderCreateDetail{
			Name:              "Q4",
			Budget:            &Money{Amount: "300", Currency: "USD"},
			PrimaryBuyerEmail: "buyer@example.com",
			SupplySources:     []string{"APPSTORE_SEARCH_RESULTS"},
		},
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
		"orgIds":[40669820],
		"bo":{
			"name":"Q4",
			"budget":{"amount":"300","currency":"USD"},
			"primaryBuyerEmail":"buyer@example.com",
			"supplySources":["APPSTORE_SEARCH_RESULTS"]
		}
	}`
	assertJSONEqual(t, encoded, []byte(want))
}

func TestAdGroupTargetingDimensions(t *testing.T) {
	input := []byte(`{
		"id":542370642,
		"automatedKeywordsRequired":true,
		"biddingStrategy":"MAX_CONVERSIONS",
		"pricingModel":"CPC",
		"targetingDimensions":{
			"age":{"included":[{"minAge":20,"maxAge":25}]},
			"deviceClass":{"included":["IPAD","IPHONE"]},
			"appCategories":{"included":[6001],"excluded":[6002]},
			"appDownloaders":{"included":["123"],"excluded":["456"]}
		}
	}`)
	var adGroup AdGroup
	if err := json.Unmarshal(input, &adGroup); err != nil {
		t.Fatal(err)
	}
	if adGroup.TargetingDimensions == nil {
		t.Fatal("targeting dimensions are nil")
	}
	if got := adGroup.TargetingDimensions.Age.Included[0].MaxAge; got != 25 {
		t.Fatalf("max age = %d", got)
	}
	if got := adGroup.TargetingDimensions.AppCategories.Excluded; !reflect.DeepEqual(got, []int64{6002}) {
		t.Fatalf("excluded categories = %#v", got)
	}
}

func TestKeywordAndNegativeKeywordWireShapes(t *testing.T) {
	keyword := Keyword{
		BidAmount: &Money{Amount: "1.25", Currency: "USD"},
		MatchType: "EXACT",
		Status:    "ACTIVE",
		Text:      "trip planner",
	}
	negative := NegativeKeyword{MatchType: "BROAD", Status: "PAUSED", Text: "free"}

	keywordJSON, err := json.Marshal(keyword)
	if err != nil {
		t.Fatal(err)
	}
	negativeJSON, err := json.Marshal(negative)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, keywordJSON, []byte(`{"bidAmount":{"amount":"1.25","currency":"USD"},"matchType":"EXACT","status":"ACTIVE","text":"trip planner"}`))
	assertJSONEqual(t, negativeJSON, []byte(`{"matchType":"BROAD","status":"PAUSED","text":"free"}`))
}

func TestMediaAndProductPagePayloads(t *testing.T) {
	input := []byte(`{
		"adamId":284815942,
		"appName":"Trip Trek",
		"artistName":"Trip Trek LLC",
		"availableStorefronts":["US","CA"],
		"deviceClasses":["IPHONE","IPAD"],
		"iconPictureUrl":"https://example.com/icon.png",
		"isPreOrder":false,
		"primaryLanguage":"en-US"
	}`)
	var detail MediaDetail
	if err := json.Unmarshal(input, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.ArtistName != "Trip Trek LLC" || detail.IconPictureURL == "" || len(detail.DeviceClasses) != 2 {
		t.Fatalf("media detail = %#v", detail)
	}

	localeInput := []byte(`{
		"productPageId":"cpp-id",
		"languageCode":"en-US",
		"appPreviewDeviceWithAssets":{
			"iphone67":{"screenshots":[{"assetGenId":"asset-1","assetType":"SCREENSHOT","sourceWidth":1290}]}
		}
	}`)
	var locale ProductPageLocaleDetail
	if err := json.Unmarshal(localeInput, &locale); err != nil {
		t.Fatal(err)
	}
	if got := locale.AppPreviewDeviceWithAssets["iphone67"].Screenshots[0].AssetGenID; got != "asset-1" {
		t.Fatalf("asset id = %q", got)
	}
}

func TestAdCreativeRejectionAndGeoPayloads(t *testing.T) {
	var reason ProductPageReason
	if err := json.Unmarshal([]byte(`{"id":7,"adamId":123,"reasonLevel":"ASSET","reasonCode":"ASSET_REJECTED","productPageId":null}`), &reason); err != nil {
		t.Fatal(err)
	}
	if reason.ID != 7 || reason.ReasonLevel != "ASSET" {
		t.Fatalf("reason = %#v", reason)
	}

	var geo SearchEntity
	if err := json.Unmarshal([]byte(`{"id":"US|CA|SAN_FRANCISCO","entity":"Locality","displayName":"San Francisco","adminArea":"California","countryOrRegion":"US"}`), &geo); err != nil {
		t.Fatal(err)
	}
	if geo.AdminArea != "California" || geo.CountryOrRegion != "US" {
		t.Fatalf("geo = %#v", geo)
	}
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode got JSON: %v", err)
	}
	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("decode want JSON: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSON mismatch\n got: %s\nwant: %s", got, want)
	}
}
