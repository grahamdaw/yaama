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

	Repo          string `toml:"repo"`
	DefaultBranch string `toml:"default_branch"`
	Worktree      bool   `toml:"worktree"`

	Setup    string `toml:"setup"`
	Teardown string `toml:"teardown"`

	LayoutFile    string `toml:"layout_file"`
	StartupWindow string `toml:"startup_window"`

	Windows []Window `toml:"windows"`
}

type Window struct {
	Name  string `toml:"name"`
	Focus bool   `toml:"focus"`
	Panes []Pane `toml:"panes"`
}

type Pane struct {
	Split string `toml:"split"`
	Size  string `toml:"size"`
	Cwd   string `toml:"cwd"`
	Run   string `toml:"run"`
	Agent bool   `toml:"agent"`
}

type RuntimeValues struct {
	WorkingDir string
	Branch     string
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
	if _, err := toml.DecodeFile(profilePath, &cfg); err != nil {
		log.Error("profile.load.parse_error",
			"profile", profileName,
			"path", profilePath,
			"err", logging.Truncate(err.Error(), 512))
		return Config{}, fmt.Errorf("load profile %q: %w", profileName, err)
	}
	cfg.Name = profileName
	if err := validateLoadedConfig(cfg); err != nil {
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
	workingDir := strings.TrimSpace(cfg.Repo)
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

	return RuntimeValues{
		WorkingDir: filepath.Clean(workingDir),
		Branch:     branch,
	}, nil
}

func validateLoadedConfig(cfg Config) error {
	if len(cfg.Windows) == 0 {
		return errors.New("profile must declare at least one [[windows]] entry")
	}
	agentSeen := false
	for idx, window := range cfg.Windows {
		if strings.TrimSpace(window.Name) == "" {
			return fmt.Errorf("profile window at index %d is missing name", idx)
		}
		for paneIdx, pane := range window.Panes {
			split := strings.TrimSpace(strings.ToLower(pane.Split))
			if split != "" && split != "horizontal" && split != "vertical" {
				return fmt.Errorf("profile window %q pane %d has invalid split %q", window.Name, paneIdx, pane.Split)
			}
			if pane.Agent {
				if agentSeen {
					return errors.New("profile may declare at most one pane with agent = true")
				}
				agentSeen = true
			}
		}
	}
	return nil
}

func (c *Config) resolveDefaultsAndPaths(configRoot string) {
	c.Repo = strings.TrimSpace(c.Repo)
	c.DefaultBranch = strings.TrimSpace(c.DefaultBranch)
	if c.DefaultBranch == "" {
		c.DefaultBranch = defaultBranchName
	}

	c.StartupWindow = strings.TrimSpace(c.StartupWindow)
	if c.LayoutFile != "" {
		c.LayoutFile = resolveConfigPath(configRoot, c.LayoutFile)
	}

	c.Setup = resolveScriptEntry(configRoot, c.Setup)
	c.Teardown = resolveScriptEntry(configRoot, c.Teardown)
}

func resolveScriptEntry(configRoot, entry string) string {
	trimmed := strings.TrimSpace(entry)
	if trimmed == "" {
		return ""
	}
	if looksLikePath(trimmed) {
		return resolveConfigPath(configRoot, trimmed)
	}
	return trimmed
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
		Name:          name,
		DefaultBranch: defaultBranchName,
		Windows: []Window{
			{
				Name:  "agent",
				Focus: true,
				Panes: []Pane{{Agent: true}},
			},
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
