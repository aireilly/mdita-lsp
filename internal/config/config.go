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
}

type CoreConfig struct {
	Markdown MarkdownConfig `yaml:"markdown"`
	Mdita    MditaConfig    `yaml:"mdita"`
}

type MarkdownConfig struct {
	FileExtensions   []string `yaml:"file_extensions"`
	TextSync         string   `yaml:"text_sync"`
	TitleFromHeading bool     `yaml:"title_from_heading"`
}

type MditaConfig struct {
	Enable        bool     `yaml:"enable"`
	MapExtensions []string `yaml:"map_extensions"`
}

type CompletionConfig struct {
	WikiStyle     string `yaml:"wiki_style"`
	MaxCandidates int    `yaml:"max_candidates"`
}

type CodeActionsConfig struct {
	ToC               ToCConfig               `yaml:"toc"`
	CreateMissingFile CreateMissingFileConfig `yaml:"create_missing_file"`
}

type ToCConfig struct {
	Enable        bool  `yaml:"enable"`
	IncludeLevels []int `yaml:"include_levels"`
}

type CreateMissingFileConfig struct {
	Enable bool `yaml:"enable"`
}

type DiagnosticsConfig struct {
	MditaCompliance   bool `yaml:"mdita_compliance"`
	DitamapValidation bool `yaml:"ditamap_validation"`
	KeyrefResolution  bool `yaml:"keyref_resolution"`
}

func Default() *Config {
	return &Config{
		Core: CoreConfig{
			Markdown: MarkdownConfig{
				FileExtensions:   []string{"md", "markdown", "mditamap"},
				TextSync:         "full",
				TitleFromHeading: true,
			},
			Mdita: MditaConfig{
				Enable:        true,
				MapExtensions: []string{"mditamap"},
			},
		},
		Completion: CompletionConfig{
			WikiStyle:     "title-slug",
			MaxCandidates: 50,
		},
		CodeActions: CodeActionsConfig{
			ToC: ToCConfig{
				Enable:        true,
				IncludeLevels: []int{1, 2, 3, 4, 5, 6},
			},
			CreateMissingFile: CreateMissingFileConfig{
				Enable: true,
			},
		},
		Diagnostics: DiagnosticsConfig{
			MditaCompliance:   true,
			DitamapValidation: true,
			KeyrefResolution:  true,
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

func Merge(base, overlay *Config) *Config {
	merged := *base

	if overlay.Core.Markdown.TextSync != "" {
		merged.Core.Markdown.TextSync = overlay.Core.Markdown.TextSync
	}
	if overlay.Core.Markdown.FileExtensions != nil {
		merged.Core.Markdown.FileExtensions = overlay.Core.Markdown.FileExtensions
	}
	if overlay.Core.Markdown.TitleFromHeading != base.Core.Markdown.TitleFromHeading {
		merged.Core.Markdown.TitleFromHeading = overlay.Core.Markdown.TitleFromHeading
	}
	if overlay.Core.Mdita.MapExtensions != nil {
		merged.Core.Mdita.MapExtensions = overlay.Core.Mdita.MapExtensions
	}

	if overlay.Completion.WikiStyle != "" {
		merged.Completion.WikiStyle = overlay.Completion.WikiStyle
	}
	if overlay.Completion.MaxCandidates != 0 {
		merged.Completion.MaxCandidates = overlay.Completion.MaxCandidates
	}

	if overlay.CodeActions.ToC.IncludeLevels != nil {
		merged.CodeActions.ToC.IncludeLevels = overlay.CodeActions.ToC.IncludeLevels
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
