package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Workspace WorkspaceConfig `yaml:"workspace"`
	Skills    SkillsConfig    `yaml:"skills"`
	LLM       LLMConfig       `yaml:"llm"`
	Agent     AgentConfig     `yaml:"agent"`
	Search    SearchConfig    `yaml:"search"`
}

type ServerConfig struct {
	Addr        string        `yaml:"addr"`
	Mode        string        `yaml:"mode"`
	CORSOrigins []string      `yaml:"cors_origins"`
	AuthToken   string        `yaml:"auth_token"` // 可选：静态 Bearer，兼容旧 CLI/脚本
	JWTSecret   string        `yaml:"jwt_secret"`
	JWTExpire   time.Duration `yaml:"jwt_expire"`
}

type DatabaseConfig struct {
	DSN          string `yaml:"dsn"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

type WorkspaceConfig struct {
	Root string `yaml:"root"`
}

type SkillsConfig struct {
	Dir string `yaml:"dir"`
}

type LLMConfig struct {
	Provider       string        `yaml:"provider"`
	APIKey         string        `yaml:"api_key"`
	BaseURL        string        `yaml:"base_url"`
	Model          string        `yaml:"model"`
	FallbackModels []string      `yaml:"fallback_models"`
	Timeout        time.Duration `yaml:"timeout"`
}

type AgentConfig struct {
	MaxIteration int           `yaml:"max_iteration"`
	RunTimeout   time.Duration `yaml:"run_timeout"`
	Language     string        `yaml:"language"`
}

type SearchConfig struct {
	Enabled    bool `yaml:"enabled"`
	MaxResults int  `yaml:"max_results"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Addr: ":8080",
			Mode: "debug",
			CORSOrigins: []string{
				"http://localhost:5173",
				"http://127.0.0.1:5173",
			},
			JWTSecret: "eino-dev-jwt-secret-change-me",
			JWTExpire: 168 * time.Hour,
		},
		Database: DatabaseConfig{
			DSN:          "postgres://eino:eino@127.0.0.1:5432/eino_work?sslmode=disable",
			MaxOpenConns: 20,
			MaxIdleConns: 5,
		},
		Workspace: WorkspaceConfig{Root: "./workspace"},
		Skills:    SkillsConfig{Dir: "./skills"},
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4o",
			Timeout:  10 * time.Minute,
		},
		Agent: AgentConfig{
			MaxIteration: 40,
			RunTimeout:   30 * time.Minute,
			Language:     "zh",
		},
		Search: SearchConfig{Enabled: true, MaxResults: 8},
	}
}

func Load(path string) (*Config, error) {
	loadDotEnv(".env")
	cfg := Default()
	if path == "" {
		path = firstNonEmpty(os.Getenv("EINO_CONFIG"), "config.yaml")
	}
	if raw, err := os.ReadFile(path); err == nil {
		expanded := os.ExpandEnv(string(raw))
		if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	applyEnv(cfg)
	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) normalize() error {
	c.LLM.Provider = strings.ToLower(strings.TrimSpace(c.LLM.Provider))
	if c.LLM.Provider == "" {
		c.LLM.Provider = "openai"
	}
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if c.Workspace.Root == "" {
		c.Workspace.Root = "./workspace"
	}
	if c.Skills.Dir == "" {
		c.Skills.Dir = "./skills"
	}
	if c.Agent.MaxIteration <= 0 {
		c.Agent.MaxIteration = 40
	}
	if c.Agent.RunTimeout <= 0 {
		c.Agent.RunTimeout = 30 * time.Minute
	}
	if c.LLM.Timeout <= 0 {
		c.LLM.Timeout = 10 * time.Minute
	}
	if c.Search.MaxResults <= 0 {
		c.Search.MaxResults = 8
	}
	if c.Database.MaxOpenConns <= 0 {
		c.Database.MaxOpenConns = 20
	}
	if c.Database.MaxIdleConns <= 0 {
		c.Database.MaxIdleConns = 5
	}
	if strings.TrimSpace(c.Server.JWTSecret) == "" {
		c.Server.JWTSecret = "eino-dev-jwt-secret-change-me"
	}
	if c.Server.JWTExpire <= 0 {
		c.Server.JWTExpire = 168 * time.Hour
	}
	return nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("EINO_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("EINO_SERVER_MODE"); v != "" {
		cfg.Server.Mode = v
	}
	if v := os.Getenv("EINO_AUTH_TOKEN"); v != "" {
		cfg.Server.AuthToken = v
	}
	if v := os.Getenv("EINO_JWT_SECRET"); v != "" {
		cfg.Server.JWTSecret = v
	}
	if v := os.Getenv("EINO_JWT_EXPIRE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.JWTExpire = d
		}
	}
	if v := os.Getenv("EINO_DATABASE_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("EINO_WORKSPACE_ROOT"); v != "" {
		cfg.Workspace.Root = v
	}
	if v := os.Getenv("EINO_SKILLS_DIR"); v != "" {
		cfg.Skills.Dir = v
	}
	if v := os.Getenv("EINO_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("EINO_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("EINO_LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv("EINO_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("EINO_LLM_FALLBACK_MODELS"); v != "" {
		cfg.LLM.FallbackModels = compactCSV(v)
	}
	if v := os.Getenv("EINO_LLM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.LLM.Timeout = d
		}
	}
	if v := os.Getenv("EINO_AGENT_MAX_ITERATION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Agent.MaxIteration = n
		}
	}
	if v := os.Getenv("EINO_AGENT_RUN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Agent.RunTimeout = d
		}
	}
	if v := os.Getenv("EINO_AGENT_LANGUAGE"); v != "" {
		cfg.Agent.Language = v
	}
}

func loadDotEnv(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func compactCSV(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
