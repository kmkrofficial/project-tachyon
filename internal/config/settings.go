package config

import (
	"crypto/rand"
	"encoding/hex"
	"project-tachyon/internal/storage"
)

// Keys for AppSettings in DB
const (
	KeyEnableAIInterface    = "enable_ai_interface"
	KeyAIToken              = "ai_token"
	KeyEnableIntegrityCheck = "enable_integrity_check"
)

type ConfigManager struct {
	storage *storage.Storage
}

func NewConfigManager(s *storage.Storage) *ConfigManager {
	return &ConfigManager{storage: s}
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
