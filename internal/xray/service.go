package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func RuntimeConfig(outbound map[string]any, port int) map[string]any {
	return map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{map[string]any{
			"tag":      "mixed-in",
			"listen":   "127.0.0.1",
			"port":     port,
			"protocol": "mixed",
			"settings": map[string]any{"udp": true},
		}},
		"outbounds": []any{outbound},
	}
}

func Start(ctx context.Context, binary string, outbound map[string]any, port int, runtimeDir string) (int, string, error) {
	if binary == "" {
		binary = "xray"
	}
	if PortOpen(port) {
		return 0, "", fmt.Errorf("端口 %d 已被占用", port)
	}
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return 0, "", err
	}
	configPath := filepath.Join(runtimeDir, "xray.json")
	raw, err := json.MarshalIndent(RuntimeConfig(outbound, port), "", "  ")
	if err != nil {
		return 0, "", err
	}
	if err := os.WriteFile(configPath, raw, 0o600); err != nil {
		return 0, "", err
	}
	validation := exec.CommandContext(ctx, binary, "run", "-test", "-c", configPath)
	if output, err := validation.CombinedOutput(); err != nil {
		return 0, "", fmt.Errorf("xray 配置验证失败: %s", string(output))
	}
	logFile, err := os.OpenFile(filepath.Join(runtimeDir, "xray.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, "", err
	}
	command := exec.Command(binary, "run", "-c", configPath)
	command.Stdout = logFile
	command.Stderr = logFile
	setDetachedProcess(command)
	if err := command.Start(); err != nil {
		logFile.Close()
		return 0, "", fmt.Errorf("启动 xray 失败: %w", err)
	}
	pid := command.Process.Pid
	_ = command.Process.Release()
	_ = logFile.Close()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if PortOpen(port) {
			return pid, configPath, nil
		}
		select {
		case <-ctx.Done():
			return 0, "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	process, _ := os.FindProcess(pid)
	if process != nil {
		_ = process.Kill()
	}
	return 0, "", fmt.Errorf("xray 未在端口 %d 上开始监听，请检查 %s", port, filepath.Join(runtimeDir, "xray.log"))
}

func Stop(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("没有记录运行中的 xray 进程")
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := process.Kill(); err != nil {
		return fmt.Errorf("停止进程 %d 失败: %w", pid, err)
	}
	return nil
}

func PortOpen(port int) bool {
	connection, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond)
	if err != nil {
		return false
	}
	connection.Close()
	return true
}
