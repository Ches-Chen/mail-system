package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	IsDebug     bool
	HttpPort    uint
	SwagHost    string
	DatabaseDSN string `mapstructure:"database_dsn"`
}

var globalConfig *Config

func GlobalConfig() *Config {
	return globalConfig
}

func SetDebug(debug bool) {
	globalConfig.IsDebug = debug
}

func SetHttpPort(port uint) {
	globalConfig.HttpPort = port
}

func SetSwagHost(host string) {
	globalConfig.SwagHost = host
}

func InitConfig() error {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("MSPS")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 确保配置正确解析
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证必须字段
	if cfg.DatabaseDSN == "" {
		return fmt.Errorf("缺少数据库连接配置")
	}

	globalConfig = &cfg
	return nil
}
