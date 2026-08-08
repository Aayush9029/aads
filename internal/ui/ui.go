package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
)

type UI struct {
	stderr io.Writer
	info   lipgloss.Style
	ok     lipgloss.Style
	warn   lipgloss.Style
	fail   lipgloss.Style
}

func New(stderr io.Writer) *UI {
	renderWriter := stderr
	if renderWriter == nil {
		renderWriter = io.Discard
	}
	renderer := lipgloss.NewRenderer(renderWriter)
	return &UI{
		stderr: stderr,
		info:   renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		ok:     renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("2")),
		warn:   renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("3")),
		fail:   renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("1")),
	}
}

func (ui *UI) Info(message string) error {
	return ui.write(ui.info, "Info:", message)
}

func (ui *UI) Success(message string) error {
	return ui.write(ui.ok, "Done:", message)
}

func (ui *UI) Warning(message string) error {
	return ui.write(ui.warn, "Warning:", message)
}

func (ui *UI) Error(message string) error {
	return ui.write(ui.fail, "Error:", message)
}

func (ui *UI) Infof(format string, values ...any) error {
	return ui.Info(fmt.Sprintf(format, values...))
}

func (ui *UI) Successf(format string, values ...any) error {
	return ui.Success(fmt.Sprintf(format, values...))
}

func (ui *UI) Warningf(format string, values ...any) error {
	return ui.Warning(fmt.Sprintf(format, values...))
}

func (ui *UI) Errorf(format string, values ...any) error {
	return ui.Error(fmt.Sprintf(format, values...))
}

func (ui *UI) write(style lipgloss.Style, prefix, message string) error {
	if ui == nil || ui.stderr == nil {
		return fmt.Errorf("diagnostic writer is nil")
	}
	_, err := fmt.Fprintf(ui.stderr, "%s %s\n", style.Render(prefix), message)
	if err != nil {
		return fmt.Errorf("write diagnostic: %w", err)
	}
	return nil
}
