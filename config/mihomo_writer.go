package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// WriteMihomoConfig generates config/config.yaml.
// Mihomo reads SubMill's node output directly via file provider ? no subscription URLs needed.
func WriteMihomoConfig() error {
	path := filepath.Join(getExecutableDir(), "config", "config.yaml")

	// Don't overwrite existing config
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	cfg := map[string]any{
		"mixed-port": 20171,
		"bind-address": "*",
		"allow-lan": true,
		"mode":       "rule",
		"log-level":  "info",
		"ipv6":          false,
		"geo-auto-update": false,
		"geo-update-interval": 99999,
		"profile": map[string]any{
			"store-selected": true,
			"store-fake-ip":  true,
		},
		"dns": map[string]any{
			"enable":        true,
			"ipv6":          false,
			"enhanced-mode": "fake-ip",
			"fake-ip-range": "198.18.0.1/16",
			"nameserver":    []string{"223.5.5.5", "119.29.29.29"},
		},
		"proxy-providers": map[string]any{
			"submill": map[string]any{
				"type": "file",
				"path": "./nodes/all.yaml",
				"health-check": map[string]any{
					"enable":   true,
					"url":      "http://www.gstatic.com/generate_204",
					"interval": 300,
				},
			},
		},
		"proxy-groups": []map[string]any{
			{
				"name":    "PROXY",
				"type":    "select",
				"proxies": []string{"auto", "balance", "DIRECT"},
			},
			{
				"name":      "auto",
				"type":      "url-test",
				"use":       []string{"submill"},
				"url":       "http://www.gstatic.com/generate_204",
				"interval":  300,
				"tolerance": 20,
			},
			{
				"name":     "balance",
				"type":     "load-balance",
				"use":      []string{"submill"},
				"url":      "http://www.gstatic.com/generate_204",
				"interval": 300,
				"strategy": "consistent-hashing",
			},
			{
				"name":    "FALLBACK",
				"type":    "fallback",
				"proxies": []string{"auto", "balance", "DIRECT"},
			},
		},
		"rules": []string{
			"IP-CIDR,192.168.0.0/16,DIRECT",
			"IP-CIDR,10.0.0.0/8,DIRECT",
			"IP-CIDR,172.16.0.0/12,DIRECT",
			"IP-CIDR,127.0.0.0/8,DIRECT",
			"MATCH,PROXY",
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal mihomo config: %w", err)
	}

	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write mihomo config: %w", err)
	}
	return nil
}
// getExecutableDir returns the directory of the running executable.
func getExecutableDir() string {
	ex, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(ex)
}
