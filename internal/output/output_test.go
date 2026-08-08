package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Aayush9029/aads/internal/output"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"json", "yaml", "table"} {
		format, err := output.ParseFormat(value)
		if err != nil {
			t.Fatalf("ParseFormat(%q) returned an error: %v", value, err)
		}
		if string(format) != value {
			t.Fatalf("ParseFormat(%q) = %q", value, format)
		}
	}

	for _, value := range []string{"", "JSON", " csv ", "csv"} {
		_, err := output.ParseFormat(value)
		if err == nil {
			t.Fatalf("ParseFormat(%q) did not return an error", value)
		}
		if !strings.Contains(err.Error(), "json, yaml, or table") {
			t.Fatalf("ParseFormat(%q) error = %q", value, err)
		}
	}
}

func TestWriteJSONUsesDataByDefault(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{
		"pagination":{"totalResults":1},
		"data":{"z":"<campaign>","a":9007199254740993}
	}`)

	var got bytes.Buffer
	err := output.Write(&got, raw, output.Options{Format: output.FormatJSON})
	if err != nil {
		t.Fatal(err)
	}

	want := "{\n  \"a\": 9007199254740993,\n  \"z\": \"<campaign>\"\n}\n"
	if got.String() != want {
		t.Fatalf("JSON output:\n%s\nwant:\n%s", got.String(), want)
	}
}

func TestWriteJSONCanKeepEnvelope(t *testing.T) {
	t.Parallel()

	value := map[string]any{
		"pagination": map[string]any{"totalResults": 1},
		"data":       []any{map[string]any{"id": 42}},
	}

	encoded, err := output.Marshal(value, output.Options{
		Format:   output.FormatJSON,
		Envelope: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	want := "{\n  \"data\": [\n    {\n      \"id\": 42\n    }\n  ],\n  \"pagination\": {\n    \"totalResults\": 1\n  }\n}\n"
	if string(encoded) != want {
		t.Fatalf("JSON output:\n%s\nwant:\n%s", encoded, want)
	}
}

func TestWriteAcceptsStructAndRawMessagePointer(t *testing.T) {
	t.Parallel()

	type response struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}

	value := response{}
	value.Data.Name = "Search Results"
	encoded, err := output.Marshal(value, output.Options{Format: output.FormatJSON})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "{\n  \"name\": \"Search Results\"\n}\n" {
		t.Fatalf("struct output = %s", encoded)
	}

	raw := json.RawMessage(`{"data":[1,2]}`)
	encoded, err = output.Marshal(&raw, output.Options{Format: output.FormatJSON})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "[\n  1,\n  2\n]\n" {
		t.Fatalf("raw message output = %s", encoded)
	}
}

func TestWriteRejectsInvalidRawMessage(t *testing.T) {
	t.Parallel()

	for _, raw := range []json.RawMessage{
		json.RawMessage(`{"data":`),
		json.RawMessage(`{} {}`),
	} {
		var got bytes.Buffer
		err := output.Write(&got, raw, output.Options{Format: output.FormatJSON})
		if err == nil || !strings.Contains(err.Error(), "decode output") {
			t.Fatalf("Write(%q) error = %v", raw, err)
		}
	}
}

func TestYAMLIsStableAndKeepsTypes(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{
		"data": {
			"zero": null,
			"text": "001",
			"items": [{"z": true, "a": 1.25}],
			"count": 9007199254740993
		}
	}`)
	encoded, err := output.Marshal(raw, output.Options{Format: output.FormatYAML})
	if err != nil {
		t.Fatal(err)
	}

	want := "count: 9007199254740993\nitems:\n  - a: 1.25\n    z: true\ntext: \"001\"\nzero: null\n"
	if string(encoded) != want {
		t.Fatalf("YAML output:\n%s\nwant:\n%s", encoded, want)
	}

	again, err := output.Marshal(raw, output.Options{Format: output.FormatYAML})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, again) {
		t.Fatalf("YAML output changed:\n%s\nthen:\n%s", encoded, again)
	}
}

func TestTableFlattensNestedValues(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{
		"data": [
			{"id": 2, "name": "Second", "targeting": {"age": {"min": 18}}, "tags": ["one", "two"]},
			{"id": 1, "name": "First\nLine", "targeting": null, "details": {}}
		]
	}`)
	encoded, err := output.Marshal(raw, output.Options{Format: output.FormatTable})
	if err != nil {
		t.Fatal(err)
	}

	want := "DETAILS  ID  NAME         TAGS[0]  TAGS[1]  TARGETING  TARGETING.AGE.MIN\n" +
		"-------  --  -----------  -------  -------  ---------  -----------------\n" +
		"         2   Second       one      two                 18               \n" +
		"{}       1   First\\nLine                    null                        \n"
	if string(encoded) != want {
		t.Fatalf("table output:\n%q\nwant:\n%q", encoded, want)
	}
	if strings.Contains(string(encoded), "map[") {
		t.Fatalf("table contains Go map formatting: %s", encoded)
	}
	if strings.Contains(string(encoded), "\x1b[") {
		t.Fatalf("table contains terminal styling for a buffer: %q", encoded)
	}
}

func TestTableEscapesColumnNamesAndControlCharacters(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage("{\"data\":[{\"a.b\":\"ok\\u001b[31m\",\"a\":{\"b\":\"nested\"}}]}")
	encoded, err := output.Marshal(raw, output.Options{Format: output.FormatTable})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `A.B     A\.B`) {
		t.Fatalf("escaped columns are missing: %q", encoded)
	}
	if !strings.Contains(string(encoded), `ok\u001b[31m`) {
		t.Fatalf("escaped control character is missing: %q", encoded)
	}
	if strings.Contains(string(encoded), "\x1b[") {
		t.Fatalf("table contains injected terminal styling: %q", encoded)
	}
}

func TestEmptyTableHasClearOutput(t *testing.T) {
	t.Parallel()

	encoded, err := output.Marshal(json.RawMessage(`{"data":[]}`), output.Options{Format: output.FormatTable})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "No results.\n" {
		t.Fatalf("empty table = %q", encoded)
	}
}

func TestWriteReportsWriterError(t *testing.T) {
	t.Parallel()

	err := output.Write(errorWriter{}, map[string]any{"data": "value"}, output.Options{Format: output.FormatJSON})
	if err == nil || !strings.Contains(err.Error(), "write JSON output") {
		t.Fatalf("Write error = %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
