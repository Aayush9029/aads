package commands

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func registry() []Command {
	var commands []Command
	add := func(command Command) { commands = append(commands, command) }

	add(command("acls", "list", "List user ACLs", http.MethodGet, "/acls", nil, nil))
	add(command("acls", "me", "Show the current user", http.MethodGet, "/me", nil, nil))

	add(campaignCreate())
	add(command("campaigns", "get", "Get a campaign", http.MethodGet, "/campaigns/{id}", []Flag{flagID, {Name: "_allow-fields", Default: "true"}}, nil))
	add(listCommand("campaigns", "list", "List campaigns", "/campaigns", []Flag{{Name: "_allow-fields", Default: "true"}}))
	add(findCommand("campaigns", "find", "Find campaigns", "/campaigns/find", nil))
	add(campaignUpdate())
	add(deleteCommand("campaigns", "delete", "Delete a campaign", "/campaigns/{id}", []Flag{flagID}, "Campaign deleted"))

	add(jsonMutation("budgetorders", "create", "Create a budget order", http.MethodPost, "/budgetorders", []Flag{{Name: "from-json", Description: "JSON value, @file, or @-", Required: true}}))
	add(command("budgetorders", "get", "Get a budget order", http.MethodGet, "/budgetorders/{id}", []Flag{flagID}, nil))
	add(listCommand("budgetorders", "list", "List budget orders", "/budgetorders", nil))
	add(jsonMutation("budgetorders", "update", "Update a budget order", http.MethodPut, "/budgetorders/{id}", []Flag{flagID, {Name: "from-json", Description: "JSON value, @file, or @-", Required: true}}))

	add(adGroupCreate())
	add(command("adgroups", "get", "Get an ad group", http.MethodGet, "/campaigns/{campaign-id}/adgroups/{id}", []Flag{flagCampaignID, flagID, {Name: "_allow-fields", Default: "true"}}, nil))
	add(listCommand("adgroups", "list", "List ad groups in a campaign", "/campaigns/{campaign-id}/adgroups", []Flag{flagCampaignID, {Name: "_allow-fields", Default: "true"}}))
	add(findCommand("adgroups", "find", "Find ad groups in a campaign", "/campaigns/{campaign-id}/adgroups/find", []Flag{flagCampaignID}))
	add(findCommand("adgroups", "find-all", "Find ad groups in the organization", "/adgroups/find", nil))
	add(adGroupUpdate())
	add(deleteCommand("adgroups", "delete", "Delete an ad group", "/campaigns/{campaign-id}/adgroups/{id}", []Flag{flagCampaignID, flagID}, "Ad group deleted"))

	add(keywordCreate())
	add(command("keywords", "get", "Get a targeting keyword", http.MethodGet, "/campaigns/{campaign-id}/adgroups/{adgroup-id}/targetingkeywords/{id}", []Flag{flagCampaignID, flagAdGroupID, flagID, {Name: "_allow-fields", Default: "true"}}, nil))
	add(listCommand("keywords", "list", "List targeting keywords in an ad group", "/campaigns/{campaign-id}/adgroups/{adgroup-id}/targetingkeywords", []Flag{flagCampaignID, flagAdGroupID, {Name: "_allow-fields", Default: "true"}}))
	add(keywordFind())
	add(findCommand("keywords", "find-campaign", "Find targeting keywords across a campaign", "/campaigns/{campaign-id}/adgroups/targetingkeywords/find", []Flag{flagCampaignID}))
	add(jsonMutation("keywords", "update", "Update targeting keywords", http.MethodPut, "/campaigns/{campaign-id}/adgroups/{adgroup-id}/targetingkeywords/bulk", []Flag{flagCampaignID, flagAdGroupID, {Name: "from-json", Description: "JSON array, @file, or @-", Required: true}}))
	add(bulkDelete("keywords", "delete", "Delete targeting keywords", "/campaigns/{campaign-id}/adgroups/{adgroup-id}/targetingkeywords/delete/bulk", []Flag{flagCampaignID, flagAdGroupID}, "Targeting keywords deleted"))
	add(deleteCommand("keywords", "delete-one", "Delete a targeting keyword", "/campaigns/{campaign-id}/adgroups/{adgroup-id}/targetingkeywords/{id}", []Flag{flagCampaignID, flagAdGroupID, flagID}, "Targeting keyword deleted"))

	addNegativeCommands(&commands)
	addAdCommands(&commands)
	addCreativeCommands(&commands)
	addAppCommands(&commands)
	addProductPageCommands(&commands)
	addRejectionCommands(&commands)
	addGeoCommands(&commands)
	addReportCommands(&commands)
	addImpressionShareCommands(&commands)

	return commands
}

func listCommand(group, name, summary, path string, required []Flag) Command {
	flags := appendCopy(required, flagLimit, flagOffset, flagAll)
	hasQuery := false
	for _, flag := range required {
		if flag.Name == "_query" {
			hasQuery = true
		}
	}
	if !hasQuery {
		flags = append(flags, internalQuery("limit", "offset"))
	}
	return command(group, name, summary, http.MethodGet, path, flags, nil)
}

func findCommand(group, name, summary, path string, required []Flag) Command {
	flags := appendCopy(required, selectorFlags...)
	return command(group, name, summary, http.MethodPost, path, flags, selectorBody)
}

func deleteCommand(group, name, summary, path string, flags []Flag, success string) Command {
	deleteFlags := appendCopy(flags, Flag{Name: "yes", Description: "Skip confirmation", Boolean: true})
	deleteFlags = appendCopy(deleteFlags, safetyFlags()...)
	cmd := command(group, name, summary, http.MethodDelete, path, deleteFlags, nil)
	cmd.SuccessText = success
	return cmd
}

func bulkDelete(group, name, summary, path string, required []Flag, success string) Command {
	flags := appendCopy(required,
		Flag{Name: "ids", Description: "Comma-separated resource IDs", Required: true},
		Flag{Name: "yes", Description: "Skip confirmation", Boolean: true},
		Flag{Name: "dry-run", Description: "Print the request without sending it", Boolean: true},
	)
	cmd := command(group, name, summary, http.MethodPost, path, flags, func(values Values, _ Globals, _ InputReader) (any, error) {
		ids, err := idList(values.Get("ids"))
		if err != nil {
			return nil, err
		}
		return ids, nil
	})
	cmd.Delete = true
	cmd.SuccessText = success
	return cmd
}

func jsonMutation(group, name, summary, method, path string, flags []Flag) Command {
	flags = appendCopy(flags, safetyFlags()...)
	return command(group, name, summary, method, path, flags, func(values Values, _ Globals, input InputReader) (any, error) {
		body, err := input.Read(values.Get("from-json"))
		if err != nil {
			return nil, err
		}
		if isEmptyJSON(body) {
			return nil, fmt.Errorf("--from-json cannot be empty")
		}
		if strings.Contains(path, "/bulk") {
			if _, err := rawArray(body, "--from-json"); err != nil {
				return nil, err
			}
		} else if _, err := nonEmptyObject(body, "--from-json"); err != nil {
			return nil, err
		}
		return body, nil
	})
}

func safetyFlags() []Flag {
	return []Flag{
		{Name: "dry-run", Description: "Print the request without sending it", Boolean: true},
	}
}

func appendCopy(base []Flag, added ...Flag) []Flag {
	result := append([]Flag(nil), base...)
	return append(result, added...)
}

func campaignCreate() Command {
	flags := appendCopy([]Flag{
		{Name: "name", Description: "Campaign name"},
		{Name: "adam-id", Description: "App Adam ID"},
		{Name: "daily-budget", Description: "Daily budget amount"},
		{Name: "countries", Description: "Comma-separated country codes"},
		{Name: "status", Description: "ENABLED or PAUSED"}, flagFromJSON,
		{Name: "ad-channel-type", Description: "Ad channel type"},
		{Name: "billing-event", Description: "Billing event"},
		{Name: "supply-sources", Description: "Comma-separated supply sources"},
	}, safetyFlags()...)
	return command("campaigns", "create", "Create a campaign", http.MethodPost, "/campaigns", flags, func(v Values, g Globals, input InputReader) (any, error) {
		return fromJSONObjectOr(input, v, func() (any, error) {
			for _, field := range []string{"name", "adam-id", "daily-budget", "countries", "ad-channel-type", "billing-event", "supply-sources"} {
				if err := required(field, v.Get(field)); err != nil {
					return nil, err
				}
			}
			if err := positiveInteger("adam-id", v.Get("adam-id")); err != nil {
				return nil, err
			}
			status := strings.ToUpper(v.Get("status"))
			if status != "" {
				if err := oneOf("status", status, "ENABLED", "PAUSED"); err != nil {
					return nil, err
				}
			}
			budget, err := money(v.Get("daily-budget"), strings.ToUpper(g.Currency))
			if err != nil {
				return nil, err
			}
			if err := oneOf("ad-channel-type", v.Get("ad-channel-type"), "SEARCH", "DISPLAY"); err != nil {
				return nil, err
			}
			if err := oneOf("billing-event", v.Get("billing-event"), "TAPS", "IMPRESSIONS"); err != nil {
				return nil, err
			}
			sources := split(v.Get("supply-sources"))
			for index, source := range sources {
				sources[index] = strings.ToUpper(source)
				if err := oneOf("supply-sources", source, "APPSTORE_SEARCH_RESULTS", "APPSTORE_SEARCH_TAB", "APPSTORE_TODAY_TAB", "APPSTORE_PRODUCT_PAGES_BROWSE"); err != nil {
					return nil, err
				}
			}
			countries, err := countryCodes(v.Get("countries"))
			if err != nil {
				return nil, err
			}
			body := map[string]any{
				"name": v.Get("name"), "adamId": mustInt(v.Get("adam-id")),
				"dailyBudgetAmount": budget, "countriesOrRegions": countries,
				"adChannelType": strings.ToUpper(v.Get("ad-channel-type")), "billingEvent": strings.ToUpper(v.Get("billing-event")),
				"supplySources": sources,
			}
			if status != "" {
				body["status"] = status
			}
			return body, nil
		})
	})
}

func campaignUpdate() Command {
	flags := appendCopy([]Flag{
		flagID, {Name: "name", Description: "New name"},
		{Name: "daily-budget", Description: "New daily budget"},
		{Name: "status", Description: "ENABLED or PAUSED"},
		{Name: "countries", Description: "Comma-separated country codes"}, flagFromJSON,
	}, safetyFlags()...)
	return command("campaigns", "update", "Update a campaign", http.MethodPut, "/campaigns/{id}", flags, func(v Values, g Globals, input InputReader) (any, error) {
		if v.Get("from-json") != "" {
			body, err := input.Read(v.Get("from-json"))
			if err != nil {
				return nil, err
			}
			if _, err := nonEmptyObject(body, "--from-json"); err != nil {
				return nil, err
			}
			return body, nil
		}
		payload, err := fromJSONOr(input, v, func() (any, error) {
			body := map[string]any{}
			if v.Get("name") != "" {
				body["name"] = v.Get("name")
			}
			if v.Get("countries") != "" {
				countries, err := countryCodes(v.Get("countries"))
				if err != nil {
					return nil, err
				}
				body["countriesOrRegions"] = countries
			}
			if status := strings.ToUpper(v.Get("status")); status != "" {
				if err := oneOf("status", status, "ENABLED", "PAUSED"); err != nil {
					return nil, err
				}
				body["status"] = status
			}
			if amount := v.Get("daily-budget"); amount != "" {
				value, err := money(amount, strings.ToUpper(g.Currency))
				if err != nil {
					return nil, err
				}
				body["dailyBudgetAmount"] = value
			}
			return body, nil
		})
		if err != nil {
			return nil, err
		}
		if isEmptyJSON(payload) {
			return nil, fmt.Errorf("provide at least one field to update")
		}
		return map[string]any{"campaign": payload}, nil
	})
}

func adGroupCreate() Command {
	flags := appendCopy([]Flag{flagCampaignID,
		{Name: "name", Description: "Ad group name"},
		{Name: "default-bid", Description: "Default bid amount"},
		{Name: "search-match", Description: "Enable Search Match", Boolean: true},
		{Name: "status", Description: "ENABLED or PAUSED"}, flagFromJSON,
		{Name: "pricing-model", Description: "Pricing model"},
		{Name: "start-time", Description: "Start time"},
	}, safetyFlags()...)
	return command("adgroups", "create", "Create an ad group", http.MethodPost, "/campaigns/{campaign-id}/adgroups", flags, func(v Values, g Globals, input InputReader) (any, error) {
		return fromJSONObjectOr(input, v, func() (any, error) {
			for _, field := range []string{"name", "default-bid", "pricing-model", "start-time"} {
				if err := required(field, v.Get(field)); err != nil {
					return nil, err
				}
			}
			status := strings.ToUpper(v.Get("status"))
			if status != "" {
				if err := oneOf("status", status, "ENABLED", "PAUSED"); err != nil {
					return nil, err
				}
			}
			bid, err := money(v.Get("default-bid"), strings.ToUpper(g.Currency))
			if err != nil {
				return nil, err
			}
			if err := oneOf("pricing-model", v.Get("pricing-model"), "CPC", "CPM"); err != nil {
				return nil, err
			}
			if err := validateStartTime(v.Get("start-time")); err != nil {
				return nil, err
			}
			body := map[string]any{"name": v.Get("name"), "defaultBidAmount": bid, "automatedKeywordsOptIn": v.Bool("search-match"), "pricingModel": strings.ToUpper(v.Get("pricing-model")), "startTime": v.Get("start-time")}
			if status != "" {
				body["status"] = status
			}
			return body, nil
		})
	})
}

func adGroupUpdate() Command {
	flags := appendCopy([]Flag{flagCampaignID, flagID,
		{Name: "name", Description: "New name"}, {Name: "default-bid", Description: "New default bid"},
		{Name: "status", Description: "ENABLED or PAUSED"}, {Name: "search-match", Description: "true or false"}, flagFromJSON,
	}, safetyFlags()...)
	return command("adgroups", "update", "Update an ad group", http.MethodPut, "/campaigns/{campaign-id}/adgroups/{id}", flags, func(v Values, g Globals, input InputReader) (any, error) {
		body, err := fromJSONObjectOr(input, v, func() (any, error) {
			body := map[string]any{}
			if v.Get("name") != "" {
				body["name"] = v.Get("name")
			}
			if v.Get("default-bid") != "" {
				bid, err := money(v.Get("default-bid"), strings.ToUpper(g.Currency))
				if err != nil {
					return nil, err
				}
				body["defaultBidAmount"] = bid
			}
			if status := strings.ToUpper(v.Get("status")); status != "" {
				if err := oneOf("status", status, "ENABLED", "PAUSED"); err != nil {
					return nil, err
				}
				body["status"] = status
			}
			if raw := v.Get("search-match"); raw != "" {
				value, err := strconv.ParseBool(raw)
				if err != nil {
					return nil, fmt.Errorf("--search-match must be true or false")
				}
				body["automatedKeywordsOptIn"] = value
			}
			return body, nil
		})
		if err != nil {
			return nil, err
		}
		if isEmptyJSON(body) {
			return nil, fmt.Errorf("provide at least one field to update")
		}
		return body, nil
	})
}

func keywordCreate() Command {
	flags := appendCopy([]Flag{flagCampaignID, flagAdGroupID,
		{Name: "text", Description: "Keyword text"}, {Name: "match-type", Description: "BROAD or EXACT", Default: "BROAD"},
		{Name: "bid", Description: "Bid amount"}, flagFromJSON,
	}, safetyFlags()...)
	return command("keywords", "create", "Create targeting keywords", http.MethodPost, "/campaigns/{campaign-id}/adgroups/{adgroup-id}/targetingkeywords/bulk", flags, func(v Values, g Globals, input InputReader) (any, error) {
		body, err := fromJSONOr(input, v, func() (any, error) {
			if err := required("text", v.Get("text")); err != nil {
				return nil, err
			}
			match := strings.ToUpper(v.Get("match-type"))
			if err := oneOf("match-type", match, "BROAD", "EXACT"); err != nil {
				return nil, err
			}
			item := map[string]any{"text": v.Get("text"), "matchType": match}
			if v.Get("bid") != "" {
				bid, err := money(v.Get("bid"), strings.ToUpper(g.Currency))
				if err != nil {
					return nil, err
				}
				item["bidAmount"] = bid
			}
			return []any{item}, nil
		})
		if err != nil {
			return nil, err
		}
		return rawArray(body, "keyword input")
	})
}

func keywordFind() Command {
	flags := appendCopy([]Flag{flagCampaignID, flagAdGroupID}, selectorFlags...)
	return command("keywords", "find", "Find targeting keywords in an ad group", http.MethodPost, "/campaigns/{campaign-id}/adgroups/targetingkeywords/find", flags, func(v Values, g Globals, input InputReader) (any, error) {
		body, err := selectorBody(v, g, input)
		if err != nil {
			return nil, err
		}
		selector := body.(map[string]any)
		if err := positiveInteger("adgroup-id", v.Get("adgroup-id")); err != nil {
			return nil, err
		}
		conditions, _ := selector["conditions"].([]any)
		selector["conditions"] = append(conditions, map[string]any{"field": "adGroupId", "operator": "EQUALS", "values": []string{v.Get("adgroup-id")}})
		return selector, nil
	})
}

func addNegativeCommands(commands *[]Command) {
	for _, scope := range []struct {
		prefix, label, base string
		ids                 []Flag
	}{
		{"campaign", "campaign-level", "/campaigns/{campaign-id}/negativekeywords", []Flag{flagCampaignID}},
		{"adgroup", "ad group-level", "/campaigns/{campaign-id}/adgroups/{adgroup-id}/negativekeywords", []Flag{flagCampaignID, flagAdGroupID}},
	} {
		createFlags := appendCopy(scope.ids, Flag{Name: "text", Description: "Keyword text"}, Flag{Name: "match-type", Description: "BROAD or EXACT", Default: "BROAD"}, flagFromJSON)
		createFlags = appendCopy(createFlags, safetyFlags()...)
		*commands = append(*commands, command("negatives", scope.prefix+"-create", "Create "+scope.label+" negative keywords", http.MethodPost, scope.base+"/bulk", createFlags, negativeCreateBody))
		*commands = append(*commands, command("negatives", scope.prefix+"-get", "Get a "+scope.label+" negative keyword", http.MethodGet, scope.base+"/{id}", appendCopy(scope.ids, flagID, Flag{Name: "_allow-fields", Default: "true"}), nil))
		*commands = append(*commands, listCommand("negatives", scope.prefix+"-list", "List "+scope.label+" negative keywords", scope.base, appendCopy(scope.ids, Flag{Name: "_allow-fields", Default: "true"})))
		findPath := scope.base + "/find"
		findIDs := scope.ids
		if scope.prefix == "adgroup" {
			findPath = "/campaigns/{campaign-id}/adgroups/negativekeywords/find"
			findIDs = []Flag{flagCampaignID}
		}
		*commands = append(*commands, findCommand("negatives", scope.prefix+"-find", "Find "+scope.label+" negative keywords", findPath, findIDs))
		*commands = append(*commands, jsonMutation("negatives", scope.prefix+"-update", "Update "+scope.label+" negative keywords", http.MethodPut, scope.base+"/bulk", appendCopy(scope.ids, Flag{Name: "from-json", Description: "JSON array, @file, or @-", Required: true})))
		*commands = append(*commands, bulkDelete("negatives", scope.prefix+"-delete", "Delete "+scope.label+" negative keywords", scope.base+"/delete/bulk", scope.ids, "Negative keywords deleted"))
	}
}

func negativeCreateBody(v Values, _ Globals, input InputReader) (any, error) {
	body, err := fromJSONOr(input, v, func() (any, error) {
		if err := required("text", v.Get("text")); err != nil {
			return nil, err
		}
		match := strings.ToUpper(v.Get("match-type"))
		if err := oneOf("match-type", match, "BROAD", "EXACT"); err != nil {
			return nil, err
		}
		return []any{map[string]any{"text": v.Get("text"), "matchType": match}}, nil
	})
	if err != nil {
		return nil, err
	}
	return rawArray(body, "negative keyword input")
}

func addAdCommands(commands *[]Command) {
	createFlags := appendCopy([]Flag{flagCampaignID, flagAdGroupID,
		{Name: "creative-id", Description: "Creative ID"}, {Name: "name", Description: "Ad name"},
		{Name: "status", Description: "ENABLED or PAUSED"}, flagFromJSON,
	}, safetyFlags()...)
	*commands = append(*commands, command("ads", "create", "Create an ad", http.MethodPost, "/campaigns/{campaign-id}/adgroups/{adgroup-id}/ads", createFlags, func(v Values, _ Globals, input InputReader) (any, error) {
		return fromJSONObjectOr(input, v, func() (any, error) {
			if err := positiveInteger("creative-id", v.Get("creative-id")); err != nil {
				return nil, err
			}
			if err := required("name", v.Get("name")); err != nil {
				return nil, err
			}
			if err := required("status", v.Get("status")); err != nil {
				return nil, err
			}
			status := strings.ToUpper(v.Get("status"))
			if status != "" {
				if err := oneOf("status", status, "ENABLED", "PAUSED"); err != nil {
					return nil, err
				}
			}
			return map[string]any{"creativeId": mustInt(v.Get("creative-id")), "name": v.Get("name"), "status": status}, nil
		})
	}))
	*commands = append(*commands, command("ads", "get", "Get an ad", http.MethodGet, "/campaigns/{campaign-id}/adgroups/{adgroup-id}/ads/{id}", []Flag{flagCampaignID, flagAdGroupID, flagID}, nil))
	*commands = append(*commands, command("ads", "list", "List ads in an ad group", http.MethodGet, "/campaigns/{campaign-id}/adgroups/{adgroup-id}/ads", []Flag{flagCampaignID, flagAdGroupID}, nil))
	*commands = append(*commands, findCommand("ads", "find", "Find ads in a campaign", "/campaigns/{campaign-id}/ads/find", []Flag{flagCampaignID}))
	*commands = append(*commands, findCommand("ads", "find-all", "Find ads in the organization", "/ads/find", nil))
	updateFlags := appendCopy([]Flag{flagCampaignID, flagAdGroupID, flagID,
		{Name: "name", Description: "New name"}, {Name: "status", Description: "ENABLED or PAUSED"}, flagFromJSON,
	}, safetyFlags()...)
	*commands = append(*commands, command("ads", "update", "Update an ad", http.MethodPut, "/campaigns/{campaign-id}/adgroups/{adgroup-id}/ads/{id}", updateFlags, func(v Values, _ Globals, input InputReader) (any, error) {
		body, err := fromJSONObjectOr(input, v, func() (any, error) {
			body := map[string]any{}
			if v.Get("name") != "" {
				body["name"] = v.Get("name")
			}
			if status := strings.ToUpper(v.Get("status")); status != "" {
				if err := oneOf("status", status, "ENABLED", "PAUSED"); err != nil {
					return nil, err
				}
				body["status"] = status
			}
			return body, nil
		})
		if err != nil {
			return nil, err
		}
		if isEmptyJSON(body) {
			return nil, fmt.Errorf("provide at least one field to update")
		}
		return body, nil
	}))
	*commands = append(*commands, deleteCommand("ads", "delete", "Delete an ad", "/campaigns/{campaign-id}/adgroups/{adgroup-id}/ads/{id}", []Flag{flagCampaignID, flagAdGroupID, flagID}, "Ad deleted"))
}

func addCreativeCommands(commands *[]Command) {
	flags := appendCopy([]Flag{{Name: "adam-id", Description: "App Adam ID"}, {Name: "product-page-id", Description: "Product page ID"}, {Name: "name", Description: "Creative name"}, {Name: "type", Description: "Creative type"}, flagFromJSON}, safetyFlags()...)
	*commands = append(*commands, command("creatives", "create", "Create a creative", http.MethodPost, "/creatives", flags, func(v Values, _ Globals, input InputReader) (any, error) {
		return fromJSONObjectOr(input, v, func() (any, error) {
			if err := positiveInteger("adam-id", v.Get("adam-id")); err != nil {
				return nil, err
			}
			for _, name := range []string{"name", "type"} {
				if err := required(name, v.Get(name)); err != nil {
					return nil, err
				}
			}
			creativeType := strings.ToUpper(v.Get("type"))
			if err := oneOf("type", creativeType, "CUSTOM_PRODUCT_PAGE", "DEFAULT_PRODUCT_PAGE"); err != nil {
				return nil, err
			}
			if creativeType == "CUSTOM_PRODUCT_PAGE" {
				if err := required("product-page-id", v.Get("product-page-id")); err != nil {
					return nil, err
				}
			}
			body := map[string]any{"adamId": mustInt(v.Get("adam-id")), "name": v.Get("name"), "type": creativeType}
			if v.Get("product-page-id") != "" {
				body["productPageId"] = v.Get("product-page-id")
			}
			return body, nil
		})
	}))
	*commands = append(*commands, command("creatives", "get", "Get a creative", http.MethodGet, "/creatives/{id}", []Flag{flagID, {Name: "include-deleted-creative-set-assets", Description: "Include deleted assets", Boolean: true}, internalQuery("include-deleted-creative-set-assets")}, nil))
	*commands = append(*commands, listCommand("creatives", "list", "List creatives", "/creatives", nil))
	*commands = append(*commands, findCommand("creatives", "find", "Find creatives", "/creatives/find", nil))
}

func addAppCommands(commands *[]Command) {
	*commands = append(*commands, listCommand("apps", "search", "Search for iOS apps", "/search/apps", []Flag{{Name: "query", Description: "Search query", Required: true}, {Name: "return-owned-apps", Description: "Only return owned apps", Boolean: true}, internalQuery("query", "return-owned-apps", "limit", "offset")}))
	flags := appendCopy([]Flag{flagAdamID}, selectorFlags...)
	*commands = append(*commands, command("apps", "eligibility", "Check app eligibility", http.MethodPost, "/apps/{adam-id}/eligibilities/find", flags, selectorBody))
	*commands = append(*commands, command("apps", "details", "Get app details", http.MethodGet, "/apps/{adam-id}", []Flag{flagAdamID}, nil))
	*commands = append(*commands, command("apps", "localized", "Get localized app details", http.MethodGet, "/apps/{adam-id}/locale-details", []Flag{flagAdamID}, nil))
}

func addProductPageCommands(commands *[]Command) {
	*commands = append(*commands, command("product-pages", "list", "List product pages for an app", http.MethodGet, "/apps/{adam-id}/product-pages", []Flag{flagAdamID, {Name: "name", Description: "Name filter"}, {Name: "states", Description: "Comma-separated states"}, internalQuery("name", "states")}, nil))
	*commands = append(*commands, command("product-pages", "get", "Get a product page", http.MethodGet, "/apps/{adam-id}/product-pages/{id}", []Flag{flagAdamID, {Name: "id", Description: "Product page ID", Required: true}}, nil))
	*commands = append(*commands, command("product-pages", "locales", "Get product page locale details", http.MethodGet, "/apps/{adam-id}/product-pages/{id}/locale-details", []Flag{flagAdamID, {Name: "id", Description: "Product page ID", Required: true}, {Name: "device-classes", Description: "Device classes"}, {Name: "expand", Description: "Expanded fields"}, {Name: "language-codes", Description: "Language codes"}, {Name: "languages", Description: "Languages"}, internalQuery("device-classes", "expand", "language-codes", "languages")}, nil))
	*commands = append(*commands, command("product-pages", "countries", "List supported countries and regions", http.MethodGet, "/countries-or-regions", []Flag{{Name: "countries-or-regions", Description: "Country or region codes"}, internalQuery("countries-or-regions")}, nil))
	*commands = append(*commands, command("product-pages", "device-sizes", "Get app preview device sizes", http.MethodGet, "/creativeappmappings/devices", nil, nil))
}

func addRejectionCommands(commands *[]Command) {
	*commands = append(*commands, findCommand("ad-rejections", "find", "Find ad rejection reasons", "/product-page-reasons/find", nil))
	*commands = append(*commands, command("ad-rejections", "get", "Get an ad rejection reason", http.MethodGet, "/product-page-reasons/{id}", []Flag{flagID}, nil))
	flags := appendCopy([]Flag{flagAdamID}, selectorFlags...)
	*commands = append(*commands, command("ad-rejections", "find-assets", "Find app assets", http.MethodPost, "/apps/{adam-id}/assets/find", flags, selectorBody))
}

func addGeoCommands(commands *[]Command) {
	*commands = append(*commands, command("geo", "search", "Search geolocations", http.MethodGet, "/search/geo", []Flag{
		{Name: "query", Description: "Search query"}, {Name: "country-code", Description: "Country code"},
		{Name: "entity", Description: "Country, AdminArea, or Locality"}, flagLimit, flagOffset, flagAll, internalQuery("query", "country-code", "entity", "limit", "offset"),
	}, nil))
	flags := []Flag{{Name: "geo-id", Description: "Geo identifier"}, {Name: "entity", Description: "Country, AdminArea, or Locality"}, {Name: "from-json", Description: "JSON array, @file, or @-"}, flagLimit, flagOffset, flagAll, internalQuery("limit", "offset")}
	*commands = append(*commands, command("geo", "get", "Get geolocation details", http.MethodPost, "/search/geo", flags, func(v Values, _ Globals, input InputReader) (any, error) {
		body, err := fromJSONOr(input, v, func() (any, error) {
			if err := required("geo-id", v.Get("geo-id")); err != nil {
				return nil, err
			}
			if err := oneOf("entity", v.Get("entity"), "COUNTRY", "ADMINAREA", "LOCALITY"); err != nil {
				return nil, err
			}
			entities := map[string]string{"COUNTRY": "Country", "ADMINAREA": "AdminArea", "LOCALITY": "Locality"}
			return []any{map[string]any{"id": v.Get("geo-id"), "entity": entities[strings.ToUpper(v.Get("entity"))]}}, nil
		})
		if err != nil {
			return nil, err
		}
		return rawArray(body, "geo input")
	}))
}

func addReportCommands(commands *[]Command) {
	scopes := []struct {
		name, summary, path string
		ids                 []Flag
		adGroupVariant      bool
	}{
		{name: "campaigns", summary: "Get campaign-level reports", path: "/reports/campaigns"},
		{name: "adgroups", summary: "Get ad group-level reports", path: "/reports/campaigns/{campaign-id}/adgroups", ids: []Flag{flagCampaignID}},
		{name: "keywords", summary: "Get keyword-level reports", path: "/reports/campaigns/{campaign-id}/keywords", ids: []Flag{flagCampaignID}, adGroupVariant: true},
		{name: "searchterms", summary: "Get search term-level reports", path: "/reports/campaigns/{campaign-id}/searchterms", ids: []Flag{flagCampaignID}, adGroupVariant: true},
		{name: "ads", summary: "Get ad-level reports", path: "/reports/campaigns/{campaign-id}/ads", ids: []Flag{flagCampaignID}},
	}
	for _, scope := range scopes {
		if scope.adGroupVariant {
			scope.ids = appendCopy(scope.ids, Flag{Name: "adgroup-id", Description: "Limit the report to an ad group"})
		}
		flags := appendCopy(scope.ids,
			Flag{Name: "start-time", Description: "Start date (YYYY-MM-DD)"},
			Flag{Name: "end-time", Description: "End date (YYYY-MM-DD)"},
			Flag{Name: "granularity", Description: "HOURLY, DAILY, WEEKLY, MONTHLY, or NONE", Default: "DAILY"},
			Flag{Name: "group-by", Description: "Grouping dimension"},
			Flag{Name: "selector-json", Description: "Selector JSON, @file, or @-"},
			Flag{Name: "selector", Description: "Alias for --selector-json"},
			Flag{Name: "from-json", Description: "Complete reporting request, @file, or @-"},
			Flag{Name: "time-zone", Description: "ORTZ or UTC"},
			Flag{Name: "return-row-totals", Description: "Return row totals", Boolean: true},
			Flag{Name: "return-grand-totals", Description: "Return grand totals", Boolean: true},
			Flag{Name: "return-records-with-no-metrics", Description: "Include records with no metrics", Boolean: true},
			flagLimit, flagOffset, flagAll,
		)
		built := command("reports", scope.name, scope.summary, http.MethodPost, scope.path, flags, reportBody)
		if scope.adGroupVariant {
			baseBuild := built.Build
			name := scope.name
			built.Build = func(values Values, globals Globals, input InputReader) (Request, error) {
				request, err := baseBuild(values, globals, input)
				if err != nil {
					return Request{}, err
				}
				if value := values.Get("adgroup-id"); value != "" {
					if err := positiveInteger("adgroup-id", value); err != nil {
						return Request{}, err
					}
					request.Path = fmt.Sprintf("/reports/campaigns/%s/adgroups/%s/%s", values.Get("campaign-id"), value, name)
				}
				return request, nil
			}
		}
		*commands = append(*commands, built)
	}
}

func reportBody(v Values, _ Globals, input InputReader) (any, error) {
	selectorSource, err := selectorInput(v)
	if err != nil {
		return nil, err
	}
	if source := v.Get("from-json"); source != "" {
		body, err := input.Read(source)
		if err != nil {
			return nil, err
		}
		object, ok := body.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("--from-json must be a JSON object")
		}
		selector, ok := object["selector"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("reporting request must contain a selector object")
		}
		if err := validatePaginationValues(v); err != nil {
			return nil, err
		}
		applyPagination(selector, v)
		return object, nil
	}
	if v.Get("selector-json") != "" && v.Get("selector") != "" {
		return nil, fmt.Errorf("use either --selector-json or --selector, not both")
	}
	for _, name := range []string{"start-time", "end-time"} {
		if err := required(name, v.Get(name)); err != nil {
			return nil, err
		}
	}
	for _, name := range []string{"start-time", "end-time"} {
		if err := validateDate(name, v.Get(name)); err != nil {
			return nil, err
		}
	}
	start, _ := timeParse(v.Get("start-time"))
	end, _ := timeParse(v.Get("end-time"))
	if start.After(end) {
		return nil, fmt.Errorf("--start-time cannot be after --end-time")
	}
	granularity := strings.ToUpper(v.Get("granularity"))
	if err := oneOf("granularity", granularity, "HOURLY", "DAILY", "WEEKLY", "MONTHLY", "NONE"); err != nil {
		return nil, err
	}
	if granularity == "NONE" && !v.Bool("return-row-totals") {
		return nil, fmt.Errorf("--return-row-totals is required when granularity is NONE")
	}
	if granularity != "NONE" && (v.Bool("return-row-totals") || v.Bool("return-grand-totals")) {
		return nil, fmt.Errorf("totals cannot be combined with granularity")
	}
	body := map[string]any{"startTime": v.Get("start-time"), "endTime": v.Get("end-time"), "returnRowTotals": v.Bool("return-row-totals"), "returnGrandTotals": v.Bool("return-grand-totals")}
	if granularity != "NONE" {
		body["granularity"] = granularity
	}
	body["returnRecordsWithNoMetrics"] = v.Bool("return-records-with-no-metrics")
	if zone := strings.ToUpper(v.Get("time-zone")); zone != "" {
		if err := oneOf("time-zone", zone, "ORTZ", "UTC"); err != nil {
			return nil, err
		}
		body["timeZone"] = zone
	}
	if group := v.Get("group-by"); group != "" {
		body["groupBy"] = split(group)
	}
	if selectorSource != "" {
		selector, err := input.Read(selectorSource)
		if err != nil {
			return nil, err
		}
		if _, ok := selector.(map[string]any); !ok {
			return nil, fmt.Errorf("selector must be a JSON object")
		}
		body["selector"] = selector
	} else {
		body["selector"] = map[string]any{"orderBy": []any{map[string]any{"field": "localSpend", "sortOrder": "DESCENDING"}}}
	}
	if err := validatePaginationValues(v); err != nil {
		return nil, err
	}
	applyPagination(body["selector"].(map[string]any), v)
	return body, nil
}

func addImpressionShareCommands(commands *[]Command) {
	flags := appendCopy([]Flag{flagFromJSON,
		{Name: "start-time", Description: "Start date (YYYY-MM-DD)"}, {Name: "end-time", Description: "End date (YYYY-MM-DD)"},
		{Name: "name", Description: "Report name"}, {Name: "granularity", Description: "DAILY or WEEKLY", Default: "DAILY"},
		{Name: "date-range", Description: "LAST_WEEK, LAST_2_WEEKS, or LAST_4_WEEKS"},
		{Name: "selector-json", Description: "Selector JSON, @file, or @-"}, {Name: "selector", Description: "Alias for --selector-json"},
	}, safetyFlags()...)
	*commands = append(*commands, command("impression-share", "create", "Create an impression share report", http.MethodPost, "/custom-reports", flags, func(v Values, _ Globals, input InputReader) (any, error) {
		if v.Get("from-json") != "" {
			body, err := input.Read(v.Get("from-json"))
			if err != nil {
				return nil, err
			}
			object, ok := body.(map[string]any)
			if !ok || len(object) == 0 {
				return nil, fmt.Errorf("--from-json must be a non-empty JSON object")
			}
			return object, nil
		}
		return fromJSONOr(input, v, func() (any, error) {
			if v.Get("selector-json") != "" && v.Get("selector") != "" {
				return nil, fmt.Errorf("use either --selector-json or --selector, not both")
			}
			if err := required("name", v.Get("name")); err != nil {
				return nil, err
			}
			if len(v.Get("name")) > 50 {
				return nil, fmt.Errorf("--name cannot exceed 50 characters")
			}
			granularity := strings.ToUpper(v.Get("granularity"))
			if err := oneOf("granularity", granularity, "DAILY", "WEEKLY"); err != nil {
				return nil, err
			}
			body := map[string]any{"name": v.Get("name"), "granularity": granularity}
			if granularity == "DAILY" {
				if (v.Get("start-time") == "") != (v.Get("end-time") == "") {
					return nil, fmt.Errorf("--start-time and --end-time must be used together")
				}
				if v.Get("start-time") != "" {
					for _, name := range []string{"start-time", "end-time"} {
						if err := validateDate(name, v.Get(name)); err != nil {
							return nil, err
						}
					}
					start, _ := timeParse(v.Get("start-time"))
					end, _ := timeParse(v.Get("end-time"))
					minimum, _ := timeParse("2020-04-12")
					if start.Before(minimum) || end.Before(minimum) {
						return nil, fmt.Errorf("report dates cannot be before 2020-04-12")
					}
					if start.After(end) {
						return nil, fmt.Errorf("--start-time cannot be after --end-time")
					}
					if end.Sub(start) > 30*24*time.Hour {
						return nil, fmt.Errorf("daily impression share reports cannot exceed 30 days")
					}
					body["startTime"], body["endTime"] = v.Get("start-time"), v.Get("end-time")
				}
				if v.Get("date-range") != "" {
					return nil, fmt.Errorf("--date-range is only valid with WEEKLY granularity")
				}
			} else {
				if v.Get("start-time") != "" || v.Get("end-time") != "" {
					return nil, fmt.Errorf("WEEKLY reports use --date-range instead of start and end dates")
				}
				rangeValue := strings.ToUpper(v.Get("date-range"))
				if rangeValue == "" {
					rangeValue = "LAST_WEEK"
				}
				if err := oneOf("date-range", rangeValue, "LAST_WEEK", "LAST_2_WEEKS", "LAST_4_WEEKS"); err != nil {
					return nil, err
				}
				body["dateRange"] = rangeValue
			}
			selectorSource := v.Get("selector-json")
			if selectorSource == "" {
				selectorSource = v.Get("selector")
			}
			if selectorSource != "" {
				selector, err := input.Read(selectorSource)
				if err != nil {
					return nil, err
				}
				if _, ok := selector.(map[string]any); !ok {
					return nil, fmt.Errorf("selector must be a JSON object")
				}
				body["selector"] = selector
			}
			return body, nil
		})
	}))
	*commands = append(*commands, command("impression-share", "get", "Get an impression share report", http.MethodGet, "/custom-reports/{id}", []Flag{flagID}, nil))
	*commands = append(*commands, command("impression-share", "list", "List impression share reports", http.MethodGet, "/custom-reports", []Flag{{Name: "field", Description: "Sort field"}, {Name: "sort-order", Description: "Sort order"}, {Name: "limit", Description: "Maximum results", Default: "50"}, flagOffset, flagAll, {Name: "_max-limit", Default: "50"}, internalQuery("field", "sort-order", "limit", "offset")}, nil))
}

func idList(raw string) ([]int64, error) {
	items := split(raw)
	if len(items) == 0 {
		return nil, fmt.Errorf("--ids must contain at least one ID")
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		n, err := strconv.ParseInt(item, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("--ids contains invalid ID %q", item)
		}
		ids = append(ids, n)
	}
	return ids, nil
}

func mustInt(raw string) int64 { value, _ := strconv.ParseInt(raw, 10, 64); return value }

func isEmptyJSON(value any) bool {
	switch value := value.(type) {
	case nil:
		return true
	case map[string]any:
		return len(value) == 0
	case []any:
		return len(value) == 0
	default:
		return false
	}
}

func timeParse(raw string) (time.Time, error) { return time.Parse("2006-01-02", raw) }
