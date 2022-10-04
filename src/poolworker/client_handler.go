package poolworker

import (
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var bigJobRegex = regexp.MustCompile(".*BzMiner.*")

const balanceDelay = time.Minute

type clientListener struct {
	logger        *zap.Logger
	shareHandler  *shareHandler
	clientLock    sync.RWMutex
	clients       map[int32]*gostratum.StratumContext
	clientCounter int32
}

func newClientListener(logger *zap.Logger, shareHandler *shareHandler) *clientListener {
	return &clientListener{
		logger:       logger,
		clientLock:   sync.RWMutex{},
		shareHandler: shareHandler,
		clients:      make(map[int32]*gostratum.StratumContext),
	}
}

func (c *clientListener) OnConnect(ctx *gostratum.StratumContext) {
	idx := atomic.AddInt32(&c.clientCounter, 1)
	ctx.Id = idx
	c.clientLock.Lock()
	c.clients[idx] = ctx
	c.clientLock.Unlock()
	ctx.Logger = ctx.Logger.With(zap.Int("client_id", int(ctx.Id)))
}

func (c *clientListener) OnDisconnect(ctx *gostratum.StratumContext) {
	ctx.Done()
	c.clientLock.Lock()
	c.logger.Info(fmt.Sprintf("removing client %d", ctx.Id))
	delete(c.clients, ctx.Id)
	c.logger.Info(fmt.Sprintf("removed client %d", ctx.Id))
	c.clientLock.Unlock()
	RecordDisconnect(ctx)
}

func (c *clientListener) NewJobAvailable(job *WorkJob) {
	c.clientLock.Lock()
	for _, c := range c.clients {
		if !c.Connected() {
			continue
		}
		go func(client *gostratum.StratumContext) {
			state := GetMiningState(client)
			if client.WalletAddr == "" {
				if time.Since(state.connectTime) > time.Second*20 { // timeout passed
					// this happens pretty frequently in gcp/aws land since script-kiddies scrape ports
					client.Logger.Warn(fmt.Sprintf("client %s misconfigured, no miner address specified - disconnecting", client.String()))
					RecordWorkerError(client.WalletAddr, ErrNoMinerAddress)
					client.Disconnect() // invalid configuration, boot the worker
				}
				return
			}
			if !state.initialized {
				state.initialized = true
				state.useBigJob = bigJobRegex.MatchString(client.RemoteApp)
				// first pass through send the difficulty since it's fixed
				if err := client.Send(gostratum.JsonRpcEvent{
					Version: "2.0",
					Method:  "mining.set_difficulty",
					Params:  []any{fixedDifficulty},
				}); err != nil {
					RecordWorkerError(client.WalletAddr, ErrFailedSetDiff)
					client.Logger.Error(errors.Wrap(err, "failed sending difficulty").Error(), zap.Any("context", client))
					return
				}
			}

			var jobParams []any
			if state.useBigJob {
				jobParams = job.BigJobParams
			} else {
				jobParams = job.NormalJobParams
			}

			// // normal notify flow
			if err := client.Send(gostratum.JsonRpcEvent{
				Version: "2.0",
				Method:  "mining.notify",
				Id:      job.Id,
				Params:  jobParams,
			}); err != nil {
				if errors.Is(err, gostratum.ErrorDisconnected) {
					RecordWorkerError(client.WalletAddr, ErrDisconnected)
					return
				}
				RecordWorkerError(client.WalletAddr, ErrFailedSendWork)
				client.Logger.Error(errors.Wrapf(err, "failed sending work packet %d", job.Id).Error())
			}

			RecordNewJob(client)
		}(c)
	}
	c.clientLock.Unlock()
}
