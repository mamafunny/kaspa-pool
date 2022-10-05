package poolworker

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/kaspanet/kaspad/domain/consensus/utils/pow"
	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type shareHandler struct {
	kaspa        *KaspaApi
	tipBlueScore uint64
	shareSet     ZSet
	redisClient  *redis.Client
	pgClient     *pgx.Conn
	jobBuffer    *JobManager
}

func newShareHandler(kaspa *KaspaApi, jm *JobManager, rd *redis.Client, pg *pgx.Conn) *shareHandler {
	return &shareHandler{
		kaspa:       kaspa,
		redisClient: rd,
		jobBuffer:   jm,
		shareSet:    NewZSet(rd, "share_buffer"),
		pgClient:    pg,
	}
}

type submitInfo struct {
	block    *appmessage.RPCBlock
	state    *MiningState
	noncestr string
	nonceVal uint64
}

func (sh *shareHandler) validateSubmit(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) (*submitInfo, error) {
	if len(event.Params) < 3 {
		RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
		return nil, fmt.Errorf("malformed event, expected at least 2 params")
	}
	jobIdStr, ok := event.Params[1].(string)
	if !ok {
		RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
		return nil, fmt.Errorf("unexpected type for param 1: %+v", event.Params...)
	}
	jobId, err := strconv.ParseUint(jobIdStr, 10, 32)
	if err != nil {
		RecordWorkerError(ctx.WalletAddr, ErrBadDataFromMiner)
		return nil, errors.Wrap(err, "job id is not parsable as an number")
	}
	state := GetMiningState(ctx)
	block, exists := sh.jobBuffer.GetJob(uint32(jobId))
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
		return postgres.PutShare(sh.pgClient, ctx.WalletAddr, si.block.Header.BlueScore, si.nonceVal, 4)
	}
	return ErrDupeShare
}

func (sh *shareHandler) HandleSubmit(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) error {
	submitInfo, err := sh.validateSubmit(ctx, event)
	if err != nil {
		return err
	}

	ctx.Logger.Debug(fmt.Sprintf("%d submit %s", submitInfo.block.Header.BlueScore, submitInfo.noncestr))
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
			ctx.Logger.Info("dupe share", zap.Error(err))
			RecordDupeShare(ctx)
			return ctx.ReplyDupeShare(event.Id)
		} else if errors.Is(err, ErrStaleShare) {
			ctx.Logger.Info("stale share", zap.Error(err))
			RecordStaleShare(ctx)
			return ctx.ReplyStaleShare(event.Id)
		}
		// unknown error somehow
		ctx.Logger.Info("unknown error during share validation", zap.Error(err))
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
	err := sh.kaspa.SubmitBlock(&externalapi.DomainBlock{
		Header:       mutable.ToImmutable(),
		Transactions: block.Transactions,
	})
	// print after the submit to get it submitted faster
	ctx.Logger.Info(fmt.Sprintf("submitted block to kaspad %s", ctx.String()))
	ctx.Logger.Info(fmt.Sprintf("Submitted nonce: %d", nonce))

	if err != nil {
		// :'(
		if strings.Contains(err.Error(), "ErrDuplicateBlock") {
			ctx.Logger.Warn("block rejected, stale")
			// stale
			RecordStaleShare(ctx)
			return ctx.ReplyStaleShare(eventId)
		} else {
			ctx.Logger.Warn("block rejected, unknown issue (probably bad pow", zap.Error(err))
			RecordInvalidShare(ctx)
			return ctx.ReplyBadShare(eventId)
		}
	}

	// :)
	ctx.Logger.Info("block accepted")
	blockhash := consensushashing.BlockHash(block)
	blockFee := uint64(0)
	for _, t := range block.Transactions {
		if len(t.Inputs) == 0 {
			// no inputs means that the transaction originated from the coinbase
		}
		blockFee += t.Fee
	}

	RecordBlockFound(ctx, block.Header.Nonce(), block.Header.BlueScore(), blockhash.String())
	return ctx.Reply(gostratum.JsonRpcResponse{
		Result: true,
	})
}
