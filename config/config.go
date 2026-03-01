package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the root configuration for nanobot
type Config struct {
	Agents    AgentsConfig    `mapstructure:"agents"`
	Channels  ChannelsConfig  `mapstructure:"channels"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Gateway   GatewayConfig   `mapstructure:"gateway"`
	Tools     ToolsConfig     `mapstructure:"tools"`
}

// AgentsConfig contains agent-specific configurations
type AgentsConfig struct {
	Defaults AgentDefaults `mapstructure:"defaults"`
}

// AgentDefaults contains default agent settings
type AgentDefaults struct {
	Workspace         string  `mapstructure:"workspace"`
	Model             string  `mapstructure:"model"`
	MaxTokens         int     `mapstructure:"max_tokens"`
	Temperature       float64 `mapstructure:"temperature"`
	MaxToolIterations int     `mapstructure:"max_tool_iterations"`
	MemoryWindow      int     `mapstructure:"memory_window"`
}

// ChannelsConfig contains configurations for various chat channels
type ChannelsConfig struct {
	SendProgress  bool           `mapstructure:"send_progress"`
	SendToolHints bool           `mapstructure:"send_tool_hints"`
	WhatsApp      WhatsAppConfig `mapstructure:"whatsapp"`
	Telegram      TelegramConfig `mapstructure:"telegram"`
	Discord       DiscordConfig  `mapstructure:"discord"`
	Feishu        FeishuConfig   `mapstructure:"feishu"`
	Mochat        MochatConfig   `mapstructure:"mochat"`
	DingTalk      DingTalkConfig `mapstructure:"dingtalk"`
	Email         EmailConfig    `mapstructure:"email"`
	QQ            QQConfig       `mapstructure:"qq"`
	Slack         SlackConfig    `mapstructure:"slack"`
	Wecom         WecomConfig    `mapstructure:"wecom"`
}

// WhatsAppConfig contains WhatsApp channel configuration
type WhatsAppConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// TelegramConfig contains Telegram channel configuration
type TelegramConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Token     string   `mapstructure:"token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// DiscordConfig contains Discord channel configuration
type DiscordConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Token     string   `mapstructure:"token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// FeishuConfig contains Feishu channel configuration
type FeishuConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	AppID        string   `mapstructure:"app_id"`
	AppSecret    string   `mapstructure:"app_secret"`
	AllowFrom    []string `mapstructure:"allow_from"`
	EncryptKey   string   `mapstructure:"encrypt_key"`
	Verification string   `mapstructure:"verification_token"`
}

// MochatConfig contains Mochat channel configuration
type MochatConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	BaseURL   string   `mapstructure:"base_url"`
	ClawToken string   `mapstructure:"claw_token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// DingTalkConfig contains DingTalk channel configuration
type DingTalkConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	ClientID  string   `mapstructure:"client_id"`
	Secret    string   `mapstructure:"client_secret"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// EmailConfig contains Email channel configuration
type EmailConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	Consent      bool     `mapstructure:"consent_granted"`
	IMAPHost     string   `mapstructure:"imap_host"`
	IMAPPort     int      `mapstructure:"imap_port"`
	IMAPUsername string   `mapstructure:"imap_username"`
	IMAPPassword string   `mapstructure:"imap_password"`
	SMTPHost     string   `mapstructure:"smtp_host"`
	SMTPPort     int      `mapstructure:"smtp_port"`
	SMTPUsername string   `mapstructure:"smtp_username"`
	SMTPPassword string   `mapstructure:"smtp_password"`
	FromAddress  string   `mapstructure:"from_address"`
	AllowFrom    []string `mapstructure:"allow_from"`
}

// SlackConfig contains Slack channel configuration
type SlackConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	BotToken  string   `mapstructure:"bot_token"`
	AppToken  string   `mapstructure:"app_token"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// QQConfig contains QQ channel configuration
type QQConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	AppID     string   `mapstructure:"app_id"`
	Secret    string   `mapstructure:"secret"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// WecomConfig contains WeCom (企业微信) channel configuration
type WecomConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	CorpID         string   `mapstructure:"corp_id"`
	AgentID        string   `mapstructure:"agent_id"`
	Secret         string   `mapstructure:"secret"`
	Token          string   `mapstructure:"token"`
	EncodingAESKey string   `mapstructure:"encoding_aes_key"`
	AllowFrom      []string `mapstructure:"allow_from"`
}

// ProvidersConfig contains configurations for LLM providers
type ProvidersConfig struct {
	Custom        ProviderConfig `mapstructure:"custom"`
	Anthropic     ProviderConfig `mapstructure:"anthropic"`
	OpenAI        ProviderConfig `mapstructure:"openai"`
	OpenRouter    ProviderConfig `mapstructure:"openrouter"`
	DeepSeek      ProviderConfig `mapstructure:"deepseek"`
	Groq          ProviderConfig `mapstructure:"groq"`
	ZhiPu         ProviderConfig `mapstructure:"zhipu"`
	DashScope     ProviderConfig `mapstructure:"dashscope"`
	VLLM          ProviderConfig `mapstructure:"vllm"`
	Gemini        ProviderConfig `mapstructure:"gemini"`
	Moonshot      ProviderConfig `mapstructure:"moonshot"`
	Minimax       ProviderConfig `mapstructure:"minimax"`
	AiHubMix      ProviderConfig `mapstructure:"aihubmix"`
	SiliconFlow   ProviderConfig `mapstructure:"siliconflow"`
	VolcEngine    ProviderConfig `mapstructure:"volcengine"`
	OpenAICodex   ProviderConfig `mapstructure:"openai_codex"`
	GithubCopilot ProviderConfig `mapstructure:"github_copilot"`
}

// ProviderConfig contains individual provider configuration
type ProviderConfig struct {
	APIKey       string            `mapstructure:"api_key"`
	APIBase      string            `mapstructure:"api_base"`
	ExtraHeaders map[string]string `mapstructure:"extra_headers"`
}

// GatewayConfig contains gateway/server configuration
type GatewayConfig struct {
	Host      string          `mapstructure:"host"`
	Port      int             `mapstructure:"port"`
	Heartbeat HeartbeatConfig `mapstructure:"heartbeat"`
}

// HeartbeatConfig contains heartbeat service configuration
type HeartbeatConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	IntervalS int  `mapstructure:"interval_s"`
}

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	Web                 WebToolsConfig `mapstructure:"web"`
	Exec                ExecToolConfig `mapstructure:"exec"`
	RestrictToWorkspace bool           `mapstructure:"restrict_to_workspace"`
	MCPServers          map[string]any `mapstructure:"mcp_servers"`
}

// WebToolsConfig contains web tools configuration
type WebToolsConfig struct {
	Search WebSearchConfig `mapstructure:"search"`
}

// WebSearchConfig contains web search tool configuration
type WebSearchConfig struct {
	APIKey     string `mapstructure:"api_key"`
	MaxResults int    `mapstructure:"max_results"`
}

// ExecToolConfig contains shell exec tool configuration
type ExecToolConfig struct {
	Timeout int `mapstructure:"timeout"`
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (*Config, error) {
	// Set defaults
	viper.SetDefault("agents.defaults.model", "anthropic/claude-opus-4-5")
	viper.SetDefault("agents.defaults.max_tokens", 8192)
	viper.SetDefault("agents.defaults.temperature", 0.1)
	viper.SetDefault("agents.defaults.max_tool_iterations", 40)
	viper.SetDefault("agents.defaults.memory_window", 100)
	viper.SetDefault("gateway.host", "0.0.0.0")
	viper.SetDefault("gateway.port", 18790)
	viper.SetDefault("gateway.heartbeat.enabled", true)
	viper.SetDefault("gateway.heartbeat.interval_s", 1800)
	viper.SetDefault("tools.exec.timeout", 60)
	viper.SetDefault("tools.restrict_to_workspace", false)
	viper.SetDefault("channels.send_progress", true)
	viper.SetDefault("channels.send_tool_hints", false)

	// Set config paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	viper.AddConfigPath(filepath.Join(homeDir, ".nanobot"))
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// Unmarshal config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Set default workspace if not configured
	if cfg.Agents.Defaults.Workspace == "" {
		cfg.Agents.Defaults.Workspace = filepath.Join(homeDir, ".nanobot", "workspace")
	}

	return &cfg, nil
}

// GetWorkspacePath returns the expanded workspace path
func (c *Config) GetWorkspacePath() string {
	workspace := c.Agents.Defaults.Workspace
	if len(workspace) > 2 && workspace[:2] == "~/" {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, workspace[2:])
	}
	return workspace
}

// GetAPIKey returns the API key for the given model
func (pc *ProvidersConfig) GetAPIKey(model string) string {
	providerName := getProviderNameForConfig(model, pc)
	if providerName != "" {
		switch providerName {
		case "custom":
			return pc.Custom.APIKey
		case "anthropic":
			return pc.Anthropic.APIKey
		case "openai":
			return pc.OpenAI.APIKey
		case "openrouter":
			return pc.OpenRouter.APIKey
		case "deepseek":
			return pc.DeepSeek.APIKey
		case "groq":
			return pc.Groq.APIKey
		case "zhipu":
			return pc.ZhiPu.APIKey
		case "dashscope":
			return pc.DashScope.APIKey
		case "vllm":
			return pc.VLLM.APIKey
		case "gemini":
			return pc.Gemini.APIKey
		case "moonshot":
			return pc.Moonshot.APIKey
		case "minimax":
			return pc.Minimax.APIKey
		case "aihubmix":
			return pc.AiHubMix.APIKey
		case "siliconflow":
			return pc.SiliconFlow.APIKey
		case "volcengine":
			return pc.VolcEngine.APIKey
		case "openai_codex":
			return pc.OpenAICodex.APIKey
		case "github_copilot":
			return pc.GithubCopilot.APIKey
		}
	}

	// Fallback to first provider with a key
	providers := []struct {
		name string
		key  string
	}{
		{"openrouter", pc.OpenRouter.APIKey},
		{"anthropic", pc.Anthropic.APIKey},
		{"openai", pc.OpenAI.APIKey},
		{"deepseek", pc.DeepSeek.APIKey},
		{"groq", pc.Groq.APIKey},
		{"zhipu", pc.ZhiPu.APIKey},
		{"dashscope", pc.DashScope.APIKey},
		{"vllm", pc.VLLM.APIKey},
		{"gemini", pc.Gemini.APIKey},
		{"moonshot", pc.Moonshot.APIKey},
		{"minimax", pc.Minimax.APIKey},
		{"aihubmix", pc.AiHubMix.APIKey},
		{"siliconflow", pc.SiliconFlow.APIKey},
		{"volcengine", pc.VolcEngine.APIKey},
	}

	for _, p := range providers {
		if p.key != "" {
			return p.key
		}
	}

	return ""
}

// GetAPIBase returns the API base URL for the given model
func (pc *ProvidersConfig) GetAPIBase(model string) string {
	providerName := getProviderNameForConfig(model, pc)
	if providerName != "" {
		switch providerName {
		case "custom":
			return pc.Custom.APIBase
		case "anthropic":
			return pc.Anthropic.APIBase
		case "openai":
			return pc.OpenAI.APIBase
		case "openrouter":
			return pc.OpenRouter.APIBase
		case "deepseek":
			return pc.DeepSeek.APIBase
		case "groq":
			return pc.Groq.APIBase
		case "zhipu":
			return pc.ZhiPu.APIBase
		case "dashscope":
			return pc.DashScope.APIBase
		case "vllm":
			return pc.VLLM.APIBase
		case "gemini":
			return pc.Gemini.APIBase
		case "moonshot":
			return pc.Moonshot.APIBase
		case "minimax":
			return pc.Minimax.APIBase
		case "aihubmix":
			return pc.AiHubMix.APIBase
		case "siliconflow":
			return pc.SiliconFlow.APIBase
		case "volcengine":
			return pc.VolcEngine.APIBase
		case "openai_codex":
			return pc.OpenAICodex.APIBase
		case "github_copilot":
			return pc.GithubCopilot.APIBase
		}
	}

	// Return default API base for gateways if no explicit base is set
	// For this simple implementation, we'll use OpenRouter as the default gateway
	if pc.OpenRouter.APIBase != "" {
		return pc.OpenRouter.APIBase
	}

	// Default to OpenRouter base URL if an OpenRouter key is present
	if pc.OpenRouter.APIKey != "" {
		return "https://openrouter.ai/api/v1"
	}

	return ""
}

// GetProvider returns the matched provider config
func (pc *ProvidersConfig) GetProvider(model string) *ProviderConfig {
	providerName := getProviderNameForConfig(model, pc)
	if providerName != "" {
		switch providerName {
		case "custom":
			return &pc.Custom
		case "anthropic":
			return &pc.Anthropic
		case "openai":
			return &pc.OpenAI
		case "openrouter":
			return &pc.OpenRouter
		case "deepseek":
			return &pc.DeepSeek
		case "groq":
			return &pc.Groq
		case "zhipu":
			return &pc.ZhiPu
		case "dashscope":
			return &pc.DashScope
		case "vllm":
			return &pc.VLLM
		case "gemini":
			return &pc.Gemini
		case "moonshot":
			return &pc.Moonshot
		case "minimax":
			return &pc.Minimax
		case "aihubmix":
			return &pc.AiHubMix
		case "siliconflow":
			return &pc.SiliconFlow
		case "volcengine":
			return &pc.VolcEngine
		case "openai_codex":
			return &pc.OpenAICodex
		case "github_copilot":
			return &pc.GithubCopilot
		}
	}

	// Fallback to first provider with a key
	providers := []*ProviderConfig{
		&pc.OpenRouter, &pc.Anthropic, &pc.OpenAI, &pc.DeepSeek,
		&pc.Groq, &pc.ZhiPu, &pc.DashScope, &pc.VLLM,
		&pc.Gemini, &pc.Moonshot, &pc.Minimax, &pc.AiHubMix,
		&pc.SiliconFlow, &pc.VolcEngine,
	}

	for _, p := range providers {
		if p.APIKey != "" {
			return p
		}
	}

	return nil
}

// Helper function to get provider name from model
func getProviderNameForConfig(model string, pc *ProvidersConfig) string {
	modelLower := strings.ToLower(model)
	parts := strings.Split(modelLower, "/")
	modelPrefix := ""
	if len(parts) > 1 {
		modelPrefix = parts[0]
	}
	normalizedPrefix := strings.ReplaceAll(modelPrefix, "-", "_")

	// Known provider prefixes
	knownPrefixes := map[string]string{
		"custom":         "custom",
		"openrouter":     "openrouter",
		"anthropic":      "anthropic",
		"openai":         "openai",
		"deepseek":       "deepseek",
		"groq":           "groq",
		"zhipu":          "zhipu",
		"dashscope":      "dashscope",
		"vllm":           "vllm",
		"gemini":         "gemini",
		"moonshot":       "moonshot",
		"minimax":        "minimax",
		"aihubmix":       "aihubmix",
		"siliconflow":    "siliconflow",
		"volcengine":     "volcengine",
		"openai_codex":   "openai_codex",
		"github_copilot": "github_copilot",
	}

	// Check if model starts with a known provider prefix
	if provider, exists := knownPrefixes[normalizedPrefix]; exists {
		return provider
	}

	// Look for keywords in the model name
	for provider, keywords := range map[string][]string{
		"openrouter": {"openrouter"},
		"anthropic":  {"claude", "anthropic"},
		"openai":     {"gpt", "openai"},
		"deepseek":   {"deepseek"},
		"groq":       {"groq", "llama", "mixtral", "gemma"},
		"zhipu":      {"glm", "zhipu"},
		"dashscope":  {"qwen", "dashscope"},
		"gemini":     {"gemini"},
		"moonshot":   {"moonshot", "kimi"},
		"minimax":    {"minimax"},
		"volcengine": {"ark", "volcengine"},
	} {
		for _, keyword := range keywords {
			if strings.Contains(modelLower, keyword) {
				return provider
			}
		}
	}

	// Default to openrouter if no specific provider is found but an API key exists
	if pc.OpenRouter.APIKey != "" {
		return "openrouter"
	}

	// Fallback to other providers with keys
	providersWithKeys := []struct {
		name string
		key  string
	}{
		{"anthropic", pc.Anthropic.APIKey},
		{"openai", pc.OpenAI.APIKey},
		{"deepseek", pc.DeepSeek.APIKey},
		{"groq", pc.Groq.APIKey},
	}

	for _, p := range providersWithKeys {
		if p.key != "" {
			return p.name
		}
	}

	return ""
}
