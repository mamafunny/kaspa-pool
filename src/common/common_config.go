package common

type CommonConfig struct {
	RPCServer       string `yaml:"kaspad_address"`
	PromPort        string `yaml:"prom_port"`
	HealthCheckPort string `yaml:"health_check_port"`
	PostgresConfig  string `yaml:"postgres"`
	PoolWallet      string `yaml:"pool_wallet"`
}
