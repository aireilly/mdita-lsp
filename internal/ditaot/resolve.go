package ditaot

import (
	"fmt"
	"os"
	"os/exec"
)

func ResolveDitaPath(configured string) (string, error) {
	if configured != "" {
		info, err := os.Stat(configured)
		if err != nil {
			return "", fmt.Errorf("configured dita path not found: %s", configured)
		}
		if info.IsDir() {
			return "", fmt.Errorf("configured dita path is a directory: %s", configured)
		}
		return configured, nil
	}
	path, err := exec.LookPath("dita")
	if err != nil {
		return "", fmt.Errorf("dita binary not found: install DITA OT and add to $PATH, or set build.dita_ot.dita_path in config")
	}
	return path, nil
}
