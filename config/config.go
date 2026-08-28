package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pharmacycounter/domain"
)

type Config struct {
	Address             string
	DatabasePath        string
	StaticPath          string
	Counters            []domain.Counter
	ReadTimeoutSeconds  int
	WriteTimeoutSeconds int
}

func Default() Config {
	return Config{
		Address:             ":8080",
		DatabasePath:        "data/pharmacy.db",
		StaticPath:          "web/dist",
		ReadTimeoutSeconds:  10,
		WriteTimeoutSeconds: 15,
		Counters: []domain.Counter{
			{Code: "C01", Name: "一号发药窗", Enabled: true, QueueLimit: 4},
			{Code: "C02", Name: "二号发药窗", Enabled: true, QueueLimit: 4},
			{Code: "C03", Name: "优先发药窗", Enabled: true, QueueLimit: 3},
		},
	}
}

func FromEnvironment() (Config, error) {
	result := Default()
	if value := strings.TrimSpace(os.Getenv("PHARMACY_ADDRESS")); value != "" {
		result.Address = value
	}
	if value := strings.TrimSpace(os.Getenv("PHARMACY_DATABASE")); value != "" {
		result.DatabasePath = value
	}
	if value := strings.TrimSpace(os.Getenv("PHARMACY_STATIC")); value != "" {
		result.StaticPath = value
	}
	if value := strings.TrimSpace(os.Getenv("PHARMACY_READ_TIMEOUT")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("解析读取超时: %w", err)
		}
		result.ReadTimeoutSeconds = seconds
	}
	if value := strings.TrimSpace(os.Getenv("PHARMACY_WRITE_TIMEOUT")); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("解析写入超时: %w", err)
		}
		result.WriteTimeoutSeconds = seconds
	}
	if err := result.Validate(); err != nil {
		return Config{}, err
	}
	return result, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Address) == "" {
		return errors.New("监听地址不能为空")
	}
	if strings.TrimSpace(c.DatabasePath) == "" {
		return errors.New("数据库路径不能为空")
	}
	if c.ReadTimeoutSeconds < 1 || c.ReadTimeoutSeconds > 300 {
		return errors.New("读取超时必须在 1 到 300 秒之间")
	}
	if c.WriteTimeoutSeconds < 1 || c.WriteTimeoutSeconds > 300 {
		return errors.New("写入超时必须在 1 到 300 秒之间")
	}
	if len(c.Counters) == 0 {
		return errors.New("至少需要一个发药窗口")
	}
	seen := make(map[string]bool)
	for _, counter := range c.Counters {
		if counter.Code == "" || counter.Name == "" {
			return errors.New("窗口代码和名称不能为空")
		}
		if seen[counter.Code] {
			return errors.New("窗口代码不能重复")
		}
		if counter.QueueLimit < 1 {
			return errors.New("窗口队列容量必须大于零")
		}
		seen[counter.Code] = true
	}
	return nil
}

func (c Config) AbsoluteDatabasePath(base string) string {
	if filepath.IsAbs(c.DatabasePath) {
		return filepath.Clean(c.DatabasePath)
	}
	return filepath.Join(base, c.DatabasePath)
}

func (c Config) EnabledCounterCodes() []string {
	result := make([]string, 0)
	for _, counter := range c.Counters {
		if counter.Enabled {
			result = append(result, counter.Code)
		}
	}
	return result
}
