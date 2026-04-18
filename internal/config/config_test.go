package config

import (
	"testing"
)

func TestParseYAML(t *testing.T) {
	input := `
core:
  markdown:
    file_extensions: [md, markdown, mditamap]
    text_sync: incremental
    title_from_heading: false
  mdita:
    enable: true
    map_extensions: [mditamap, ditamap]
completion:
  max_candidates: 100
code_actions:
  toc:
    enable: false
    include_levels: [1, 2, 3]
  create_missing_file:
    enable: true
diagnostics:
  mdita_compliance: true
  ditamap_validation: false
  keyref_resolution: true
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if cfg.Core.Markdown.TextSync != "incremental" {
		t.Errorf("TextSync = %q, want %q", cfg.Core.Markdown.TextSync, "incremental")
	}
	if cfg.Core.Markdown.TitleFromHeading != false {
		t.Errorf("TitleFromHeading = %v, want false", cfg.Core.Markdown.TitleFromHeading)
	}
	if cfg.Completion.MaxCandidates != 100 {
		t.Errorf("MaxCandidates = %d, want 100", cfg.Completion.MaxCandidates)
	}
	if cfg.CodeActions.ToC.Enable == nil || *cfg.CodeActions.ToC.Enable != false {
		t.Errorf("ToC.Enable = %v, want false", cfg.CodeActions.ToC.Enable)
	}
	exts := cfg.Core.Mdita.MapExtensions
	if len(exts) != 2 || exts[0] != "mditamap" || exts[1] != "ditamap" {
		t.Errorf("MapExtensions = %v, want [mditamap ditamap]", exts)
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Core.Markdown.TextSync != "full" {
		t.Errorf("default TextSync = %q, want %q", cfg.Core.Markdown.TextSync, "full")
	}
	if !BoolVal(cfg.Core.Mdita.Enable) {
		t.Errorf("default Mdita.Enable = %v, want true", cfg.Core.Mdita.Enable)
	}
	if cfg.Completion.MaxCandidates != 50 {
		t.Errorf("default MaxCandidates = %d, want 50", cfg.Completion.MaxCandidates)
	}
	if len(cfg.CodeActions.ToC.IncludeLevels) != 6 {
		t.Errorf("default IncludeLevels len = %d, want 6", len(cfg.CodeActions.ToC.IncludeLevels))
	}
	if !BoolVal(cfg.Build.DitaOT.Enable) {
		t.Error("default Build.DitaOT.Enable should be true")
	}
	if cfg.Build.DitaOT.OutputDir != "out" {
		t.Errorf("default OutputDir = %q, want %q", cfg.Build.DitaOT.OutputDir, "out")
	}
}

func TestMerge(t *testing.T) {
	base := Default()
	overlay, _ := Parse([]byte(`
completion:
  max_candidates: 25
`))
	merged := Merge(base, overlay)
	if merged.Completion.MaxCandidates != 25 {
		t.Errorf("merged MaxCandidates = %d, want 25", merged.Completion.MaxCandidates)
	}
	if !BoolVal(merged.Core.Mdita.Enable) {
		t.Errorf("merged Mdita.Enable = %v, want true (from default)", merged.Core.Mdita.Enable)
	}
}

func TestMergeBoolOverrides(t *testing.T) {
	base := Default()
	overlay, _ := Parse([]byte(`
core:
  mdita:
    enable: false
code_actions:
  toc:
    enable: false
  create_missing_file:
    enable: false
diagnostics:
  mdita_compliance: false
  ditamap_validation: false
  keyref_resolution: false
`))
	merged := Merge(base, overlay)

	if BoolVal(merged.Core.Mdita.Enable) {
		t.Error("merged Mdita.Enable should be false after overlay")
	}
	if BoolVal(merged.CodeActions.ToC.Enable) {
		t.Error("merged ToC.Enable should be false after overlay")
	}
	if BoolVal(merged.CodeActions.CreateMissingFile.Enable) {
		t.Error("merged CreateMissingFile.Enable should be false after overlay")
	}
	if BoolVal(merged.Diagnostics.MditaCompliance) {
		t.Error("merged MditaCompliance should be false after overlay")
	}
	if BoolVal(merged.Diagnostics.DitamapValidation) {
		t.Error("merged DitamapValidation should be false after overlay")
	}
	if BoolVal(merged.Diagnostics.KeyrefResolution) {
		t.Error("merged KeyrefResolution should be false after overlay")
	}
}

func TestMergeUnsetBoolPreservesBase(t *testing.T) {
	base := Default()
	overlay, _ := Parse([]byte(`
completion:
  max_candidates: 10
`))
	merged := Merge(base, overlay)

	if !BoolVal(merged.Core.Mdita.Enable) {
		t.Error("unset overlay should preserve base Mdita.Enable=true")
	}
	if !BoolVal(merged.Diagnostics.MditaCompliance) {
		t.Error("unset overlay should preserve base MditaCompliance=true")
	}
	if !BoolVal(merged.CodeActions.ToC.Enable) {
		t.Error("unset overlay should preserve base ToC.Enable=true")
	}
}

func TestParseBuildConfig(t *testing.T) {
	input := `
build:
  dita_ot:
    enable: true
    dita_path: "/opt/dita-ot/bin/dita"
    output_dir: "build-output"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !BoolVal(cfg.Build.DitaOT.Enable) {
		t.Error("Build.DitaOT.Enable should be true")
	}
	if cfg.Build.DitaOT.DitaPath != "/opt/dita-ot/bin/dita" {
		t.Errorf("DitaPath = %q, want %q", cfg.Build.DitaOT.DitaPath, "/opt/dita-ot/bin/dita")
	}
	if cfg.Build.DitaOT.OutputDir != "build-output" {
		t.Errorf("OutputDir = %q, want %q", cfg.Build.DitaOT.OutputDir, "build-output")
	}
}

func TestMergeBuildConfig(t *testing.T) {
	base := Default()
	overlay, _ := Parse([]byte(`
build:
  dita_ot:
    enable: false
    dita_path: "/custom/dita"
    output_dir: "custom-out"
`))
	merged := Merge(base, overlay)
	if BoolVal(merged.Build.DitaOT.Enable) {
		t.Error("merged Build.DitaOT.Enable should be false")
	}
	if merged.Build.DitaOT.DitaPath != "/custom/dita" {
		t.Errorf("merged DitaPath = %q, want %q", merged.Build.DitaOT.DitaPath, "/custom/dita")
	}
	if merged.Build.DitaOT.OutputDir != "custom-out" {
		t.Errorf("merged OutputDir = %q, want %q", merged.Build.DitaOT.OutputDir, "custom-out")
	}
}

func TestParseEmpty(t *testing.T) {
	cfg, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("Parse empty error: %v", err)
	}
	if cfg.Core.Markdown.TextSync != "" {
		t.Errorf("empty parse should have zero values, got TextSync = %q", cfg.Core.Markdown.TextSync)
	}
}
