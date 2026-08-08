package commands

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type contractInput map[string]any

func (input contractInput) Read(source string) (any, error) {
	value, ok := input[source]
	if !ok {
		return nil, fmt.Errorf("unexpected input source %q", source)
	}
	return value, nil
}

type routeContract struct {
	key    string
	method string
	path   string
	values Values
}

var officialRouteContracts = []routeContract{
	{key: "acls list", method: http.MethodGet, path: "/acls"},
	{key: "acls me", method: http.MethodGet, path: "/me"},

	{key: "apps search", method: http.MethodGet, path: "/search/apps", values: Values{"query": "notes"}},
	{key: "apps eligibility", method: http.MethodPost, path: "/apps/303/eligibilities/find", values: Values{"adam-id": "303"}},
	{key: "apps details", method: http.MethodGet, path: "/apps/303", values: Values{"adam-id": "303"}},
	{key: "apps localized", method: http.MethodGet, path: "/apps/303/locale-details", values: Values{"adam-id": "303"}},

	{key: "campaigns create", method: http.MethodPost, path: "/campaigns", values: Values{"from-json": "object"}},
	{key: "campaigns find", method: http.MethodPost, path: "/campaigns/find"},
	{key: "campaigns get", method: http.MethodGet, path: "/campaigns/101", values: Values{"id": "101"}},
	{key: "campaigns list", method: http.MethodGet, path: "/campaigns"},
	{key: "campaigns update", method: http.MethodPut, path: "/campaigns/101", values: Values{"id": "101", "from-json": "object"}},
	{key: "campaigns delete", method: http.MethodDelete, path: "/campaigns/101", values: Values{"id": "101"}},

	{key: "budgetorders create", method: http.MethodPost, path: "/budgetorders", values: Values{"from-json": "object"}},
	{key: "budgetorders update", method: http.MethodPut, path: "/budgetorders/404", values: Values{"id": "404", "from-json": "object"}},
	{key: "budgetorders get", method: http.MethodGet, path: "/budgetorders/404", values: Values{"id": "404"}},
	{key: "budgetorders list", method: http.MethodGet, path: "/budgetorders"},

	{key: "adgroups create", method: http.MethodPost, path: "/campaigns/101/adgroups", values: Values{"campaign-id": "101", "from-json": "object"}},
	{key: "adgroups find", method: http.MethodPost, path: "/campaigns/101/adgroups/find", values: Values{"campaign-id": "101"}},
	{key: "adgroups find-all", method: http.MethodPost, path: "/adgroups/find"},
	{key: "adgroups get", method: http.MethodGet, path: "/campaigns/101/adgroups/202", values: Values{"campaign-id": "101", "id": "202"}},
	{key: "adgroups list", method: http.MethodGet, path: "/campaigns/101/adgroups", values: Values{"campaign-id": "101"}},
	{key: "adgroups update", method: http.MethodPut, path: "/campaigns/101/adgroups/202", values: Values{"campaign-id": "101", "id": "202", "from-json": "object"}},
	{key: "adgroups delete", method: http.MethodDelete, path: "/campaigns/101/adgroups/202", values: Values{"campaign-id": "101", "id": "202"}},

	{key: "keywords create", method: http.MethodPost, path: "/campaigns/101/adgroups/202/targetingkeywords/bulk", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
	{key: "keywords find", method: http.MethodPost, path: "/campaigns/101/adgroups/targetingkeywords/find", values: Values{"campaign-id": "101", "adgroup-id": "202"}},
	{key: "keywords find-campaign", method: http.MethodPost, path: "/campaigns/101/adgroups/targetingkeywords/find", values: Values{"campaign-id": "101"}},
	{key: "keywords get", method: http.MethodGet, path: "/campaigns/101/adgroups/202/targetingkeywords/505", values: Values{"campaign-id": "101", "adgroup-id": "202", "id": "505"}},
	{key: "keywords list", method: http.MethodGet, path: "/campaigns/101/adgroups/202/targetingkeywords", values: Values{"campaign-id": "101", "adgroup-id": "202"}},
	{key: "keywords update", method: http.MethodPut, path: "/campaigns/101/adgroups/202/targetingkeywords/bulk", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
	{key: "keywords delete", method: http.MethodPost, path: "/campaigns/101/adgroups/202/targetingkeywords/delete/bulk", values: Values{"campaign-id": "101", "adgroup-id": "202", "ids": "505"}},
	{key: "keywords delete-one", method: http.MethodDelete, path: "/campaigns/101/adgroups/202/targetingkeywords/505", values: Values{"campaign-id": "101", "adgroup-id": "202", "id": "505"}},

	{key: "negatives campaign-create", method: http.MethodPost, path: "/campaigns/101/negativekeywords/bulk", values: Values{"campaign-id": "101", "from-json": "array"}},
	{key: "negatives campaign-find", method: http.MethodPost, path: "/campaigns/101/negativekeywords/find", values: Values{"campaign-id": "101"}},
	{key: "negatives campaign-get", method: http.MethodGet, path: "/campaigns/101/negativekeywords/505", values: Values{"campaign-id": "101", "id": "505"}},
	{key: "negatives campaign-list", method: http.MethodGet, path: "/campaigns/101/negativekeywords", values: Values{"campaign-id": "101"}},
	{key: "negatives campaign-update", method: http.MethodPut, path: "/campaigns/101/negativekeywords/bulk", values: Values{"campaign-id": "101", "from-json": "array"}},
	{key: "negatives campaign-delete", method: http.MethodPost, path: "/campaigns/101/negativekeywords/delete/bulk", values: Values{"campaign-id": "101", "ids": "505"}},
	{key: "negatives adgroup-create", method: http.MethodPost, path: "/campaigns/101/adgroups/202/negativekeywords/bulk", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
	{key: "negatives adgroup-find", method: http.MethodPost, path: "/campaigns/101/adgroups/negativekeywords/find", values: Values{"campaign-id": "101"}},
	{key: "negatives adgroup-get", method: http.MethodGet, path: "/campaigns/101/adgroups/202/negativekeywords/505", values: Values{"campaign-id": "101", "adgroup-id": "202", "id": "505"}},
	{key: "negatives adgroup-list", method: http.MethodGet, path: "/campaigns/101/adgroups/202/negativekeywords", values: Values{"campaign-id": "101", "adgroup-id": "202"}},
	{key: "negatives adgroup-update", method: http.MethodPut, path: "/campaigns/101/adgroups/202/negativekeywords/bulk", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "array"}},
	{key: "negatives adgroup-delete", method: http.MethodPost, path: "/campaigns/101/adgroups/202/negativekeywords/delete/bulk", values: Values{"campaign-id": "101", "adgroup-id": "202", "ids": "505"}},

	{key: "ads create", method: http.MethodPost, path: "/campaigns/101/adgroups/202/ads", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "object"}},
	{key: "ads find", method: http.MethodPost, path: "/campaigns/101/ads/find", values: Values{"campaign-id": "101"}},
	{key: "ads find-all", method: http.MethodPost, path: "/ads/find"},
	{key: "ads get", method: http.MethodGet, path: "/campaigns/101/adgroups/202/ads/606", values: Values{"campaign-id": "101", "adgroup-id": "202", "id": "606"}},
	{key: "ads list", method: http.MethodGet, path: "/campaigns/101/adgroups/202/ads", values: Values{"campaign-id": "101", "adgroup-id": "202"}},
	{key: "ads update", method: http.MethodPut, path: "/campaigns/101/adgroups/202/ads/606", values: Values{"campaign-id": "101", "adgroup-id": "202", "id": "606", "from-json": "object"}},
	{key: "ads delete", method: http.MethodDelete, path: "/campaigns/101/adgroups/202/ads/606", values: Values{"campaign-id": "101", "adgroup-id": "202", "id": "606"}},

	{key: "creatives create", method: http.MethodPost, path: "/creatives", values: Values{"from-json": "object"}},
	{key: "creatives find", method: http.MethodPost, path: "/creatives/find"},
	{key: "creatives get", method: http.MethodGet, path: "/creatives/707", values: Values{"id": "707"}},
	{key: "creatives list", method: http.MethodGet, path: "/creatives"},

	{key: "product-pages list", method: http.MethodGet, path: "/apps/303/product-pages", values: Values{"adam-id": "303"}},
	{key: "product-pages get", method: http.MethodGet, path: "/apps/303/product-pages/page-one", values: Values{"adam-id": "303", "id": "page-one"}},
	{key: "product-pages locales", method: http.MethodGet, path: "/apps/303/product-pages/page-one/locale-details", values: Values{"adam-id": "303", "id": "page-one"}},
	{key: "product-pages countries", method: http.MethodGet, path: "/countries-or-regions"},
	{key: "product-pages device-sizes", method: http.MethodGet, path: "/creativeappmappings/devices"},

	{key: "ad-rejections find", method: http.MethodPost, path: "/product-page-reasons/find"},
	{key: "ad-rejections get", method: http.MethodGet, path: "/product-page-reasons/808", values: Values{"id": "808"}},
	{key: "ad-rejections find-assets", method: http.MethodPost, path: "/apps/303/assets/find", values: Values{"adam-id": "303"}},

	{key: "geo search", method: http.MethodGet, path: "/search/geo", values: Values{"query": "Toronto"}},
	{key: "geo get", method: http.MethodPost, path: "/search/geo", values: Values{"from-json": "array"}},

	{key: "reports campaigns", method: http.MethodPost, path: "/reports/campaigns", values: Values{"from-json": "object"}},
	{key: "reports adgroups", method: http.MethodPost, path: "/reports/campaigns/101/adgroups", values: Values{"campaign-id": "101", "from-json": "object"}},
	{key: "reports keywords", method: http.MethodPost, path: "/reports/campaigns/101/keywords", values: Values{"campaign-id": "101", "from-json": "object"}},
	{key: "reports keywords", method: http.MethodPost, path: "/reports/campaigns/101/adgroups/202/keywords", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "object"}},
	{key: "reports searchterms", method: http.MethodPost, path: "/reports/campaigns/101/searchterms", values: Values{"campaign-id": "101", "from-json": "object"}},
	{key: "reports searchterms", method: http.MethodPost, path: "/reports/campaigns/101/adgroups/202/searchterms", values: Values{"campaign-id": "101", "adgroup-id": "202", "from-json": "object"}},
	{key: "reports ads", method: http.MethodPost, path: "/reports/campaigns/101/ads", values: Values{"campaign-id": "101", "from-json": "object"}},

	{key: "impression-share create", method: http.MethodPost, path: "/custom-reports", values: Values{"from-json": "object"}},
	{key: "impression-share get", method: http.MethodGet, path: "/custom-reports/909", values: Values{"id": "909"}},
	{key: "impression-share list", method: http.MethodGet, path: "/custom-reports"},
}

func TestRegistryMatchesAllOfficialV5Routes(t *testing.T) {
	input := contractInput{
		"object": map[string]any{"sentinel": true, "selector": map[string]any{}},
		"array":  []any{map[string]any{"sentinel": true}},
	}
	coveredCommands := map[string]int{}
	uniqueRoutes := map[string]bool{}
	for _, test := range officialRouteContracts {
		t.Run(strings.ReplaceAll(test.key+" "+test.path, "/", "_"), func(t *testing.T) {
			command, ok := Lookup(strings.SplitN(test.key, " ", 2)[0], strings.SplitN(test.key, " ", 2)[1])
			if !ok {
				t.Fatalf("official command %q is not registered", test.key)
			}
			values := cloneValues(test.values)
			if err := ValidateFlags(command, values); err != nil {
				t.Fatalf("ValidateFlags() error = %v", err)
			}
			request, err := command.Build(values, Globals{Currency: "USD"}, input)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if request.Method != test.method || request.Path != test.path {
				t.Errorf("route = %s %s, want %s %s", request.Method, request.Path, test.method, test.path)
			}
			if request.Operation != strings.Replace(test.key, " ", ".", 1) {
				t.Errorf("Operation = %q, want %q", request.Operation, strings.Replace(test.key, " ", ".", 1))
			}
			coveredCommands[test.key]++
			uniqueRoutes[test.method+" "+test.path] = true
		})
	}

	if got := len(uniqueRoutes); got != 73 {
		t.Errorf("unique official routes = %d, want 73", got)
	}
	var registered []string
	for _, command := range Registry() {
		registered = append(registered, command.Key())
		if coveredCommands[command.Key()] == 0 {
			t.Errorf("registered command %q is not an active official v5 operation", command.Key())
		}
	}
	var covered []string
	for key := range coveredCommands {
		covered = append(covered, key)
	}
	sort.Strings(registered)
	sort.Strings(covered)
	if !reflect.DeepEqual(registered, covered) {
		t.Errorf("registry keys do not exactly match official commands\nregistered: %v\nofficial: %v", registered, covered)
	}
	if got := len(registered); got != 72 {
		t.Errorf("registered commands = %d, want 72 commands representing 73 official routes", got)
	}
}

func TestEveryRequiredFlagIsEnforced(t *testing.T) {
	for _, test := range officialRouteContracts {
		command, _ := Lookup(strings.SplitN(test.key, " ", 2)[0], strings.SplitN(test.key, " ", 2)[1])
		for _, flag := range command.Flags {
			if !flag.Required {
				continue
			}
			t.Run(test.key+"/--"+flag.Name, func(t *testing.T) {
				values := cloneValues(test.values)
				delete(values, flag.Name)
				err := ValidateFlags(command, values)
				if err == nil || !strings.Contains(err.Error(), "--"+flag.Name+" is required") {
					t.Fatalf("ValidateFlags() error = %v, want required flag error", err)
				}
			})
		}
	}
}

func cloneValues(values Values) Values {
	clone := make(Values, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}
