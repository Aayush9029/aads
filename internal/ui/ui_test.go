package ui_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Aayush9029/aads/internal/ui"
)

func TestDiagnosticsUseTheConfiguredWriter(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	terminal := ui.New(&stderr)

	if err := terminal.Info("Fetching campaigns."); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Successf("Found %d campaigns.", 2); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Warning("The request is taking longer than expected."); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Errorf("Apple Ads returned status %d.", 500); err != nil {
		t.Fatal(err)
	}

	want := "Info: Fetching campaigns.\n" +
		"Done: Found 2 campaigns.\n" +
		"Warning: The request is taking longer than expected.\n" +
		"Error: Apple Ads returned status 500.\n"
	if stderr.String() != want {
		t.Fatalf("stderr:\n%q\nwant:\n%q", stderr.String(), want)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "\x1b[") {
		t.Fatalf("non-TTY diagnostics contain styling: %q", stderr.String())
	}
}

func TestNilDiagnosticWriterReturnsAnError(t *testing.T) {
	t.Parallel()

	err := ui.New(nil).Error("Request failed.")
	if err == nil || err.Error() != "diagnostic writer is nil" {
		t.Fatalf("Error() = %v", err)
	}
}

func TestDiagnosticWriterErrorsAreReturned(t *testing.T) {
	t.Parallel()

	err := ui.New(errorWriter{}).Warning("Try again.")
	if err == nil || !strings.Contains(err.Error(), "write diagnostic") {
		t.Fatalf("Warning() = %v", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
