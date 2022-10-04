package poolworker

import (
	"time"

	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
)

type MiningState struct {
	initialized bool
	useBigJob   bool
	connectTime time.Time
}

func MiningStateGenerator() any {
	return &MiningState{
		connectTime: time.Now(),
	}
}

func GetMiningState(ctx *gostratum.StratumContext) *MiningState {
	return ctx.State.(*MiningState)
}
