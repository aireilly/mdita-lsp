package ditaot

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveDitaPath_Configured(t *testing.T) {
	tmp := t.TempDir()
	fakeDita := filepath.Join(tmp, "dita")
	if runtime.GOOS == "windows" {
		fakeDita += ".exe"
	}
	if err := os.WriteFile(fakeDita, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveDitaPath(fakeDita)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fakeDita {
		t.Errorf("got %q, want %q", got, fakeDita)
	}
}

func TestResolveDitaPath_ConfiguredNotFound(t *testing.T) {
	_, err := ResolveDitaPath("/nonexistent/dita")
	if err == nil {
		t.Fatal("expected error for missing configured path")
	}
}

func TestResolveDitaPath_EmptyFallbackNotOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := ResolveDitaPath("")
	if err == nil {
		t.Fatal("expected error when dita not on PATH")
	}
}
