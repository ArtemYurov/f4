package main

import (
	"github.com/unxed/f4/internal/config"
	"path/filepath"
	"testing"
)

func TestConfig_FallbackLanguagePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	oldCfg := config.App
	oldGetUserConfigIniPath := config.GetUserConfigIniPath
	oldGetConfigPaths := config.GetConfigIniPaths
	defer func() {
		config.App = oldCfg
		config.GetUserConfigIniPath = oldGetUserConfigIniPath
		config.GetConfigIniPaths = oldGetConfigPaths
	}()
	config.GetUserConfigIniPath = func() string {
		return filepath.Join(tmpDir, "settings.ini")
	}
	config.GetConfigIniPaths = func() []string {
		return []string{filepath.Join(tmpDir, "settings.ini")}
	}

	config.App.Language = "ka"
	config.App.FallbackLanguage = "ru"
	config.SaveConfig()

	// Reset in-memory values
	config.App.Language = ""
	config.App.FallbackLanguage = ""

	config.LoadConfig()

	if config.App.Language != "ka" {
		t.Errorf("expected Primary Language 'ka', got '%s'", config.App.Language)
	}
	if config.App.FallbackLanguage != "ru" {
		t.Errorf("expected Fallback Language 'ru', got '%s'", config.App.FallbackLanguage)
	}
}
