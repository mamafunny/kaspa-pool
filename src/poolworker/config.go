package poolworker

import "github.com/onemorebsmith/kaspa-pool/src/common"

type WorkerConfig struct {
	common.CommonConfig `yaml:",inline"`
	StratumPort         string `yaml:"stratum_port"`
	RedisConfig         string `yaml:"redis"`
}
