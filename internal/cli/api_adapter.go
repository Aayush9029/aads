package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Aayush9029/aads/internal/api"
	"github.com/Aayush9029/aads/internal/commands"
)

type apiAdapter struct{ client *api.Client }

func (adapter *apiAdapter) Do(ctx context.Context, request commands.Request) (commands.Result, error) {
	if !request.AllPages {
		response, err := adapter.request(ctx, request)
		if err != nil {
			return commands.Result{}, err
		}
		if len(response.Body) == 0 {
			return commands.Result{}, nil
		}
		return commands.Result{Data: response.Envelope.Data, Envelope: response.Body}, nil
	}
	if strings.HasPrefix(request.Operation, "reports.") {
		return adapter.collectReportPages(ctx, request)
	}
	limit, offset := paginationValues(request)
	if request.Operation == "impression-share.list" && limit == 0 {
		limit = 50
	}
	pages, err := api.CollectPages(ctx, api.PaginationOptions{Limit: limit, Offset: offset}, func(ctx context.Context, pageLimit, pageOffset int) (*api.Response, error) {
		page := withPagination(request, pageLimit, pageOffset)
		return adapter.request(ctx, page)
	})
	if err != nil {
		return commands.Result{}, err
	}
	envelope, err := pages.CombinedEnvelope()
	if err != nil {
		return commands.Result{}, err
	}
	raw, err := marshalEnvelope(envelope)
	if err != nil {
		return commands.Result{}, err
	}
	return commands.Result{Data: envelope.Data, Envelope: raw}, nil
}

func (adapter *apiAdapter) collectReportPages(ctx context.Context, request commands.Request) (commands.Result, error) {
	limit, offset := paginationValues(request)
	if limit == 0 {
		limit = api.DefaultPageSize
	}
	var rows []json.RawMessage
	var first *api.Response
	var lastPagination *api.PageDetail
	completed := false
	for pageNumber := 0; pageNumber < api.DefaultMaxPages; pageNumber++ {
		if err := ctx.Err(); err != nil {
			return commands.Result{}, err
		}
		response, err := adapter.request(ctx, withPagination(request, limit, offset))
		if err != nil {
			return commands.Result{}, err
		}
		if first == nil {
			first = response
		}
		lastPagination = response.Envelope.Pagination
		pageRows, err := reportRows(response.Envelope.Data)
		if err != nil {
			return commands.Result{}, fmt.Errorf("report page %d: %w", pageNumber+1, err)
		}
		rows = append(rows, pageRows...)
		if len(pageRows) == 0 {
			completed = true
			break
		}
		nextOffset := offset + len(pageRows)
		if lastPagination != nil && lastPagination.TotalResults > 0 && nextOffset >= lastPagination.TotalResults {
			completed = true
			break
		}
		if len(pageRows) < limit || nextOffset <= offset {
			completed = true
			break
		}
		offset = nextOffset
	}
	if first == nil {
		return commands.Result{}, errors.New("report pagination returned no response")
	}
	if !completed {
		return commands.Result{}, fmt.Errorf("report pagination exceeded %d pages", api.DefaultMaxPages)
	}
	data, err := combineReportData(first.Envelope.Data, rows)
	if err != nil {
		return commands.Result{}, err
	}
	envelope := first.Envelope
	envelope.Data = data
	envelope.Pagination = lastPagination
	raw, err := marshalEnvelope(envelope)
	if err != nil {
		return commands.Result{}, err
	}
	return commands.Result{Data: data, Envelope: raw}, nil
}

func reportRows(data json.RawMessage) ([]json.RawMessage, error) {
	_, reporting, err := reportObjects(data)
	if err != nil {
		return nil, err
	}
	raw := reporting["row"]
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("reportingDataResponse.row is not an array: %w", err)
	}
	return rows, nil
}

func combineReportData(data json.RawMessage, rows []json.RawMessage) (json.RawMessage, error) {
	outer, reporting, err := reportObjects(data)
	if err != nil {
		return nil, err
	}
	reporting["row"], err = json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	outer["reportingDataResponse"], err = json.Marshal(reporting)
	if err != nil {
		return nil, err
	}
	return json.Marshal(outer)
}

func reportObjects(data json.RawMessage) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	outer := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &outer); err != nil {
		return nil, nil, fmt.Errorf("report data is not an object: %w", err)
	}
	raw, ok := outer["reportingDataResponse"]
	if !ok {
		return nil, nil, errors.New("report data is missing reportingDataResponse")
	}
	reporting := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &reporting); err != nil {
		return nil, nil, fmt.Errorf("reportingDataResponse is not an object: %w", err)
	}
	return outer, reporting, nil
}

func (adapter *apiAdapter) request(ctx context.Context, request commands.Request) (*api.Response, error) {
	path := request.Path
	if query := request.Query.Encode(); query != "" {
		path += "?" + query
	}
	mode := api.OrgContextOmit
	if request.OrgContext {
		mode = api.OrgContextInclude
	}
	return adapter.client.Request(ctx, api.Request{Method: request.Method, Path: path, Body: request.Body, Operation: operation(request), OrgContext: mode})
}

func operation(request commands.Request) api.Operation {
	if strings.HasPrefix(request.Operation, "reports.") {
		return api.OperationReport
	}
	if request.Operation == "impression-share.create" {
		return api.OperationCreate
	}
	if strings.HasPrefix(request.Operation, "impression-share.") {
		return api.OperationRead
	}
	if request.Operation == "geo.get" {
		return api.OperationRead
	}
	if strings.Contains(request.Operation, ".find") || request.Operation == "apps.eligibility" {
		return api.OperationFind
	}
	if request.Method == "GET" {
		return api.OperationRead
	}
	if request.Method == "DELETE" {
		return api.OperationDelete
	}
	if strings.Contains(request.Operation, "delete") {
		return api.OperationBulkDelete
	}
	if request.Method == "PUT" {
		return api.OperationUpdate
	}
	return api.OperationCreate
}

func paginationValues(request commands.Request) (int, int) {
	limit, _ := strconv.Atoi(request.Query.Get("limit"))
	offset, _ := strconv.Atoi(request.Query.Get("offset"))
	if body, ok := request.Body.(map[string]any); ok {
		container := body
		if selector, ok := body["selector"].(map[string]any); ok {
			container = selector
		}
		if pagination, ok := container["pagination"].(map[string]any); ok {
			if value, ok := number(pagination["limit"]); ok {
				limit = value
			}
			if value, ok := number(pagination["offset"]); ok {
				offset = value
			}
		}
	}
	return limit, offset
}

func withPagination(request commands.Request, limit, offset int) commands.Request {
	request.Query = cloneQuery(request.Query)
	if body, ok := request.Body.(map[string]any); ok {
		cloned := cloneMap(body)
		container := cloned
		if selector, ok := cloned["selector"].(map[string]any); ok {
			container = selector
		} else if strings.HasPrefix(request.Operation, "reports.") {
			selector := map[string]any{}
			cloned["selector"] = selector
			container = selector
		}
		pagination, _ := container["pagination"].(map[string]any)
		if pagination == nil {
			pagination = map[string]any{}
			container["pagination"] = pagination
		}
		pagination["limit"] = limit
		pagination["offset"] = offset
		request.Body = cloned
	} else {
		request.Query.Set("limit", strconv.Itoa(limit))
		request.Query.Set("offset", strconv.Itoa(offset))
	}
	return request
}

func cloneQuery(source url.Values) url.Values {
	clone := url.Values{}
	for key, values := range source {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}
func cloneMap(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		if nested, ok := value.(map[string]any); ok {
			clone[key] = cloneMap(nested)
		} else {
			clone[key] = value
		}
	}
	return clone
}
func number(value any) (int, bool) {
	switch value := value.(type) {
	case int:
		return value, true
	case float64:
		return int(value), true
	case json.Number:
		n, err := strconv.Atoi(string(value))
		return n, err == nil
	default:
		return 0, false
	}
}

var _ commands.Client = (*apiAdapter)(nil)
