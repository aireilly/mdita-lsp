package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Core        CoreConfig        `yaml:"core"`
	Completion  CompletionConfig  `yaml:"completion"`
	CodeActions CodeActionsConfig `yaml:"code_actions"`
	Diagnostics DiagnosticsConfig `yaml:"diagnostics"`
	Build       BuildConfig       `yaml:"build"`
}

type CoreConfig struct {
	Markdown MarkdownConfig `yaml:"markdown"`
	Mdita    MditaConfig    `yaml:"mdita"`
}

type MarkdownConfig struct {
	FileExtensions []string `yaml:"file_extensions"`
	// TextSync selects the document sync mode the server advertises:
	// "incremental" (the default) or "full".
	TextSync string `yaml:"text_sync"`
}

type MditaConfig struct {
	Enable        *bool    `yaml:"enable"`
	MapExtensions []string `yaml:"map_extensions"`
	// ApplyToMarkdown treats .md and .markdown topics that declare no MDITA
	// $schema as MDITA extended, for a workspace whose map gives them
	// format="mdita". DITA-OT takes the format from the topicref rather than
	// the extension, so the server cannot work this out on its own.
	//
	// There is no companion `profile` setting: the profile comes from the
	// $schema, and MDitaReader's own default is the extended one. Declare
	// `$schema: urn:oasis:names:tc:mdita:core:xsd:topic.xsd` in a topic that
	// needs core.
	ApplyToMarkdown *bool `yaml:"apply_to_markdown"`
	// ImplicitTaskSections overrides the heading titles that map to task
	// section elements, matching the plug-in's
	// http://lwdita.org/sax/properties/implicit-task-sections/* properties.
	// Keys are prereq, context, steps, result, and postreq.
	ImplicitTaskSections map[string][]string `yaml:"implicit_task_sections"`
	FormatTablesOnSave   *bool               `yaml:"formatTablesOnSave"`
}

type CompletionConfig struct {
	MaxCandidates int `yaml:"max_candidates"`
}

type CodeActionsConfig struct {
	CreateMissingFile CreateMissingFileConfig `yaml:"create_missing_file"`
}

type CreateMissingFileConfig struct {
	Enable *bool `yaml:"enable"`
}

type DiagnosticsConfig struct {
	MditaCompliance   *bool `yaml:"mdita_compliance"`
	DitamapValidation *bool `yaml:"ditamap_validation"`
	KeyrefResolution  *bool `yaml:"keyref_resolution"`
	LinkValidation    *bool `yaml:"link_validation"`
	NbspDetection     *bool `yaml:"nbsp_detection"`
}

type BuildConfig struct {
	DitaOT DitaOTConfig `yaml:"dita_ot"`
}

type DitaOTConfig struct {
	Enable    *bool  `yaml:"enable"`
	DitaPath  string `yaml:"dita_path,omitempty"`
	OutputDir string `yaml:"output_dir,omitempty"`
}

func boolPtr(v bool) *bool { return &v }

func BoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func Default() *Config {
	return &Config{
		Core: CoreConfig{
			Markdown: MarkdownConfig{
				FileExtensions: []string{"md", "markdown", "mdita", "mditamap"},
				TextSync:       "incremental",
			},
			Mdita: MditaConfig{
				Enable:             boolPtr(true),
				ApplyToMarkdown:    boolPtr(false),
				MapExtensions:      []string{"mditamap"},
				FormatTablesOnSave: boolPtr(true),
			},
		},
		Completion: CompletionConfig{
			MaxCandidates: 50,
		},
		CodeActions: CodeActionsConfig{
			CreateMissingFile: CreateMissingFileConfig{
				Enable: boolPtr(true),
			},
		},
		Diagnostics: DiagnosticsConfig{
			MditaCompliance:   boolPtr(true),
			DitamapValidation: boolPtr(true),
			KeyrefResolution:  boolPtr(true),
			LinkValidation:    boolPtr(true),
			NbspDetection:     boolPtr(true),
		},
		Build: BuildConfig{
			DitaOT: DitaOTConfig{
				Enable:    boolPtr(true),
				OutputDir: "out",
			},
		},
	}
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	if len(data) == 0 {
		return &cfg, nil
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	return Parse(data)
}

func mergeBool(base, overlay *bool) *bool {
	if overlay != nil {
		return overlay
	}
	return base
}

func Merge(base, overlay *Config) *Config {
	merged := *base

	if overlay.Core.Markdown.TextSync != "" {
		merged.Core.Markdown.TextSync = overlay.Core.Markdown.TextSync
	}
	if overlay.Core.Markdown.FileExtensions != nil {
		merged.Core.Markdown.FileExtensions = overlay.Core.Markdown.FileExtensions
	}
	merged.Core.Mdita.Enable = mergeBool(base.Core.Mdita.Enable, overlay.Core.Mdita.Enable)
	if overlay.Core.Mdita.MapExtensions != nil {
		merged.Core.Mdita.MapExtensions = overlay.Core.Mdita.MapExtensions
	}
	merged.Core.Mdita.ApplyToMarkdown = mergeBool(base.Core.Mdita.ApplyToMarkdown, overlay.Core.Mdita.ApplyToMarkdown)
	if overlay.Core.Mdita.FormatTablesOnSave != nil {
		merged.Core.Mdita.FormatTablesOnSave = overlay.Core.Mdita.FormatTablesOnSave
	}
	if len(overlay.Core.Mdita.ImplicitTaskSections) > 0 {
		merged.Core.Mdita.ImplicitTaskSections = overlay.Core.Mdita.ImplicitTaskSections
	}

	if overlay.Completion.MaxCandidates != 0 {
		merged.Completion.MaxCandidates = overlay.Completion.MaxCandidates
	}

	merged.CodeActions.CreateMissingFile.Enable = mergeBool(base.CodeActions.CreateMissingFile.Enable, overlay.CodeActions.CreateMissingFile.Enable)

	merged.Diagnostics.MditaCompliance = mergeBool(base.Diagnostics.MditaCompliance, overlay.Diagnostics.MditaCompliance)
	merged.Diagnostics.DitamapValidation = mergeBool(base.Diagnostics.DitamapValidation, overlay.Diagnostics.DitamapValidation)
	merged.Diagnostics.KeyrefResolution = mergeBool(base.Diagnostics.KeyrefResolution, overlay.Diagnostics.KeyrefResolution)
	merged.Diagnostics.LinkValidation = mergeBool(base.Diagnostics.LinkValidation, overlay.Diagnostics.LinkValidation)
	merged.Diagnostics.NbspDetection = mergeBool(base.Diagnostics.NbspDetection, overlay.Diagnostics.NbspDetection)

	merged.Build.DitaOT.Enable = mergeBool(base.Build.DitaOT.Enable, overlay.Build.DitaOT.Enable)
	if overlay.Build.DitaOT.DitaPath != "" {
		merged.Build.DitaOT.DitaPath = overlay.Build.DitaOT.DitaPath
	}
	if overlay.Build.DitaOT.OutputDir != "" {
		merged.Build.DitaOT.OutputDir = overlay.Build.DitaOT.OutputDir
	}

	return &merged
}

func LoadMerged(folderRoot string) *Config {
	cfg := Default()

	home, err := os.UserHomeDir()
	if err == nil {
		userCfg, err := Load(filepath.Join(home, ".config", "mdita-lsp", "config.yaml"))
		if err == nil {
			cfg = Merge(cfg, userCfg)
		}
	}

	folderCfg, err := Load(filepath.Join(folderRoot, ".mdita-lsp.yaml"))
	if err == nil {
		cfg = Merge(cfg, folderCfg)
	}

	return cfg
}
