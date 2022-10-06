package kaspaapi

import (
	"context"
	"fmt"
	"time"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
	"github.com/onemorebsmith/kaspa-pool/src/common"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type KaspaApi struct {
	address   string
	logger    *zap.Logger
	kaspad    *rpcclient.RPCClient
	connected bool
}

func NewKaspaAPI(address string, logger *zap.Logger) (*KaspaApi, error) {
	client, err := rpcclient.NewRPCClient(address)
	if err != nil {
		return nil, err
	}

	return &KaspaApi{
		address:   address,
		logger:    logger.With(zap.String("component", "kaspaapi:"+address)),
		kaspad:    client,
		connected: true,
	}, nil
}

func (ks *KaspaApi) Start(ctx context.Context, blockCb func()) {
	ks.waitForSync(true)
	go ks.startBlockTemplateListener(ctx, blockCb)
	//go ks.startStatsThread(ctx)
}

// func (ks *KaspaApi) startStatsThread(ctx context.Context) {
// 	ticker := time.NewTicker(30 * time.Second)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			ks.logger.Warn("context cancelled, stopping stats thread")
// 			return
// 		case <-ticker.C:
// 			dagResponse, err := ks.kaspad.GetBlockDAGInfo()
// 			if err != nil {
// 				ks.logger.Warn("failed to get network hashrate from kaspa, prom stats will be out of date", zap.Error(err))
// 				continue
// 			}
// 			response, err := ks.kaspad.EstimateNetworkHashesPerSecond(dagResponse.TipHashes[0], 1000)
// 			if err != nil {
// 				ks.logger.Warn("failed to get network hashrate from kaspa, prom stats will be out of date", zap.Error(err))
// 				continue
// 			}
// 			RecordNetworkStats(response.NetworkHashesPerSecond, dagResponse.BlockCount, dagResponse.Difficulty)
// 		}
// 	}
// }

func (ks *KaspaApi) reconnect() error {
	if ks.kaspad != nil {
		return ks.kaspad.Reconnect()
	}

	client, err := rpcclient.NewRPCClient(ks.address)
	if err != nil {
		return err
	}
	ks.kaspad = client
	return nil
}

func (s *KaspaApi) waitForSync(verbose bool) error {
	if verbose {
		s.logger.Info("checking kaspad sync state")
	}
	for {
		clientInfo, err := s.kaspad.GetInfo()
		if err != nil {
			return errors.Wrapf(err, "error fetching server info from kaspad @ %s", s.address)
		}
		if clientInfo.IsSynced {
			break
		}
		s.logger.Warn("Kaspa is not synced, waiting for sync before starting bridge")
		time.Sleep(5 * time.Second)
	}
	if verbose {
		s.logger.Info("kaspad synced, starting server")
	}
	return nil
}

func (s *KaspaApi) startBlockTemplateListener(ctx context.Context, blockReadyCb func()) {
	blockReadyChan := make(chan bool)
	err := s.kaspad.RegisterForNewBlockTemplateNotifications(func(_ *appmessage.NewBlockTemplateNotificationMessage) {
		blockReadyChan <- true
	})
	if err != nil {
		s.logger.Error("fatal: failed to register for block notifications from kaspa")
	}

	const tickerTime = 500 * time.Millisecond
	ticker := time.NewTicker(tickerTime)
	for {
		if err := s.waitForSync(false); err != nil {
			s.logger.Error("error checking kaspad sync state, attempting reconnect ", zap.Error(err))
			if err := s.reconnect(); err != nil {
				s.logger.Error("error reconnecting to kaspad, waiting before retry ", zap.Error(err))
				time.Sleep(5 * time.Second)
			}
		}
		select {
		case <-ctx.Done():
			s.logger.Warn("context cancelled, stopping block update listener")
			return
		case <-blockReadyChan:
			blockReadyCb()
			ticker.Reset(tickerTime)
		case <-ticker.C: // timeout, manually check for new blocks
			blockReadyCb()
		}
	}
}

var blockSlug = fmt.Sprintf(`onemorebsmith/kaspa-pool_%s`, common.Version)

func (ks *KaspaApi) GetBlockTemplate(addr string) (*appmessage.GetBlockTemplateResponseMessage, error) {
	template, err := ks.kaspad.GetBlockTemplate(addr, blockSlug)
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching new block template from kaspa")
	}
	return template, nil
}

func (ks *KaspaApi) SubmitBlock(block *externalapi.DomainBlock) error {
	_, err := ks.kaspad.SubmitBlock(block)
	return err
}
