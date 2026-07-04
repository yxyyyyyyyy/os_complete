package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr           string
	Mode               string
	DataDir            string
	SocketPath         string
	WorkerCommand      string
	HeartbeatTimeoutMS int
	CgroupRoot         string
}

func Load(path string) (Config, error) {
	cfg := Config{
		HTTPAddr:           "127.0.0.1:8080",
		Mode:               "mock",
		DataDir:            ".aort-dev",
		HeartbeatTimeoutMS: 6000,
	}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Config{}, fmt.Errorf("invalid config line %q", line)
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "http_addr":
			cfg.HTTPAddr = value
		case "mode":
			cfg.Mode = value
		case "data_dir":
			cfg.DataDir = value
		case "socket_path":
			cfg.SocketPath = value
		case "worker_command":
			cfg.WorkerCommand = value
		case "heartbeat_timeout_ms":
			var timeout int
			if _, err := fmt.Sscanf(value, "%d", &timeout); err != nil {
				return Config{}, fmt.Errorf("invalid heartbeat_timeout_ms %q", value)
			}
			cfg.HeartbeatTimeoutMS = timeout
		case "cgroup_root":
			cfg.CgroupRoot = value
		default:
			return Config{}, fmt.Errorf("unknown config key %q", strings.TrimSpace(key))
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
