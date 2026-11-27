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
	A2A       A2AConfig       `yaml:"a2a"`
	MCP       MCPConfig       `yaml:"mcp"`
}

// MCPConfig 暴露 Model Context Protocol：主进程 Streamable HTTP + 独立 cmd/mcp stdio。
type MCPConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
	Path    string `yaml:"path"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// A2AConfig 启用 Google A2A（Agent-to-Agent）协议：入站建任务、出站挂远程子代理。
type A2AConfig struct {
	Enabled       bool                `yaml:"enabled"`
	Addr          string              `yaml:"addr"`
	PublicURL     string              `yaml:"public_url"`
	HandlerPath   string              `yaml:"handler_path"`
	AuthToken     string              `yaml:"auth_token"`
	Name          string              `yaml:"name"`
	Description   string              `yaml:"description"`
	Version       string              `yaml:"version"`
	ConfirmPolicy string              `yaml:"confirm_policy"`
	// AutoRegister：主 API 监听成功后自动拉起 A2A，并向 registries 登记 Agent Card。
	AutoRegister bool                `yaml:"auto_register"`
	Registries   []A2ARegistryConfig `yaml:"registries"`
	Remotes      []A2ARemoteConfig   `yaml:"remotes"`
}

// A2ARegistryConfig 启动成功后向外部注册中心提交 wellKnownURI（Agent Card）。
type A2ARegistryConfig struct {
	URL       string `yaml:"url"`
	AuthToken string `yaml:"auth_token"`
}

type A2ARemoteConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	BaseURL     string `yaml:"base_url"`
	HandlerPath string `yaml:"handler_path"`
	AuthToken   string `yaml:"auth_token"`
	Streaming   *bool  `yaml:"streaming"`
}

type ServerConfig struct {
	Addr        string        `yaml:"addr"`
	Mode        string        `yaml:"mode"`
	LogLevel    string        `yaml:"log_level"`
	LogFormat   string        `yaml:"log_format"`
	LogFile     string        `yaml:"log_file"`
	LogKeepDays int           `yaml:"log_keep_days"`
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
				"http://localhost:5273",
				"http://127.0.0.1:5273",
				"http://tauri.localhost",
				"https://tauri.localhost",
				"tauri://localhost",
			},
			LogLevel:    "info",
			LogFormat:   "console",
			LogKeepDays: 14,
			JWTSecret:   "eino-dev-jwt-secret-change-me",
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
		A2A: A2AConfig{
			Enabled:       true,
			AutoRegister:  true,
			Addr:          ":8181",
			PublicURL:     "http://127.0.0.1:8181",
			HandlerPath:   "/a2a",
			Name:          "eino-work-harness",
			Description:   "长任务工作 Harness：下达目标、拆解执行、交付可下载产物",
			Version:       "1.0.0",
			ConfirmPolicy: "never",
		},
		MCP: MCPConfig{
			Enabled: true,
			Addr:    ":8182",
			Path:    "/mcp",
			Name:    "eino-work-harness",
			Version: "1.0.0",
		},
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
	if strings.TrimSpace(c.Server.LogLevel) == "" {
		c.Server.LogLevel = "info"
	}
	if strings.TrimSpace(c.Server.LogFormat) == "" {
		if strings.EqualFold(c.Server.Mode, "release") {
			c.Server.LogFormat = "json"
		} else {
			c.Server.LogFormat = "console"
		}
	}
	if c.Server.LogKeepDays < 0 {
		c.Server.LogKeepDays = 14
	}
	if c.A2A.Addr == "" {
		c.A2A.Addr = ":8181"
	}
	if c.A2A.PublicURL == "" {
		c.A2A.PublicURL = "http://127.0.0.1:8181"
	}
	if c.A2A.HandlerPath == "" {
		c.A2A.HandlerPath = "/a2a"
	}
	if !strings.HasPrefix(c.A2A.HandlerPath, "/") {
		c.A2A.HandlerPath = "/" + c.A2A.HandlerPath
	}
	if c.A2A.Name == "" {
		c.A2A.Name = "eino-work-harness"
	}
	if c.A2A.Description == "" {
		c.A2A.Description = "长任务工作 Harness：下达目标、拆解执行、交付可下载产物"
	}
	if c.A2A.Version == "" {
		c.A2A.Version = "1.0.0"
	}
	if c.A2A.ConfirmPolicy == "" {
		c.A2A.ConfirmPolicy = "never"
	}
	for i := range c.A2A.Remotes {
		r := &c.A2A.Remotes[i]
		if r.HandlerPath == "" {
			r.HandlerPath = "/a2a"
		}
		if !strings.HasPrefix(r.HandlerPath, "/") {
			r.HandlerPath = "/" + r.HandlerPath
		}
	}
	if c.MCP.Addr == "" {
		c.MCP.Addr = ":8182"
	}
	if c.MCP.Path == "" {
		c.MCP.Path = "/mcp"
	}
	if !strings.HasPrefix(c.MCP.Path, "/") {
		c.MCP.Path = "/" + c.MCP.Path
	}
	if c.MCP.Name == "" {
		c.MCP.Name = "eino-work-harness"
	}
	if c.MCP.Version == "" {
		c.MCP.Version = "1.0.0"
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
	if v := os.Getenv("EINO_LOG_LEVEL"); v != "" {
		cfg.Server.LogLevel = v
	}
	if v := os.Getenv("EINO_LOG_FORMAT"); v != "" {
		cfg.Server.LogFormat = v
	}
	if v := os.Getenv("EINO_LOG_FILE"); v != "" {
		cfg.Server.LogFile = v
	}
	if v := os.Getenv("EINO_LOG_KEEP_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.LogKeepDays = n
		}
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
	if v := os.Getenv("EINO_A2A_ENABLED"); v != "" {
		cfg.A2A.Enabled = strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	if v := os.Getenv("EINO_A2A_ADDR"); v != "" {
		cfg.A2A.Addr = v
	}
	if v := os.Getenv("EINO_A2A_PUBLIC_URL"); v != "" {
		cfg.A2A.PublicURL = v
	}
	if v := os.Getenv("EINO_A2A_HANDLER_PATH"); v != "" {
		cfg.A2A.HandlerPath = v
	}
	if v := os.Getenv("EINO_A2A_AUTH_TOKEN"); v != "" {
		cfg.A2A.AuthToken = v
	}
	if v := os.Getenv("EINO_A2A_NAME"); v != "" {
		cfg.A2A.Name = v
	}
	if v := os.Getenv("EINO_A2A_CONFIRM_POLICY"); v != "" {
		cfg.A2A.ConfirmPolicy = v
	}
	if v := os.Getenv("EINO_A2A_AUTO_REGISTER"); v != "" {
		cfg.A2A.AutoRegister = strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	if v := os.Getenv("EINO_MCP_ENABLED"); v != "" {
		cfg.MCP.Enabled = strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	if v := os.Getenv("EINO_MCP_ADDR"); v != "" {
		cfg.MCP.Addr = v
	}
	if v := os.Getenv("EINO_MCP_PATH"); v != "" {
		cfg.MCP.Path = v
	}
	if v := os.Getenv("EINO_MCP_NAME"); v != "" {
		cfg.MCP.Name = v
	}
	if v := os.Getenv("EINO_MCP_VERSION"); v != "" {
		cfg.MCP.Version = v
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
