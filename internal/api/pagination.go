package api

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	DefaultPageSize = 1000
	DefaultMaxPages = 1000
)

// PaginationOptions controls offset-based collection without mutating the
// caller's selector or query state.
type PaginationOptions struct {
	Offset   int
	Limit    int
	MaxPages int
}

// PageFetcher fetches exactly one page at the supplied limit and offset.
type PageFetcher func(context.Context, int, int) (*Response, error)

// PaginatedResult preserves every raw page and exposes array elements without
// decoding their evolving Apple Ads schema.
type PaginatedResult struct {
	Pages      []*Response
	Items      []json.RawMessage
	Pagination *PageDetail
	Offset     int
}

// CollectPages follows Apple Ads offset pagination. It stops on totals, an
// empty/short page, non-advancing metadata, cancellation, or MaxPages.
func CollectPages(ctx context.Context, options PaginationOptions, fetch PageFetcher) (*PaginatedResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if fetch == nil {
		return nil, fmt.Errorf("page fetcher is nil")
	}
	if options.Offset < 0 {
		return nil, fmt.Errorf("pagination offset cannot be negative")
	}
	limit := options.Limit
	if limit == 0 {
		limit = DefaultPageSize
	}
	if limit < 1 || limit > DefaultPageSize {
		return nil, fmt.Errorf("pagination limit must be between 1 and %d", DefaultPageSize)
	}
	maxPages := options.MaxPages
	if maxPages == 0 {
		maxPages = DefaultMaxPages
	}
	if maxPages < 1 {
		return nil, fmt.Errorf("max pages must be positive")
	}

	result := &PaginatedResult{Offset: options.Offset}
	offset := options.Offset
	for pageNumber := 0; pageNumber < maxPages; pageNumber++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := fetch(ctx, limit, offset)
		if err != nil {
			return nil, err
		}
		if page == nil {
			return nil, fmt.Errorf("page fetcher returned nil response")
		}
		if pageNumber > 0 && page.Envelope.Pagination != nil && page.Envelope.Pagination.hasStartIndex && page.Envelope.Pagination.StartIndex < offset {
			return result, nil
		}
		items, err := decodeJSONArray(page.Envelope.Data)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNumber+1, err)
		}
		result.Pages = append(result.Pages, page)
		result.Items = append(result.Items, items...)
		result.Pagination = page.Envelope.Pagination

		if len(items) == 0 {
			return result, nil
		}
		nextOffset := offset + len(items)
		if page.Envelope.Pagination != nil {
			pagination := page.Envelope.Pagination
			if pagination.TotalResults > 0 && nextOffset >= pagination.TotalResults {
				return result, nil
			}
			if pagination.hasStartIndex && pagination.StartIndex >= offset {
				candidate := pagination.StartIndex + len(items)
				if candidate <= offset {
					return result, nil
				}
				nextOffset = candidate
			}
		}
		if len(items) < limit || nextOffset <= offset {
			return result, nil
		}
		offset = nextOffset
	}
	return nil, fmt.Errorf("pagination exceeded %d pages", maxPages)
}

func decodeJSONArray(data json.RawMessage) ([]json.RawMessage, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("response data is not an array: %w", err)
	}
	return items, nil
}

// CombinedEnvelope creates an envelope whose data is the collected raw array.
// It keeps unknown top-level fields from the first page and pagination metadata
// from the last page; all original envelopes remain available in Pages.
func (result *PaginatedResult) CombinedEnvelope() (Envelope, error) {
	if result == nil {
		return Envelope{}, fmt.Errorf("paginated result is nil")
	}
	data, err := json.Marshal(result.Items)
	if err != nil {
		return Envelope{}, fmt.Errorf("combine paginated data: %w", err)
	}
	combined := Envelope{Data: data, Pagination: result.Pagination}
	if len(result.Pages) != 0 {
		combined.Unknown = cloneRawMap(result.Pages[0].Envelope.Unknown)
		combined.Error = result.Pages[0].Envelope.Error
	}
	return combined, nil
}

func cloneRawMap(source map[string]json.RawMessage) map[string]json.RawMessage {
	if source == nil {
		return nil
	}
	clone := make(map[string]json.RawMessage, len(source))
	for key, value := range source {
		clone[key] = append(json.RawMessage(nil), value...)
	}
	return clone
}
