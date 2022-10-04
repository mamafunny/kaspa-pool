package poolworker

import "github.com/jackc/pgx"

type WorkerConfig struct {
	StratumPort     string `yaml:"stratum_port"`
	RPCServer       string `yaml:"kaspad_address"`
	PromPort        string `yaml:"prom_port"`
	HealthCheckPort string `yaml:"health_check_port"`

	PostgresConfig PostgresConfig `yaml:"postgres"`
	RedisConfig    RedisConfig    `yaml:"redis"`
	PoolWallet     string         `yaml:"pool_wallet"` // escrow wallet
}

type PostgresConfig struct {
	Host     string       `yaml:"host"`
	Port     uint16       `yaml:"port"`
	Database string       `yaml:"database"`
	User     string       `yaml:"user"`
	Password string       `yaml:"password"`
	LogLevel pgx.LogLevel `yaml:"log_level"`
}

type RedisConfig struct {
}
