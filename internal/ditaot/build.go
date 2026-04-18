package ditaot

import (
	"bytes"
	"context"
	"os/exec"
	"sync"
	"time"
)

type Builder struct {
	mu       sync.Mutex
	building bool
}

type BuildResult struct {
	Success bool
	Output  string
	Elapsed time.Duration
}

func (b *Builder) TryAcquire() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.building {
		return false
	}
	b.building = true
	return true
}

func (b *Builder) Release() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.building = false
}

func (b *Builder) Run(ctx context.Context, ditaPath, mapPath, format, outputDir string) (*BuildResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, ditaPath, "--input="+mapPath, "--format="+format, "--output="+outputDir)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	elapsed := time.Since(start)

	result := &BuildResult{
		Output:  buf.String(),
		Elapsed: elapsed,
	}

	if err != nil {
		result.Success = false
		return result, nil
	}
	result.Success = true
	return result, nil
}
