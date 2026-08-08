package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Aayush9029/aads/internal/api"
	"github.com/Aayush9029/aads/internal/commands"
	"github.com/Aayush9029/aads/internal/config"
	"github.com/Aayush9029/aads/internal/output"
	"golang.org/x/term"
)

type Dependencies struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Version string

	Client     commands.Client
	IsTerminal func(io.Reader) bool
	LoadConfig func() (*config.Config, error)
	SaveConfig func(*config.Config) error
}

type globalFlags struct {
	Output   string
	Verbose  bool
	OrgID    string
	Fields   string
	Currency string
	Envelope bool
	Help     bool
	Version  bool
}

func Run(ctx context.Context, args []string, dependencies Dependencies) int {
	dependencies = defaults(dependencies)
	globals, remaining, err := parseGlobals(args)
	if err != nil {
		return fail(dependencies.Stderr, err, 2)
	}
	if globals.Version {
		fmt.Fprintln(dependencies.Stdout, versionLine(dependencies.Version))
		return 0
	}
	if len(remaining) == 0 {
		if globals.Help {
			printRootHelp(dependencies.Stdout)
			return 0
		}
		printRootHelp(dependencies.Stderr)
		return 2
	}
	if remaining[0] == "help" {
		switch len(remaining) {
		case 1:
			printRootHelp(dependencies.Stdout)
			return 0
		case 2:
			if len(commands.ForGroup(remaining[1])) == 0 {
				return fail(dependencies.Stderr, fmt.Errorf("unknown command %q", remaining[1]), 2)
			}
			printGroupHelp(dependencies.Stdout, remaining[1])
			return 0
		case 3:
			command, ok := commands.Lookup(remaining[1], remaining[2])
			if !ok {
				return fail(dependencies.Stderr, fmt.Errorf("unknown command %q for %s", remaining[2], remaining[1]), 2)
			}
			printCommandHelp(dependencies.Stdout, command)
			return 0
		default:
			return fail(dependencies.Stderr, errors.New("help accepts a command and optional action"), 2)
		}
	}
	if remaining[0] == "version" {
		if len(remaining) != 1 {
			return fail(dependencies.Stderr, errors.New("version does not accept arguments"), 2)
		}
		fmt.Fprintln(dependencies.Stdout, versionLine(dependencies.Version))
		return 0
	}
	if remaining[0] == "configure" {
		if globals.Help {
			printConfigureHelp(dependencies.Stdout)
			return 0
		}
		if len(remaining) != 1 {
			return fail(dependencies.Stderr, errors.New("configure does not accept arguments"), 2)
		}
		if err := configure(dependencies); err != nil {
			return fail(dependencies.Stderr, err, 1)
		}
		return 0
	}

	group := remaining[0]
	if len(remaining) >= 2 && remaining[1] == "help" {
		if len(commands.ForGroup(group)) == 0 {
			return fail(dependencies.Stderr, fmt.Errorf("unknown command %q", group), 2)
		}
		printGroupHelp(dependencies.Stdout, group)
		return 0
	}
	if len(remaining) == 1 || globals.Help && len(remaining) < 2 {
		if len(commands.ForGroup(group)) == 0 {
			return fail(dependencies.Stderr, fmt.Errorf("unknown command %q", group), 2)
		}
		printGroupHelp(dependencies.Stdout, group)
		return 0
	}
	name := remaining[1]
	command, ok := commands.Lookup(group, name)
	if !ok {
		if len(commands.ForGroup(group)) == 0 {
			return fail(dependencies.Stderr, fmt.Errorf("unknown command %q", group), 2)
		}
		return fail(dependencies.Stderr, fmt.Errorf("unknown command %q for %s", name, group), 2)
	}
	if globals.Help {
		printCommandHelp(dependencies.Stdout, command)
		return 0
	}
	values, err := parseCommandFlags(remaining[2:], command)
	if err != nil {
		return fail(dependencies.Stderr, err, 2)
	}
	if err := commands.ValidateFlags(command, values); err != nil {
		return fail(dependencies.Stderr, err, 2)
	}
	if globals.Fields != "" && !supportsFields(command) {
		return fail(dependencies.Stderr, fmt.Errorf("--fields is not supported by %s", command.Key()), 2)
	}
	format, err := output.ParseFormat(globals.Output)
	if err != nil {
		return fail(dependencies.Stderr, err, 2)
	}

	cfg, err := dependencies.LoadConfig()
	if err != nil {
		return fail(dependencies.Stderr, fmt.Errorf("load config: %w", err), 1)
	}
	if globals.OrgID != "" {
		cfg.OrgID = globals.OrgID
	}
	if command.Group != "acls" && cfg.OrgID != "" {
		if err := positiveOrgID(cfg.OrgID); err != nil {
			return fail(dependencies.Stderr, err, 2)
		}
	}
	if globals.Currency == "" {
		globals.Currency = cfg.DefaultCurrency
	}
	client := dependencies.Client
	if globals.Currency == "" && needsCurrency(values) {
		if values.Bool("dry-run") {
			return fail(dependencies.Stderr, errors.New("dry-run needs --currency or a configured default when money flags are used"), 2)
		}
		if client == nil {
			if err := cfg.ValidateAuth(); err != nil {
				return fail(dependencies.Stderr, fmt.Errorf("invalid config: %w; run 'aads configure'", err), 1)
			}
			client, err = newAPIClient(cfg, globals.Verbose, dependencies.Stderr)
			if err != nil {
				return fail(dependencies.Stderr, err, 1)
			}
		}
		globals.Currency, err = inferCurrency(ctx, client, cfg.OrgID)
		if err != nil {
			return fail(dependencies.Stderr, err, 1)
		}
	}
	reader := &jsonInputReader{stdin: dependencies.Stdin}
	request, err := command.Build(values, commands.Globals{Fields: globals.Fields, Currency: strings.ToUpper(globals.Currency)}, reader)
	if err != nil {
		return fail(dependencies.Stderr, err, 2)
	}
	if values.Bool("dry-run") {
		preview := requestPreview(request, cfg.OrgID)
		if err := output.Write(dependencies.Stdout, preview, output.Options{Format: format, Envelope: true}); err != nil {
			return fail(dependencies.Stderr, err, 1)
		}
		return 0
	}
	if command.Delete && !values.Bool("yes") {
		if !dependencies.IsTerminal(dependencies.Stdin) {
			return fail(dependencies.Stderr, errors.New("delete commands require --yes when stdin is not a terminal"), 2)
		}
		confirmed, err := confirm(dependencies.Stdin, dependencies.Stderr, command.Summary)
		if err != nil {
			return fail(dependencies.Stderr, err, 1)
		}
		if !confirmed {
			fmt.Fprintln(dependencies.Stderr, "Cancelled.")
			return 0
		}
	}

	if client == nil {
		if request.OrgContext {
			if err := cfg.Validate(); err != nil {
				return fail(dependencies.Stderr, fmt.Errorf("invalid config: %w; run 'aads configure'", err), 1)
			}
		} else if err := cfg.ValidateAuth(); err != nil {
			return fail(dependencies.Stderr, fmt.Errorf("invalid config: %w; run 'aads configure'", err), 1)
		}
		client, err = newAPIClient(cfg, globals.Verbose, dependencies.Stderr)
		if err != nil {
			return fail(dependencies.Stderr, err, 1)
		}
	}
	result, err := client.Do(ctx, request)
	if err != nil {
		return fail(dependencies.Stderr, err, 1)
	}
	value := result.Data
	if globals.Envelope {
		value = result.Envelope
	}
	if value == nil && command.SuccessText != "" {
		value = map[string]any{"status": command.SuccessText}
	}
	if err := output.Write(dependencies.Stdout, value, output.Options{Format: format, Envelope: true}); err != nil {
		return fail(dependencies.Stderr, err, 1)
	}
	return 0
}

func supportsFields(command commands.Command) bool {
	for _, flag := range command.Flags {
		if flag.Name == "_allow-fields" {
			return true
		}
	}
	return false
}

func newAPIClient(cfg *config.Config, verbose bool, stderr io.Writer) (commands.Client, error) {
	options := []api.Option{}
	if verbose {
		options = append(options, api.WithVerbose(stderr))
	}
	client, err := api.NewClient(cfg, options...)
	if err != nil {
		return nil, err
	}
	return &apiAdapter{client: client}, nil
}

func needsCurrency(values commands.Values) bool {
	if values.Get("from-json") != "" {
		return false
	}
	for _, name := range []string{"daily-budget", "default-bid", "bid"} {
		if values.Get(name) != "" {
			return true
		}
	}
	return false
}

func inferCurrency(ctx context.Context, client commands.Client, orgID string) (string, error) {
	result, err := client.Do(ctx, commands.Request{Method: "GET", Path: "/acls", Operation: "acls.list"})
	if err != nil {
		return "", fmt.Errorf("infer currency from ACLs: %w", err)
	}
	value := result.Data
	if value == nil {
		value = result.Envelope
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("read ACL currencies: %w", err)
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return "", fmt.Errorf("read ACL currencies: %w", err)
	}
	if object, ok := decoded.(map[string]any); ok {
		decoded = object["data"]
	}
	items, ok := decoded.([]any)
	if !ok {
		return "", errors.New("infer currency from ACLs: response data is not an array")
	}
	currencies := map[string]bool{}
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		currency, _ := object["currency"].(string)
		currency = strings.ToUpper(currency)
		if currency == "" {
			continue
		}
		currencies[currency] = true
		if fmt.Sprint(object["orgId"]) == orgID {
			return currency, nil
		}
	}
	if len(currencies) == 1 {
		for currency := range currencies {
			return currency, nil
		}
	}
	return "", errors.New("unable to determine organization currency; pass --currency or configure a default")
}

func defaults(dependencies Dependencies) Dependencies {
	if dependencies.Stdin == nil {
		dependencies.Stdin = os.Stdin
	}
	if dependencies.Stdout == nil {
		dependencies.Stdout = os.Stdout
	}
	if dependencies.Stderr == nil {
		dependencies.Stderr = os.Stderr
	}
	if dependencies.Version == "" {
		dependencies.Version = "dev"
	}
	if dependencies.IsTerminal == nil {
		dependencies.IsTerminal = isTerminal
	}
	if dependencies.LoadConfig == nil {
		dependencies.LoadConfig = config.Load
	}
	if dependencies.SaveConfig == nil {
		dependencies.SaveConfig = config.Save
	}
	return dependencies
}

func fail(writer io.Writer, err error, code int) int {
	fmt.Fprintf(writer, "aads: %v\n", err)
	return code
}

func versionLine(version string) string { return "aads " + version }

func isTerminal(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func confirm(reader io.Reader, writer io.Writer, summary string) (bool, error) {
	fmt.Fprintf(writer, "%s? [y/N] ", summary)
	line, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func requestPreview(request commands.Request, orgID string) map[string]any {
	path := request.Path
	if encoded := request.Query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	headers := map[string]any{"Authorization": "Bearer [redacted]"}
	if request.OrgContext && orgID != "" {
		headers["X-AP-Context"] = "[redacted]"
	}
	return map[string]any{"method": request.Method, "path": path, "headers": headers, "body": request.Body}
}

type jsonInputReader struct{ stdin io.Reader }

func (reader *jsonInputReader) Read(source string) (any, error) {
	var input io.Reader
	switch {
	case source == "@-" || source == "-":
		input = reader.stdin
	case strings.HasPrefix(source, "@"):
		file, err := os.Open(strings.TrimPrefix(source, "@"))
		if err != nil {
			return nil, fmt.Errorf("read JSON: %w", err)
		}
		defer file.Close()
		input = file
	default:
		input = strings.NewReader(source)
	}
	decoder := json.NewDecoder(input)
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("parse JSON: multiple values")
		}
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	return value, nil
}

func parseGlobals(args []string) (globalFlags, []string, error) {
	flags := globalFlags{Output: "json"}
	var remaining []string
	for index := 0; index < len(args); index++ {
		arg := args[index]
		name, inline, hasInline := splitFlag(arg)
		switch name {
		case "--output", "-o":
			value, next, err := flagValue(name, inline, hasInline, args, index)
			if err != nil {
				return flags, nil, err
			}
			flags.Output = value
			index = next
		case "--org-id":
			value, next, err := flagValue(name, inline, hasInline, args, index)
			if err != nil {
				return flags, nil, err
			}
			flags.OrgID = value
			index = next
		case "--fields":
			value, next, err := flagValue(name, inline, hasInline, args, index)
			if err != nil {
				return flags, nil, err
			}
			flags.Fields = value
			index = next
		case "--currency":
			value, next, err := flagValue(name, inline, hasInline, args, index)
			if err != nil {
				return flags, nil, err
			}
			flags.Currency = value
			index = next
		case "--verbose", "-v":
			if hasInline {
				value, err := parseBool(name, inline)
				if err != nil {
					return flags, nil, err
				}
				flags.Verbose = value
			} else {
				flags.Verbose = true
			}
		case "--envelope":
			if hasInline {
				value, err := parseBool(name, inline)
				if err != nil {
					return flags, nil, err
				}
				flags.Envelope = value
			} else {
				flags.Envelope = true
			}
		case "--help", "-h":
			flags.Help = true
		case "--version":
			flags.Version = true
		default:
			remaining = append(remaining, arg)
		}
	}
	return flags, remaining, nil
}

func parseCommandFlags(args []string, command commands.Command) (commands.Values, error) {
	definitions := map[string]commands.Flag{}
	for _, flag := range command.Flags {
		definitions[flag.Name] = flag
	}
	values := commands.Values{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("unexpected argument %q", arg)
		}
		name, inline, hasInline := splitFlag(arg)
		key := strings.TrimPrefix(name, "--")
		definition, ok := definitions[key]
		if !ok {
			return nil, fmt.Errorf("unknown flag %s for %s", name, command.Key())
		}
		if strings.HasPrefix(key, "_") {
			return nil, fmt.Errorf("unknown flag %s for %s", name, command.Key())
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf("flag %s was provided more than once", name)
		}
		if definition.Boolean {
			if hasInline {
				parsed, err := parseBool(name, inline)
				if err != nil {
					return nil, err
				}
				values[key] = fmt.Sprint(parsed)
			} else {
				values[key] = "true"
			}
			continue
		}
		value, next, err := flagValue(name, inline, hasInline, args, index)
		if err != nil {
			return nil, err
		}
		values[key] = value
		index = next
	}
	return values, nil
}

func splitFlag(arg string) (string, string, bool) {
	if index := strings.IndexByte(arg, '='); index >= 0 {
		return arg[:index], arg[index+1:], true
	}
	return arg, "", false
}

func flagValue(name, inline string, hasInline bool, args []string, index int) (string, int, error) {
	if hasInline {
		if inline == "" {
			return "", index, fmt.Errorf("%s needs a value", name)
		}
		return inline, index, nil
	}
	if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") || args[index+1] == "-v" || args[index+1] == "-h" || args[index+1] == "-o" {
		return "", index, fmt.Errorf("%s needs a value", name)
	}
	return args[index+1], index + 1, nil
}

func parseBool(name, value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", name)
	}
}

func printRootHelp(writer io.Writer) {
	fmt.Fprintln(writer, "Apple Ads Campaign Management API v5 client")
	fmt.Fprintln(writer, "\nUsage:\n  aads <command> <action> [flags]")
	fmt.Fprintln(writer, "\nCommands:")
	groups := append(commands.Groups(), "configure", "version")
	sort.Strings(groups)
	for _, group := range groups {
		fmt.Fprintf(writer, "  %s\n", group)
	}
	fmt.Fprintln(writer, "\nGlobal flags:\n  -o, --output <json|table|yaml>\n  -v, --verbose\n      --org-id <id>\n      --fields <fields>\n      --currency <code>\n      --envelope\n  -h, --help\n      --version")
}

func printGroupHelp(writer io.Writer, group string) {
	fmt.Fprintf(writer, "Usage:\n  aads %s <action> [flags]\n\nActions:\n", group)
	for _, command := range commands.ForGroup(group) {
		fmt.Fprintf(writer, "  %-20s %s\n", command.Name, command.Summary)
	}
}

func printCommandHelp(writer io.Writer, command commands.Command) {
	fmt.Fprintf(writer, "%s\n\nUsage:\n  aads %s %s [flags]\n", command.Summary, command.Group, command.Name)
	visible := 0
	for _, flag := range command.Flags {
		if !strings.HasPrefix(flag.Name, "_") {
			visible++
		}
	}
	if visible == 0 {
		return
	}
	fmt.Fprintln(writer, "\nFlags:")
	for _, flag := range command.Flags {
		if strings.HasPrefix(flag.Name, "_") {
			continue
		}
		suffix := " <value>"
		if flag.Boolean {
			suffix = ""
		}
		details := flag.Description
		if flag.Default != "" {
			details += " (default " + flag.Default + ")"
		}
		if flag.Required {
			details += " (required)"
		}
		fmt.Fprintf(writer, "  --%-26s %s\n", flag.Name+suffix, details)
	}
}

func printConfigureHelp(writer io.Writer) {
	fmt.Fprintln(writer, "Save Apple Ads credentials\n\nUsage:\n  aads configure")
}

func configure(dependencies Dependencies) error {
	cfg, err := dependencies.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	reader := bufio.NewReader(dependencies.Stdin)
	fields := []struct {
		label string
		value *string
	}{
		{"Client ID", &cfg.ClientID}, {"Team ID", &cfg.TeamID}, {"Key ID", &cfg.KeyID}, {"Organization ID", &cfg.OrgID},
		{"Private key path", &cfg.PrivateKeyPath}, {"Default currency", &cfg.DefaultCurrency},
	}
	for _, field := range fields {
		if *field.value == "" {
			fmt.Fprintf(dependencies.Stderr, "%s: ", field.label)
		} else {
			fmt.Fprintf(dependencies.Stderr, "%s [%s]: ", field.label, *field.value)
		}
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fmt.Errorf("read %s: %w", strings.ToLower(field.label), readErr)
		}
		if value := strings.TrimSpace(line); value != "" {
			*field.value = value
		}
		if errors.Is(readErr, io.EOF) && line == "" {
			return fmt.Errorf("read %s: input ended", strings.ToLower(field.label))
		}
	}
	cfg.DefaultCurrency = strings.ToUpper(strings.TrimSpace(cfg.DefaultCurrency))
	if err := positiveOrgID(cfg.OrgID); err != nil {
		return err
	}
	if cfg.DefaultCurrency != "" {
		if err := validCurrency(cfg.DefaultCurrency); err != nil {
			return err
		}
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if err := dependencies.SaveConfig(cfg); err != nil {
		return err
	}
	fmt.Fprintln(dependencies.Stdout, "Configuration saved.")
	return nil
}

func positiveOrgID(value string) error {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return errors.New("organization ID must be a positive integer")
	}
	return nil
}

func validCurrency(value string) error {
	if len(value) != 3 {
		return errors.New("default currency must be a three-letter ISO code")
	}
	for _, ch := range value {
		if ch < 'A' || ch > 'Z' {
			return errors.New("default currency must be a three-letter ISO code")
		}
	}
	return nil
}

func marshalEnvelope(envelope api.Envelope) (json.RawMessage, error) {
	value := map[string]any{"data": envelope.Data}
	if envelope.Pagination != nil {
		value["pagination"] = envelope.Pagination
	}
	if envelope.Error != nil {
		value["error"] = envelope.Error
	}
	for key, raw := range envelope.Unknown {
		value[key] = raw
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buffer.Bytes()), nil
}
