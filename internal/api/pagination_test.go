package api

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func paginatedResponse(t *testing.T, body string) *Response {
	t.Helper()
	response := &Response{Body: json.RawMessage(body)}
	if err := json.Unmarshal(response.Body, &response.Envelope); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestCollectPagesPreservesOffsetRawItemsAndUnknownFields(t *testing.T) {
	var offsets []int
	result, err := CollectPages(context.Background(), PaginationOptions{Offset: 7, Limit: 2}, func(_ context.Context, limit, offset int) (*Response, error) {
		if limit != 2 {
			t.Fatalf("limit = %d", limit)
		}
		offsets = append(offsets, offset)
		switch offset {
		case 7:
			return paginatedResponse(t, `{"data":[{"id":7,"future":"a"},{"id":8}],"pagination":{"totalResults":10,"startIndex":7,"itemsPerPage":2,"cursor":"one"},"meta":{"request":"x"}}`), nil
		case 9:
			return paginatedResponse(t, `{"data":[{"id":9,"future":"b"}],"pagination":{"totalResults":10,"startIndex":9,"itemsPerPage":1,"cursor":"two"}}`), nil
		default:
			t.Fatalf("unexpected offset %d", offset)
			return nil, nil
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{7, 9}; !reflect.DeepEqual(offsets, want) {
		t.Fatalf("offsets = %v, want %v", offsets, want)
	}
	if len(result.Items) != 3 || !json.Valid(result.Items[0]) || string(result.Items[2]) != `{"id":9,"future":"b"}` {
		t.Fatalf("raw items = %q", result.Items)
	}
	combined, err := result.CombinedEnvelope()
	if err != nil {
		t.Fatal(err)
	}
	if string(combined.Unknown["meta"]) != `{"request":"x"}` || combined.Pagination.StartIndex != 9 {
		t.Fatalf("combined envelope = %#v", combined)
	}
	var items []map[string]any
	if err := json.Unmarshal(combined.Data, &items); err != nil || len(items) != 3 {
		t.Fatalf("combined data: %v, %#v", err, items)
	}
}

func TestCollectPagesStopsBeforeRepeatedServerPage(t *testing.T) {
	var calls int
	result, err := CollectPages(context.Background(), PaginationOptions{Limit: 1}, func(_ context.Context, _, offset int) (*Response, error) {
		calls++
		if offset == 0 {
			return paginatedResponse(t, `{"data":[{"id":1}],"pagination":{"startIndex":0,"itemsPerPage":1}}`), nil
		}
		return paginatedResponse(t, `{"data":[{"id":1}],"pagination":{"startIndex":0,"itemsPerPage":1}}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(result.Items) != 1 || len(result.Pages) != 1 {
		t.Fatalf("calls=%d items=%d pages=%d", calls, len(result.Items), len(result.Pages))
	}
}

func TestCollectPagesStopsOnShortPageWithoutMetadata(t *testing.T) {
	result, err := CollectPages(context.Background(), PaginationOptions{Limit: 3}, func(context.Context, int, int) (*Response, error) {
		return paginatedResponse(t, `{"data":[1,2]}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 || len(result.Pages) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCollectPagesHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := CollectPages(ctx, PaginationOptions{}, func(context.Context, int, int) (*Response, error) {
		t.Fatal("fetch should not be called")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestCollectPagesRejectsObjectData(t *testing.T) {
	_, err := CollectPages(context.Background(), PaginationOptions{}, func(context.Context, int, int) (*Response, error) {
		return paginatedResponse(t, `{"data":{"row":[]}}`), nil
	})
	if err == nil {
		t.Fatal("expected non-array data error")
	}
}
