package command

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Command interface {
	Name() string
	Description() string
	Run(ctx context.Context, args []string) error
}

type Registry struct {
	commands map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

func (r *Registry) Register(cmd Command) error {
	if r == nil {
		return errors.New("command registry is nil")
	}
	if cmd == nil {
		return errors.New("command is required")
	}
	name := strings.TrimSpace(cmd.Name())
	if name == "" {
		return errors.New("command name is required")
	}
	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command %q already registered", name)
	}
	r.commands[name] = cmd
	return nil
}

func (r *Registry) Run(ctx context.Context, args []string) error {
	if r == nil {
		return errors.New("command registry is nil")
	}
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return errors.New("command name is required")
	}
	name := strings.TrimSpace(args[0])
	cmd, ok := r.commands[name]
	if !ok {
		return fmt.Errorf("unknown command %q", name)
	}
	return cmd.Run(ctx, args[1:])
}

func (r *Registry) Usage() string {
	if r == nil || len(r.commands) == 0 {
		return "No commands registered."
	}
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("Available commands:\n")
	for _, name := range names {
		b.WriteString("- ")
		b.WriteString(name)
		desc := strings.TrimSpace(r.commands[name].Description())
		if desc != "" {
			b.WriteString(": ")
			b.WriteString(desc)
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
