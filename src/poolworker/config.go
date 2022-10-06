package poolworker

type WorkerConfig struct {
	StratumPort     string `yaml:"stratum_port"`
	RPCServer       string `yaml:"kaspad_address"`
	PromPort        string `yaml:"prom_port"`
	HealthCheckPort string `yaml:"health_check_port"`

	PostgresConfig string `yaml:"postgres"`
	RedisConfig    string `yaml:"redis"`
	PoolWallet     string `yaml:"pool_wallet"` // escrow wallet
}
