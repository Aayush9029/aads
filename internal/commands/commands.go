package commands

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Client is the small API surface the command runner needs. The HTTP client
// implements it, while tests can use a recording client.
type Client interface {
	Do(context.Context, Request) (Result, error)
}

type Request struct {
	Method     string
	Path       string
	Query      url.Values
	Body       any
	Operation  string
	OrgContext bool
	Mutation   bool
	AllPages   bool
}

type Result struct {
	Data     any
	Envelope any
}

type Flag struct {
	Name        string
	Description string
	Default     string
	Boolean     bool
	Required    bool
}

type Command struct {
	Group       string
	Name        string
	Summary     string
	Flags       []Flag
	Build       func(Values, Globals, InputReader) (Request, error)
	Delete      bool
	Mutation    bool
	SuccessText string
}

type Globals struct {
	Fields   string
	Currency string
}

type InputReader interface {
	Read(string) (any, error)
}

type Values map[string]string

func (v Values) Get(name string) string { return v[name] }

func (v Values) Bool(name string) bool { return v[name] == "true" }

func (c Command) Key() string { return c.Group + " " + c.Name }

func Registry() []Command {
	commands := registry()
	sort.Slice(commands, func(i, j int) bool { return commands[i].Key() < commands[j].Key() })
	return commands
}

func Lookup(group, name string) (Command, bool) {
	for _, command := range registry() {
		if command.Group == group && command.Name == name {
			return command, true
		}
	}
	return Command{}, false
}

func Groups() []string {
	seen := map[string]bool{}
	var groups []string
	for _, command := range Registry() {
		if !seen[command.Group] {
			seen[command.Group] = true
			groups = append(groups, command.Group)
		}
	}
	return groups
}

func ForGroup(group string) []Command {
	var found []Command
	for _, command := range Registry() {
		if command.Group == group {
			found = append(found, command)
		}
	}
	return found
}

func ValidateFlags(command Command, values Values) error {
	allowed := make(map[string]Flag, len(command.Flags))
	for _, flag := range command.Flags {
		allowed[flag.Name] = flag
		if _, ok := values[flag.Name]; !ok && flag.Default != "" {
			values[flag.Name] = flag.Default
		}
	}
	for name := range values {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("unknown flag --%s for %s", name, command.Key())
		}
	}
	for _, flag := range command.Flags {
		if flag.Required && strings.TrimSpace(values[flag.Name]) == "" {
			return fmt.Errorf("--%s is required", flag.Name)
		}
	}
	return nil
}
