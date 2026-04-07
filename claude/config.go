package claude

import (
	"time"

	"whatsapp-mcp/config"
)

// TriggerConfig holds configuration for the @claude trigger feature.
type TriggerConfig struct {
	Enabled            bool
	ClaudePath         string        // path to claude CLI binary
	Model              string        // optional model override
	CompactModel       string        // model used for context compaction summarization
	CompactThreshold   int           // input-token threshold above which the next turn triggers compaction
	MaxBudget          string        // max budget per invocation in USD
	Timeout            time.Duration // max time to wait for Claude response
	MCPPort            string        // WhatsApp MCP server port
	MCPAPIKey          string        // WhatsApp MCP server API key
}

// LoadConfig loads trigger configuration from environment variables.
func LoadConfig() TriggerConfig {
	return TriggerConfig{
		Enabled:          config.GetEnvBool("CLAUDE_TRIGGER_ENABLED", false),
		ClaudePath:       config.GetEnv("CLAUDE_PATH", "claude"),
		Model:            config.GetEnv("CLAUDE_MODEL", ""),
		CompactModel:     config.GetEnv("CLAUDE_COMPACT_MODEL", "haiku"),
		CompactThreshold: config.GetEnvInt("CLAUDE_COMPACT_THRESHOLD_TOKENS", 120000),
		MaxBudget:        config.GetEnv("CLAUDE_MAX_BUDGET_USD", "1.00"),
		Timeout:          time.Duration(config.GetEnvInt("CLAUDE_TIMEOUT_SECONDS", 300)) * time.Second,
		MCPPort:          config.GetEnv("MCP_PORT", "8080"),
		MCPAPIKey:        config.GetEnv("MCP_API_KEY", "change-me-in-production"),
	}
}
