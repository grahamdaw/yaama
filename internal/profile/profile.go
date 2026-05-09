package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	configDirName      = ".config/yaam"
	profilesSubdirName = "profiles"
	defaultBranchName  = "main"
	defaultPromptArg   = "--prompt"
	defaultTicketArg   = "--ticket"
)

type Config struct {
	Name string

	Agent AgentConfig `toml:"agent"`
	Repo  RepoConfig  `toml:"repo"`
	Tmux  TmuxConfig  `toml:"tmux"`

	Scripts ScriptsConfig `toml:"scripts"`
}

type AgentConfig struct {
	Command   string   `toml:"command"`
	Args      []string `toml:"args"`
	PromptArg string   `toml:"prompt_arg"`
	TicketArg string   `toml:"ticket_arg"`
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
	profilesDir := filepath.Join(home, configDirName, profilesSubdirName)
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
	path := filepath.Join(home, configDirName, profilesSubdirName, profileName+".toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.New("profile not found in ~/.config/yaam/profiles")
		}
		return errors.New("unable to verify profile file")
	}
	return nil
}

func Load(name string) (Config, error) {
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
	configRoot := filepath.Join(home, configDirName)
	profilePath := filepath.Join(configRoot, profilesSubdirName, profileName+".toml")
	if _, err := os.Stat(profilePath); err != nil {
		if os.IsNotExist(err) && profileName == "default" {
			return defaultConfig(profileName), nil
		}
	}

	var cfg Config
	meta, err := toml.DecodeFile(profilePath, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("load profile %q: %w", profileName, err)
	}
	cfg.Name = profileName
	if err := validateLoadedConfig(cfg, meta); err != nil {
		return Config{}, err
	}

	cfg.resolveDefaultsAndPaths(configRoot)
	return cfg, nil
}

func ResolveRuntimeValues(cfg Config, fallbackDir, taskID string) (RuntimeValues, error) {
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

	branch := strings.TrimSpace(cfg.Repo.DefaultBranch)
	if branch == "" {
		branch = defaultBranchName
	}

	agentCommand := make([]string, 0, len(cfg.Agent.Args)+3)
	agentCommand = append(agentCommand, strings.TrimSpace(cfg.Agent.Command))
	agentCommand = append(agentCommand, cfg.Agent.Args...)
	task := strings.TrimSpace(taskID)
	if task != "" {
		agentCommand = append(agentCommand, cfg.Agent.TicketArg, task)
	}

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
	c.Agent.PromptArg = strings.TrimSpace(c.Agent.PromptArg)
	if c.Agent.PromptArg == "" {
		c.Agent.PromptArg = defaultPromptArg
	}
	c.Agent.TicketArg = strings.TrimSpace(c.Agent.TicketArg)
	if c.Agent.TicketArg == "" {
		c.Agent.TicketArg = defaultTicketArg
	}

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
			Command:   "codex",
			PromptArg: defaultPromptArg,
			TicketArg: defaultTicketArg,
		},
		Repo: RepoConfig{
			DefaultBranch: defaultBranchName,
		},
		Tmux: TmuxConfig{
			StartupWindow: "agent",
		},
	}
}
