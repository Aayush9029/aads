package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	decimalMoney    = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]{1,2})?$`)
	appleCurrencies = stringSet("AUD CAD EUR GBP JPY MXN NZD RUB USD")
	isoCountryCodes = stringSet("AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW")
)

func stringSet(values string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, value := range strings.Fields(values) {
		result[value] = struct{}{}
	}
	return result
}

var (
	flagID         = Flag{Name: "id", Description: "Resource ID", Required: true}
	flagCampaignID = Flag{Name: "campaign-id", Description: "Campaign ID", Required: true}
	flagAdGroupID  = Flag{Name: "adgroup-id", Description: "Ad group ID", Required: true}
	flagAdamID     = Flag{Name: "adam-id", Description: "App Adam ID", Required: true}
	flagFromJSON   = Flag{Name: "from-json", Description: "JSON value, @file, or @-"}
	flagLimit      = Flag{Name: "limit", Description: "Maximum results"}
	flagOffset     = Flag{Name: "offset", Description: "Starting offset"}
	flagAll        = Flag{Name: "all", Description: "Fetch every page", Boolean: true}
	selectorFlags  = []Flag{
		{Name: "selector-json", Description: "Selector JSON, @file, or @-"},
		{Name: "selector", Description: "Alias for --selector-json"},
		{Name: "field", Description: "Filter field"},
		{Name: "op", Description: "Filter operator"},
		{Name: "values", Description: "Comma-separated filter values"},
		{Name: "limit", Description: "Maximum results", Default: "20"},
		flagOffset,
		flagAll,
	}
)

func command(group, name, summary, method, path string, flags []Flag, body func(Values, Globals, InputReader) (any, error)) Command {
	safePost := method == http.MethodPost && (strings.Contains(name, "find") || group == "reports" || group == "geo" && name == "get" || group == "apps" && name == "eligibility")
	mutation := method == http.MethodPut || method == http.MethodDelete || method == http.MethodPost && !safePost
	return Command{
		Group: group, Name: name, Summary: summary, Flags: flags, Mutation: mutation,
		Delete: method == http.MethodDelete,
		Build: func(values Values, globals Globals, input InputReader) (Request, error) {
			resolvedPath, err := expandPath(path, values)
			if err != nil {
				return Request{}, err
			}
			query, err := buildQuery(values, globals)
			if err != nil {
				return Request{}, err
			}
			var payload any
			if body != nil {
				payload, err = body(values, globals, input)
				if err != nil {
					return Request{}, err
				}
			}
			return Request{
				Method: method, Path: resolvedPath, Query: query, Body: payload,
				Operation: group + "." + name, OrgContext: group != "acls",
				Mutation: mutation, AllPages: values.Bool("all"),
			}, nil
		},
	}
}

func validatePaginationValues(values Values) error {
	if raw := values.Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 1000 {
			return fmt.Errorf("--limit must be between 1 and 1000")
		}
	}
	if raw := values.Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return fmt.Errorf("--offset must be a non-negative integer")
		}
	}
	return nil
}

func expandPath(path string, values Values) (string, error) {
	replacements := []struct{ token, flag string }{
		{"{campaign-id}", "campaign-id"}, {"{adgroup-id}", "adgroup-id"},
		{"{adam-id}", "adam-id"}, {"{id}", "id"}, {"{geo-id}", "geo-id"},
	}
	for _, replacement := range replacements {
		if !strings.Contains(path, replacement.token) {
			continue
		}
		value := strings.TrimSpace(values.Get(replacement.flag))
		if value == "" {
			return "", fmt.Errorf("--%s is required", replacement.flag)
		}
		mustBeNumeric := replacement.flag != "geo-id" && (replacement.flag != "id" || !strings.Contains(path, "product-pages"))
		if mustBeNumeric {
			if err := positiveInteger(replacement.flag, value); err != nil {
				return "", err
			}
		}
		path = strings.ReplaceAll(path, replacement.token, url.PathEscape(value))
	}
	return path, nil
}

func buildQuery(values Values, globals Globals) (url.Values, error) {
	query := url.Values{}
	allowed := map[string]bool{}
	for _, name := range strings.Split(values.Get("_query"), ",") {
		allowed[name] = true
	}
	if globals.Fields != "" && values.Get("_allow-fields") == "true" {
		query.Set("fields", globals.Fields)
	}
	for _, name := range []string{"limit", "offset"} {
		if !allowed[name] {
			continue
		}
		if value := values.Get(name); value != "" {
			n, err := strconv.Atoi(value)
			maxLimit := 1000
			if raw := values.Get("_max-limit"); raw != "" {
				maxLimit, _ = strconv.Atoi(raw)
			}
			invalid := err != nil || n < 0 || name == "limit" && (n < 1 || n > maxLimit)
			if invalid {
				if name == "limit" {
					return nil, fmt.Errorf("--limit must be between 1 and %d", maxLimit)
				}
				return nil, fmt.Errorf("--%s must be a non-negative integer", name)
			}
			if n > 0 {
				query.Set(name, strconv.Itoa(n))
			}
		}
	}
	for flag, parameter := range map[string]string{
		"query": "query", "country-code": "countrycode", "entity": "entity",
		"return-owned-apps": "returnOwnedApps", "name": "name", "states": "states",
		"device-classes": "deviceClasses", "expand": "expand", "language-codes": "languageCodes",
		"languages": "languages", "countries-or-regions": "countriesOrRegions",
		"include-deleted-creative-set-assets": "includeDeletedCreativeSetAssets",
		"field":                               "field", "sort-order": "sortOrder",
	} {
		if !allowed[flag] {
			continue
		}
		if value := values.Get(flag); value != "" && value != "false" {
			if flag == "country-code" {
				codes, err := countryCodes(value)
				if err != nil {
					return nil, err
				}
				value = codes[0]
			}
			query.Set(parameter, value)
		}
	}
	return query, nil
}

func countryCodes(raw string) ([]string, error) {
	codes := split(raw)
	if len(codes) == 0 {
		return nil, fmt.Errorf("country codes cannot be empty")
	}
	for index, code := range codes {
		code = strings.ToUpper(code)
		if _, ok := isoCountryCodes[code]; !ok {
			return nil, fmt.Errorf("country code %q must be an ISO 3166-1 alpha-2 code", codes[index])
		}
		codes[index] = code
	}
	return codes, nil
}

func internalQuery(names ...string) Flag {
	return Flag{Name: "_query", Default: strings.Join(names, ",")}
}

func rawArray(body any, name string) ([]any, error) {
	items, ok := body.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("%s must be a non-empty JSON array", name)
	}
	return items, nil
}

func fromJSONOr(input InputReader, values Values, fallback func() (any, error)) (any, error) {
	if source := values.Get("from-json"); source != "" {
		return input.Read(source)
	}
	return fallback()
}

func fromJSONObjectOr(input InputReader, values Values, fallback func() (any, error)) (any, error) {
	value, err := fromJSONOr(input, values, fallback)
	if err != nil {
		return nil, err
	}
	if _, err := nonEmptyObject(value, "--from-json"); err != nil {
		return nil, err
	}
	return value, nil
}

func nonEmptyObject(value any, name string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok || len(object) == 0 {
		return nil, fmt.Errorf("%s must be a non-empty JSON object", name)
	}
	return object, nil
}

func selectorInput(values Values) (string, error) {
	primary := values.Get("selector-json")
	alias := values.Get("selector")
	if primary != "" && alias != "" {
		return "", fmt.Errorf("use either --selector-json or --selector, not both")
	}
	if primary != "" {
		return primary, nil
	}
	return alias, nil
}

func selectorBody(values Values, _ Globals, input InputReader) (any, error) {
	if err := validatePaginationValues(values); err != nil {
		return nil, err
	}
	source, err := selectorInput(values)
	if err != nil {
		return nil, err
	}
	if source != "" {
		body, err := input.Read(source)
		if err != nil {
			return nil, err
		}
		selector, ok := body.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("selector must be a JSON object")
		}
		applyPagination(selector, values)
		return selector, nil
	}
	selector := map[string]any{}
	field, op, rawValues := values.Get("field"), values.Get("op"), values.Get("values")
	set := 0
	for _, value := range []string{field, op, rawValues} {
		if value != "" {
			set++
		}
	}
	if set != 0 && set != 3 {
		return nil, fmt.Errorf("--field, --op, and --values must be used together")
	}
	if set == 3 {
		selector["conditions"] = []any{map[string]any{
			"field": field, "operator": strings.ToUpper(op), "values": split(rawValues),
		}}
	}
	applyPagination(selector, values)
	return selector, nil
}

func applyPagination(selector map[string]any, values Values) {
	limit, _ := strconv.Atoi(values.Get("limit"))
	offset, _ := strconv.Atoi(values.Get("offset"))
	if limit > 0 || offset > 0 {
		pagination, _ := selector["pagination"].(map[string]any)
		if pagination == nil {
			pagination = map[string]any{}
			selector["pagination"] = pagination
		}
		if limit > 0 {
			pagination["limit"] = limit
		}
		if offset > 0 {
			pagination["offset"] = offset
		}
	}
}

func positiveInteger(name, value string) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return fmt.Errorf("--%s must be a positive integer", name)
	}
	return nil
}

func required(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--%s is required", name)
	}
	return nil
}

func oneOf(name, value string, choices ...string) error {
	value = strings.ToUpper(value)
	for _, choice := range choices {
		if value == choice {
			return nil
		}
	}
	return fmt.Errorf("--%s must be one of %s", name, strings.Join(choices, ", "))
}

func split(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func validateDate(name, value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("--%s must use YYYY-MM-DD", name)
	}
	return nil
}

func validateStartTime(value string) error {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000", "2006-01-02"} {
		if _, err := time.Parse(layout, value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("--start-time must be an ISO date or date-time")
}

func money(amount, currency string) (map[string]any, error) {
	if !decimalMoney.MatchString(amount) || strings.Trim(amount, "0.") == "" {
		return nil, fmt.Errorf("money amount %q must be a positive decimal with at most two fractional digits", amount)
	}
	if _, ok := appleCurrencies[currency]; !ok {
		return nil, fmt.Errorf("--currency must be an Apple Ads ISO 4217 currency when money flags are used")
	}
	return map[string]any{"amount": amount, "currency": currency}, nil
}
