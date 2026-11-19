package main

import (
	"os"
	"path/filepath"
)

// writeTempConfig 将内嵌的 config.yaml 写入临时文件并返回路径
func writeTempConfig() (string, error) {
	dir := filepath.Join(os.TempDir(), "hr-webapp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, configYAML, 0o644); err != nil {
		return "", err
	}
	return p, nil
}
