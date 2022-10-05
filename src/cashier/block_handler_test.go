package cashier

import (
	"log"
	"math/big"
	"testing"

	"github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
	"github.com/kaspanet/kaspad/util"
)

func TestBlockPayments(t *testing.T) {
	addr, _ := util.DecodeAddress("kaspa:qzk3uh2twkhu0fmuq50mdy3r2yzuwqvstq745hxs7tet25hfd4egcafcdmpdl", util.Bech32PrefixKaspa)
	script, _ := txscript.PayToAddrScript(addr)

	i := big.Int{}
	i.SetString(script.String(), 16)
	log.Printf("%x", script.Script)
}
