package kaspaapi

import (
	"context"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/client"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/pb"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type KaspawalletApi struct {
	address string
	logger  *zap.Logger
}

func NewKaspawalletApi(daemonAddress string, logger *zap.Logger) KaspawalletApi {
	return KaspawalletApi{
		address: daemonAddress,
		logger:  logger.With(zap.String("address", daemonAddress), zap.String("component", "kaspawallet_api")),
	}
}

func (ks *KaspawalletApi) SendKas(ctx context.Context, to, from model.KaspaWalletAddr, amount uint64, pass string) (*model.MinerPayoutTransction, error) {
	ks.logger.Info("sending kas", zap.String("to", string(to)), zap.String("from", string(from)), zap.Uint64("amount", amount))
	walletClient, deferMe, err := client.Connect(ks.address)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect wallet client to kaspad")
	}
	defer deferMe()
	resp, err := walletClient.Send(ctx, &pb.SendRequest{
		ToAddress:                string(to),
		Amount:                   amount,
		Password:                 pass,
		From:                     []string{string(from)},
		UseExistingChangeAddress: true,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send %d kas to wallet %s from %s", amount, to, from)
	}

	if len(resp.TxIDs) == 0 {
		return nil, errors.Wrapf(err, "No tx ids returned after sending %d kas to wallet %s from %s", amount, to, from)
	}
	ks.logger.Info("sent kas", zap.String("to", string(to)), zap.String("from", string(from)), zap.Uint64("amount", amount))

	return &model.MinerPayoutTransction{
		Amount: amount,
		TxId:   resp.TxIDs[0],
		Raw:    resp,
	}, nil
}
