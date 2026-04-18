package ditaot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTryAcquireRelease(t *testing.T) {
	b := &Builder{}
	if !b.TryAcquire() {
		t.Fatal("first TryAcquire should succeed")
	}
	if b.TryAcquire() {
		t.Fatal("second TryAcquire should fail while acquired")
	}
	b.Release()
	if !b.TryAcquire() {
		t.Fatal("TryAcquire should succeed after Release")
	}
	b.Release()
}

func TestRunSuccess(t *testing.T) {
	b := &Builder{}
	result, err := b.Run(context.Background(), "echo", "hello", "xhtml", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Elapsed <= 0 {
		t.Error("expected positive elapsed time")
	}
}

func TestRunFailure(t *testing.T) {
	b := &Builder{}
	result, err := b.Run(context.Background(), "false", "input.mditamap", "xhtml", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
}

func TestRunCapturesOutput(t *testing.T) {
	b := &Builder{}
	result, err := b.Run(context.Background(), "echo", "test-output", "xhtml", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output == "" {
		t.Error("expected captured output")
	}
}

func TestIntegrationBuild(t *testing.T) {
	ditaPath := os.Getenv("DITA_OT_PATH")
	if ditaPath == "" {
		t.Skip("DITA_OT_PATH not set, skipping integration test")
	}

	ditaBin := filepath.Join(ditaPath, "bin", "dita")
	tmp := t.TempDir()

	mapContent := "# Test Map\n\n- [Topic](topic.md)\n"
	topicContent := "---\n$schema: \"urn:oasis:names:tc:mdita:rng:topic.rng\"\n---\n\n# Test Topic\n\nHello world.\n"

	if err := os.WriteFile(filepath.Join(tmp, "test.mditamap"), []byte(mapContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "topic.md"), []byte(topicContent), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmp, "out")
	b := &Builder{}
	result, err := b.Run(context.Background(), ditaBin, filepath.Join(tmp, "test.mditamap"), "xhtml", outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("build failed, output:\n%s", result.Output)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("cannot read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected output files in out dir")
	}
}
