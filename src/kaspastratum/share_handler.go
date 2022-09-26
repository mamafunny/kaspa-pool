package kaspastratum

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/pow"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
	"github.com/pkg/errors"
)

type shareHandler struct {
	kaspa        *rpcclient.RPCClient
	statsLock    sync.Mutex
	tipBlueScore uint64
	redis        redis.Client
	shareSet     ZSet
	postgres     *pgx.Conn
}

func newShareHandler(kaspa *rpcclient.RPCClient, redis *redis.Client, pg *pgx.Conn) *shareHandler {
	return &shareHandler{
		kaspa:     kaspa,
		statsLock: sync.Mutex{},
		redis:     *redis,
		shareSet:  NewZSet(redis, "share_buffer"),
		postgres:  pg,
	}
}

type submitInfo struct {
	block    *appmessage.RPCBlock
	state    *MiningState
	noncestr string
	nonceVal uint64
}

func validateSubmit(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) (*submitInfo, error) {
	if len(event.Params) < 2 {
		RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
		return nil, fmt.Errorf("malformed event, expected at least 2 params")
	}
	jobIdStr, ok := event.Params[1].(string)
	if !ok {
		RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
		return nil, fmt.Errorf("unexpected type for param 1: %+v", event.Params...)
	}
	jobId, err := strconv.ParseInt(jobIdStr, 10, 0)
	if err != nil {
		RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
		return nil, errors.Wrap(err, "job id is not parsable as an number")
	}
	state := GetMiningState(ctx)
	block, exists := state.GetJob(int(jobId))
	if !exists {
		RecordWorkerError(ctx.WalletAddr, ErrMissingJob)
		return nil, fmt.Errorf("job does not exist. stale?")
	}
	noncestr, ok := event.Params[2].(string)
	if !ok {
		RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
		return nil, fmt.Errorf("unexpected type for param 2: %+v", event.Params...)
	}
	return &submitInfo{
		state:    state,
		block:    block,
		noncestr: strings.Replace(noncestr, "0x", "", 1),
	}, nil
}

var (
	ErrStaleShare = fmt.Errorf("stale share")
	ErrDupeShare  = fmt.Errorf("duplicate share")
)

// the max difference between tip blue score and job blue score that we'll accept
// anything greater than this is considered a stale
const workWindow = 8

func (sh *shareHandler) checkStales(ctx *gostratum.StratumContext, si *submitInfo) error {
	tip := sh.tipBlueScore
	if si.block.Header.BlueScore > tip {
		sh.tipBlueScore = si.block.Header.BlueScore
	} else if tip-si.block.Header.BlueScore > workWindow {
		RecordStaleShare(ctx)
		return errors.Wrapf(ErrStaleShare, "blueScore %d vs %d", si.block.Header.BlueScore, tip)
	}
	val, err := sh.shareSet.AddValues(ctx, ZSetKVP{
		Score:  float64(time.Now().Unix()),
		Member: fmt.Sprintf("%d_%d", si.block.Header.BlueScore, si.nonceVal),
	})
	if err != nil {
		return errors.Wrap(err, "failed writing share to redis")
	}
	if val > 0 { // val > 0 means new value
		// credit to miner
		_, err := sh.postgres.Exec(`INSERT into shares(wallet, bluescore, nonce, timestamp) 
									 VALUES ($1, $2, $3, $4)`,
			ctx.WalletAddr, si.block.Header.BlueScore, si.nonceVal, time.Now())
		if err != nil {
			return errors.Wrap(err, "failed writing share to pg")
		}
		return nil
	}
	return ErrDupeShare
}

func (sh *shareHandler) HandleSubmit(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) error {
	submitInfo, err := validateSubmit(ctx, event)
	if err != nil {
		return err
	}
	ctx.Logger.Debug(submitInfo.block.Header.BlueScore, " submit ", submitInfo.noncestr)
	if GetMiningState(ctx).useBigJob {
		submitInfo.nonceVal, err = strconv.ParseUint(submitInfo.noncestr, 16, 64)
		if err != nil {
			RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
			return errors.Wrap(err, "failed parsing noncestr")
		}
	} else {
		submitInfo.nonceVal, err = strconv.ParseUint(submitInfo.noncestr, 16, 64)
		if err != nil {
			RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
			return errors.Wrap(err, "failed parsing noncestr")
		}
	}
	if err := sh.checkStales(ctx, submitInfo); err != nil {
		if err == ErrDupeShare {
			ctx.Logger.Info("dupe share "+submitInfo.noncestr, ctx.WorkerName, ctx.WalletAddr)
			RecordDupeShare(ctx)
			return ctx.ReplyDupeShare(event.Id)
		} else if errors.Is(err, ErrStaleShare) {
			ctx.Logger.Info(err.Error(), ctx.WorkerName, ctx.WalletAddr)
			RecordStaleShare(ctx)
			return ctx.ReplyStaleShare(event.Id)
		}
		// unknown error somehow
		ctx.Logger.Error("unknown error during check stales: ", err.Error())
		return ctx.ReplyBadShare(event.Id)
	}

	converted, err := appmessage.RPCBlockToDomainBlock(submitInfo.block)
	if err != nil {
		return fmt.Errorf("failed to cast block to mutable block: %+v", err)
	}
	mutableHeader := converted.Header.ToMutable()
	mutableHeader.SetNonce(submitInfo.nonceVal)
	powState := pow.NewState(mutableHeader)
	powValue := powState.CalculateProofOfWorkValue()

	// The block hash must be less or equal than the claimed target.
	if powValue.Cmp(&powState.Target) <= 0 {
		return sh.submit(ctx, converted, submitInfo.nonceVal, event.Id)
	}
	// remove for now until I can figure it out. No harm here as we're not
	// } else if powValue.Cmp(fixedDifficultyBI) >= 0 {
	// 	ctx.Logger.Warn("weak block")
	// 	RecordWeakShare(ctx)
	// 	return ctx.ReplyLowDiffShare(event.Id)
	// }
	RecordShareFound(ctx)

	return ctx.Reply(gostratum.JsonRpcResponse{
		Id:     event.Id,
		Result: true,
	})
}

func (sh *shareHandler) submit(ctx *gostratum.StratumContext,
	block *externalapi.DomainBlock, nonce uint64, eventId any) error {
	mutable := block.Header.ToMutable()
	mutable.SetNonce(nonce)
	_, err := sh.kaspa.SubmitBlock(&externalapi.DomainBlock{
		Header:       mutable.ToImmutable(),
		Transactions: block.Transactions,
	})
	// print after the submit to get it submitted faster
	ctx.Logger.Info("submitted block to kaspad")
	ctx.Logger.Info(fmt.Sprintf("Submitted nonce: %d", nonce))

	if err != nil {
		// :'(
		if strings.Contains(err.Error(), "ErrDuplicateBlock") {
			ctx.Logger.Warn("block rejected, stale")
			// stale
			RecordStaleShare(ctx)
			return ctx.ReplyStaleShare(eventId)
		} else {
			ctx.Logger.Warn("block rejected, unknown issue (probably bad pow")
			RecordInvalidShare(ctx)
			return ctx.ReplyBadShare(eventId)
		}
	}

	// :)
	ctx.Logger.Info("block accepted")
	RecordBlockFound(ctx)
	return ctx.Reply(gostratum.JsonRpcResponse{
		Result: true,
	})
}
