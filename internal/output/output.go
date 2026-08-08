package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
	FormatTable Format = "table"
)

type Options struct {
	Format   Format
	Envelope bool
}

func ParseFormat(value string) (Format, error) {
	format := Format(value)
	if err := format.Validate(); err != nil {
		return "", err
	}
	return format, nil
}

func (f Format) Validate() error {
	switch f {
	case FormatJSON, FormatYAML, FormatTable:
		return nil
	case "":
		return errors.New("output format is required; use json, yaml, or table")
	default:
		return fmt.Errorf("unsupported output format %q; use json, yaml, or table", f)
	}
}

func Write(writer io.Writer, value any, options Options) error {
	if writer == nil {
		return errors.New("output writer is nil")
	}
	if err := options.Format.Validate(); err != nil {
		return err
	}

	normalized, err := normalize(value)
	if err != nil {
		return err
	}
	if !options.Envelope {
		normalized = dataValue(normalized)
	}

	switch options.Format {
	case FormatJSON:
		return writeJSON(writer, normalized)
	case FormatYAML:
		return writeYAML(writer, normalized)
	case FormatTable:
		return writeTable(writer, normalized)
	default:
		return options.Format.Validate()
	}
}

func Marshal(value any, options Options) ([]byte, error) {
	var buffer bytes.Buffer
	if err := Write(&buffer, value, options); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func normalize(value any) (any, error) {
	switch raw := value.(type) {
	case json.RawMessage:
		return decodeJSON(raw)
	case *json.RawMessage:
		if raw == nil {
			return nil, nil
		}
		return decodeJSON(*raw)
	case []byte:
		return decodeJSON(raw)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode output: %w", err)
	}
	return decodeJSON(encoded)
}

func decodeJSON(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode output: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("decode output: multiple JSON values")
		}
		return nil, fmt.Errorf("decode output: %w", err)
	}
	return value, nil
}

func dataValue(value any) any {
	object, ok := value.(map[string]any)
	if !ok {
		return value
	}
	data, ok := object["data"]
	if !ok {
		return value
	}
	return data
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON output: %w", err)
	}
	return nil
}

func writeYAML(writer io.Writer, value any) error {
	node, err := yamlNode(value)
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return fmt.Errorf("write YAML output: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("finish YAML output: %w", err)
	}
	return nil
}

func yamlNode(value any) (*yaml.Node, error) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, key := range keys {
			child, err := yamlNode(typed[key])
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				child,
			)
		}
		return node, nil
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			child, err := yamlNode(item)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return node, nil
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: typed}, nil
	case json.Number:
		tag := "!!int"
		if strings.ContainsAny(string(typed), ".eE") {
			tag = "!!float"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: string(typed)}, nil
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(typed)}, nil
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	default:
		return nil, fmt.Errorf("write YAML output: unsupported value of type %T", value)
	}
}

func writeTable(writer io.Writer, value any) error {
	headings, rows := tableRows(value)
	if len(rows) == 0 {
		_, err := io.WriteString(writer, "No results.\n")
		if err != nil {
			return fmt.Errorf("write table output: %w", err)
		}
		return nil
	}

	renderer := lipgloss.NewRenderer(writer)
	headerStyle := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	separatorStyle := renderer.NewStyle().Faint(true)

	widths := make([]int, len(headings))
	for index, heading := range headings {
		widths[index] = lipgloss.Width(heading)
	}
	for _, row := range rows {
		for index, heading := range headings {
			width := lipgloss.Width(row[heading])
			if width > widths[index] {
				widths[index] = width
			}
		}
	}

	var table strings.Builder
	for index, heading := range headings {
		if index > 0 {
			table.WriteString("  ")
		}
		table.WriteString(pad(headerStyle.Render(strings.ToUpper(heading)), widths[index]))
	}
	table.WriteByte('\n')
	for index, width := range widths {
		if index > 0 {
			table.WriteString("  ")
		}
		table.WriteString(separatorStyle.Render(strings.Repeat("-", width)))
	}
	table.WriteByte('\n')
	for _, row := range rows {
		for index, heading := range headings {
			if index > 0 {
				table.WriteString("  ")
			}
			table.WriteString(pad(row[heading], widths[index]))
		}
		table.WriteByte('\n')
	}

	if _, err := io.WriteString(writer, table.String()); err != nil {
		return fmt.Errorf("write table output: %w", err)
	}
	return nil
}

func tableRows(value any) ([]string, []map[string]string) {
	values, ok := value.([]any)
	if !ok {
		values = []any{value}
	}

	rows := make([]map[string]string, 0, len(values))
	columnSet := make(map[string]struct{})
	for _, item := range values {
		row := make(map[string]string)
		flatten(row, "", item)
		if len(row) == 0 {
			row["value"] = compactJSON(item)
		}
		for column := range row {
			columnSet[column] = struct{}{}
		}
		rows = append(rows, row)
	}

	columns := make([]string, 0, len(columnSet))
	for column := range columnSet {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	return columns, rows
}

func flatten(row map[string]string, path string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			row[columnName(path)] = "{}"
			return
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			flatten(row, joinPath(path, escapeSegment(key)), typed[key])
		}
	case []any:
		if len(typed) == 0 {
			row[columnName(path)] = "[]"
			return
		}
		for index, item := range typed {
			flatten(row, path+"["+strconv.Itoa(index)+"]", item)
		}
	default:
		row[columnName(path)] = cellValue(typed)
	}
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func escapeSegment(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, ".", "\\.")
}

func columnName(path string) string {
	if path == "" {
		return "value"
	}
	return path
}

func cellValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return sanitizeCell(typed)
	case json.Number:
		return string(typed)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return sanitizeCell(compactJSON(typed))
	}
}

func compactJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func sanitizeCell(value string) string {
	var clean strings.Builder
	for len(value) > 0 {
		r, size := utf8.DecodeRuneInString(value)
		value = value[size:]
		switch r {
		case '\n':
			clean.WriteString(`\n`)
		case '\r':
			clean.WriteString(`\r`)
		case '\t':
			clean.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&clean, `\u%04x`, r)
				continue
			}
			clean.WriteRune(r)
		}
	}
	return clean.String()
}

func pad(value string, width int) string {
	padding := width - lipgloss.Width(value)
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}
