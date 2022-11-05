package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/pkg/errors"
)

func PutTransactions(ctx context.Context, transactions []model.RawKaspaTransaction) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to begin transaction for PutPayable")
		}
		defer tx.Rollback(ctx)
		for _, v := range transactions {
			// the 'verbose' data sometimes has the transaction id, but it is flaky. Compute it
			// manually so we can reliably keep the index
			domain, err := appmessage.RPCTransactionToDomainTransaction(v)
			if err != nil {
				return errors.Wrap(err, "failed changing domain rpc to domain transaction")
			}
			txid := consensushashing.TransactionID(domain)
			encoded, err := json.Marshal(v)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal transaction to json")
			}
			_, err = conn.Exec(ctx,
				`INSERT into global_tx_ledger(tx_id, raw)
						VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				txid.String(), encoded)
			if err != nil {
				return errors.Wrapf(err, "failed to insert ledger")
			}
		}
		return tx.Commit(ctx)
	})
}
