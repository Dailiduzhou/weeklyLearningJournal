package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironmentOverridesFileAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("model:\n  api_key: file-key\n  name: file-model\nagent:\n  max_rounds: 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_MODEL__API_KEY", "env-key")
	t.Setenv("AGENT_MODEL__NAME", "env-model")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model.APIKey != "env-key" || cfg.Model.Name != "env-model" || cfg.Agent.MaxRounds != 3 || cfg.Agent.ToolTimeout <= 0 {
		t.Fatalf("unexpected precedence result: %+v", cfg)
	}
}

func TestDatabaseRequiresDSNWhenEnabled(t *testing.T) {
	t.Setenv("AGENT_MODEL__API_KEY", "test-key")
	t.Setenv("AGENT_DATABASE__ENABLED", "true")
	if _, err := Load(""); err == nil {
		t.Fatal("expected missing database DSN to fail validation")
	}
}
