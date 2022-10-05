package cashier

import (
	"github.com/jackc/pgx/v5"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
	"github.com/kaspanet/kaspad/util"
)

func HandleMinedBlock(payTo string, pg *pgx.Conn, block *externalapi.DomainBlock) {
	value, fee := determineBlockValue(payTo, block)
	_ = value
	_ = fee
}

func determineBlockValue(payTo string, block *externalapi.DomainBlock) (blockValue, feeValue uint64) {
	addr, _ := util.DecodeAddress("kaspa:qzk3uh2twkhu0fmuq50mdy3r2yzuwqvstq745hxs7tet25hfd4egcafcdmpdl", util.Bech32PrefixKaspa)
	script, _ := txscript.PayToAddrScript(addr)
	for _, t := range block.Transactions {
		if len(t.Inputs) == 0 {
			// no inputs means the transaction orignated from the coinbase (aka miner reward)
			for _, o := range t.Outputs {
				if o.ScriptPublicKey.Equal(script) {
					blockValue += o.Value
				}
			}
		}
		feeValue += t.Fee
	}
	return blockValue, feeValue
}
