package config

import (
	"crypto/rand"
	"encoding/hex"
	"project-tachyon/internal/storage"
	"strconv"
)

// Keys for AppSettings in DB
const (
	KeyEnableAIInterface    = "enable_ai_interface"
	KeyAIToken              = "ai_token"
	KeyEnableIntegrityCheck = "enable_integrity_check"
	KeyAIPort               = "ai_port"
	KeyAIMaxConcurrent      = "ai_max_concurrent"
	KeyUserAgent            = "user_agent"
)

type ConfigManager struct {
	storage *storage.Storage
}

func NewConfigManager(s *storage.Storage) *ConfigManager {
	return &ConfigManager{storage: s}
}

func (c *ConfigManager) GetAIPort() int {
	valStr, err := c.storage.GetString(KeyAIPort)
	if err != nil || valStr == "" {
		return 4444 // Default
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 4444
	}
	return val
}

func (c *ConfigManager) SetAIPort(port int) error {
	return c.storage.SetString(KeyAIPort, strconv.Itoa(port))
}

func (c *ConfigManager) GetAIMaxConcurrent() int {
	valStr, err := c.storage.GetString(KeyAIMaxConcurrent)
	if err != nil || valStr == "" {
		return 5 // Default
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 5
	}
	return val
}

func (c *ConfigManager) SetAIMaxConcurrent(max int) error {
	return c.storage.SetString(KeyAIMaxConcurrent, strconv.Itoa(max))
}

func (c *ConfigManager) GetEnableAI() bool {
	val, err := c.storage.GetString(KeyEnableAIInterface)
	if err != nil {
		return false
	}
	return val == "true"
}

func (c *ConfigManager) SetEnableAI(enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return c.storage.SetString(KeyEnableAIInterface, val)
}

func (c *ConfigManager) GetAIToken() string {
	val, err := c.storage.GetString(KeyAIToken)
	if err != nil || val == "" {
		// Generate if missing
		token := generateSecureToken()
		c.storage.SetString(KeyAIToken, token)
		return token
	}
	return val
}

func (c *ConfigManager) GetEnableIntegrityCheck() bool {
	val, err := c.storage.GetString(KeyEnableIntegrityCheck)
	if err != nil {
		return true // Default True
	}
	return val != "false"
}

func (c *ConfigManager) SetEnableIntegrityCheck(enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return c.storage.SetString(KeyEnableIntegrityCheck, val)
}

func generateSecureToken() string {
	b := make([]byte, 16) // 16 bytes = 32 hex chars
	if _, err := rand.Read(b); err != nil {
		// Fallback (extremely unlikely)
		return "tachyon-fallback-token-change-me"
	}
	return hex.EncodeToString(b)
}

// GetUserAgent returns the custom User-Agent string
// Returns empty string if not set (caller should use default)
func (c *ConfigManager) GetUserAgent() string {
	val, err := c.storage.GetString(KeyUserAgent)
	if err != nil {
		return "" // Use default
	}
	return val
}

// SetUserAgent stores a custom User-Agent string
func (c *ConfigManager) SetUserAgent(ua string) error {
	return c.storage.SetString(KeyUserAgent, ua)
}

// FactoryReset resets all configuration to defaults
func (c *ConfigManager) FactoryReset() error {
	// We just delete the keys, so getters will return defaults
	keys := []string{
		KeyEnableAIInterface,
		KeyAIToken,
		KeyEnableIntegrityCheck,
		KeyAIPort,
		KeyAIMaxConcurrent,
		KeyUserAgent,
	}

	for _, key := range keys {
		// Set to empty string effectively resets it (or we could use a DeleteString if we had one)
		// Since we don't have DeleteString in Storage interface exposed here yet (it only has DeleteTask/Location),
		// we can set to empty. Getters check for empty string.
		if err := c.storage.SetString(key, ""); err != nil {
			return err
		}
	}
	return nil
}
