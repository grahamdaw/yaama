package profile

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/grahamdaw/yaama/internal/logging"
)

const (
	configDirName      = ".config/yaama"
	profilesSubdirName = "profiles"
	defaultBranchName  = "main"
)

type Config struct {
	Name string

	Agent AgentConfig `toml:"agent"`
	Repo  RepoConfig  `toml:"repo"`
	Tmux  TmuxConfig  `toml:"tmux"`

	Scripts ScriptsConfig `toml:"scripts"`
}

type AgentConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

type RepoConfig struct {
	Path          string `toml:"path"`
	DefaultBranch string `toml:"default_branch"`
}

type ScriptsConfig struct {
	BeforeStart []string `toml:"before_start"`
	AfterStart  []string `toml:"after_start"`
	Cleanup     []string `toml:"cleanup"`
}

type TmuxConfig struct {
	SessionPrefix string       `toml:"session_prefix"`
	LayoutFile    string       `toml:"layout_file"`
	StartupWindow string       `toml:"startup_window"`
	Windows       []TmuxWindow `toml:"windows"`
}

type TmuxWindow struct {
	Name  string     `toml:"name"`
	Focus bool       `toml:"focus"`
	Panes []TmuxPane `toml:"panes"`
}

type TmuxPane struct {
	Split string `toml:"split"`
	Size  string `toml:"size"`
	Cwd   string `toml:"cwd"`
	Run   string `toml:"run"`
}

type RuntimeValues struct {
	WorkingDir   string
	Branch       string
	AgentCommand []string
}

func ListAvailable() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{"default"}
	}
	profilesDir := profilesDirForHome(home)
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return []string{"default"}
	}

	profiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".toml")
		if strings.TrimSpace(base) == "" {
			continue
		}
		profiles = append(profiles, base)
	}
	if len(profiles) == 0 {
		return []string{"default"}
	}
	sort.Strings(profiles)
	return profiles
}

func ValidateReference(name string) error {
	profileName := strings.TrimSpace(name)
	if profileName == "" {
		return nil
	}
	if profileName == "default" {
		return nil
	}
	if !isSafeProfileName(profileName) {
		return errors.New("invalid profile name")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.New("cannot resolve home directory")
	}
	path := filepath.Join(profilesDirForHome(home), profileName+".toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.New("profile not found in ~/.config/yaama/profiles")
		}
		return errors.New("unable to verify profile file")
	}
	return nil
}

func Load(name string) (Config, error) {
	return LoadWithLogger(name, nil)
}

// LoadWithLogger behaves like Load but emits diagnostic events to the
// supplied logger. A nil logger discards events; the API is purely
// additive so existing callers do not need to change.
func LoadWithLogger(name string, logger *slog.Logger) (Config, error) {
	log := logger
	if log == nil {
		log = logging.Discard()
	}

	profileName := strings.TrimSpace(name)
	if profileName == "" {
		return Config{}, errors.New("profile name is required")
	}
	if !isSafeProfileName(profileName) {
		return Config{}, errors.New("invalid profile name")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, errors.New("cannot resolve home directory")
	}
	configRoot := configRootForHome(home)
	profilePath := filepath.Join(configRoot, profilesSubdirName, profileName+".toml")
	if _, err := os.Stat(profilePath); err != nil {
		if os.IsNotExist(err) && profileName == "default" {
			log.Warn("profile.load.missing_default", "profile", profileName, "path", profilePath)
			return defaultConfig(profileName), nil
		}
	}

	var cfg Config
	meta, err := toml.DecodeFile(profilePath, &cfg)
	if err != nil {
		log.Error("profile.load.parse_error",
			"profile", profileName,
			"path", profilePath,
			"err", logging.Truncate(err.Error(), 512))
		return Config{}, fmt.Errorf("load profile %q: %w", profileName, err)
	}
	cfg.Name = profileName
	if err := validateLoadedConfig(cfg, meta); err != nil {
		log.Error("profile.load.validation_failed",
			"profile", profileName,
			"err", logging.Truncate(err.Error(), 512))
		return Config{}, err
	}

	cfg.resolveDefaultsAndPaths(configRoot)
	log.Debug("profile.load.ok", "profile", profileName, "path", profilePath)
	return cfg, nil
}

func ResolveRuntimeValues(cfg Config, fallbackDir, _, branchInput string) (RuntimeValues, error) {
	workingDir := strings.TrimSpace(cfg.Repo.Path)
	if workingDir == "" {
		workingDir = strings.TrimSpace(fallbackDir)
	}
	if workingDir == "" {
		return RuntimeValues{}, errors.New("unable to resolve working directory")
	}
	if !filepath.IsAbs(workingDir) {
		workingDir = filepath.Clean(filepath.Join(fallbackDir, workingDir))
	}

	branch := strings.TrimSpace(branchInput)
	if branch == "" {
		return RuntimeValues{}, errors.New("branch is required")
	}

	agentCommand := make([]string, 0, len(cfg.Agent.Args)+1)
	agentCommand = append(agentCommand, strings.TrimSpace(cfg.Agent.Command))
	agentCommand = append(agentCommand, cfg.Agent.Args...)

	return RuntimeValues{
		WorkingDir:   filepath.Clean(workingDir),
		Branch:       branch,
		AgentCommand: agentCommand,
	}, nil
}

func validateLoadedConfig(cfg Config, meta toml.MetaData) error {
	for _, section := range []string{"agent", "repo", "tmux"} {
		if !meta.IsDefined(section) {
			return fmt.Errorf("profile is missing [%s] section", section)
		}
	}
	if meta.IsDefined("agent", "prompt_arg") {
		return errors.New(`profile [agent].prompt_arg is no longer supported; move prompt flags into [agent].args`)
	}
	if meta.IsDefined("agent", "ticket_arg") {
		return errors.New(`profile [agent].ticket_arg is no longer supported; move ticket flags into [agent].args`)
	}
	if strings.TrimSpace(cfg.Agent.Command) == "" {
		return errors.New("profile [agent].command is required")
	}
	for idx, window := range cfg.Tmux.Windows {
		if strings.TrimSpace(window.Name) == "" {
			return fmt.Errorf("profile tmux window at index %d is missing name", idx)
		}
		for paneIdx, pane := range window.Panes {
			split := strings.TrimSpace(strings.ToLower(pane.Split))
			if split != "" && split != "horizontal" && split != "vertical" {
				return fmt.Errorf("profile tmux window %q pane %d has invalid split %q", window.Name, paneIdx, pane.Split)
			}
		}
	}
	return nil
}

func (c *Config) resolveDefaultsAndPaths(configRoot string) {
	c.Agent.Command = strings.TrimSpace(c.Agent.Command)

	c.Repo.Path = strings.TrimSpace(c.Repo.Path)
	c.Repo.DefaultBranch = strings.TrimSpace(c.Repo.DefaultBranch)
	if c.Repo.DefaultBranch == "" {
		c.Repo.DefaultBranch = defaultBranchName
	}

	c.Tmux.StartupWindow = strings.TrimSpace(c.Tmux.StartupWindow)
	if c.Tmux.LayoutFile != "" {
		c.Tmux.LayoutFile = resolveConfigPath(configRoot, c.Tmux.LayoutFile)
	}

	c.Scripts.BeforeStart = resolveScriptEntries(configRoot, c.Scripts.BeforeStart)
	c.Scripts.AfterStart = resolveScriptEntries(configRoot, c.Scripts.AfterStart)
	c.Scripts.Cleanup = resolveScriptEntries(configRoot, c.Scripts.Cleanup)
}

func resolveScriptEntries(configRoot string, entries []string) []string {
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, item := range entries {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if looksLikePath(trimmed) {
			out = append(out, resolveConfigPath(configRoot, trimmed))
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func resolveConfigPath(configRoot, pathValue string) string {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	return filepath.Clean(filepath.Join(configRoot, trimmed))
}

func looksLikePath(value string) bool {
	return strings.HasPrefix(value, ".") ||
		strings.HasPrefix(value, "/") ||
		strings.Contains(value, "/") ||
		strings.HasSuffix(value, ".sh")
}

func isSafeProfileName(name string) bool {
	return !strings.Contains(name, "/") && !strings.Contains(name, `\`) && !strings.Contains(name, "..")
}

func defaultConfig(name string) Config {
	return Config{
		Name: name,
		Agent: AgentConfig{
			Command: "codex",
		},
		Repo: RepoConfig{
			DefaultBranch: defaultBranchName,
		},
		Tmux: TmuxConfig{
			StartupWindow: "agent",
		},
	}
}

func configRootForHome(home string) string {
	configRoot := filepath.Join(home, configDirName)
	_ = os.MkdirAll(filepath.Join(configRoot, profilesSubdirName), 0o755)
	return configRoot
}

func profilesDirForHome(home string) string {
	return filepath.Join(configRootForHome(home), profilesSubdirName)
}
