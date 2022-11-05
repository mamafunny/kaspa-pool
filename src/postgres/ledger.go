package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/pkg/errors"
)

func PutPayable(ctx context.Context, blockHash string, balances []model.LedgerEntry) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to begin transaction for PutPayable")
		}
		defer tx.Rollback(ctx)
		for _, v := range balances {
			_, err := conn.Exec(ctx,
				`INSERT into ledger(payee, amount, daascore, status)
						VALUES ($1, $2, $3, $4)`,
				v.Wallet, v.Amount, v.Daascore, v.Status)
			if err != nil {
				return errors.Wrapf(err, "failed to insert ledger")
			}
		}
		if _, err := conn.Exec(ctx,
			`UPDATE blocks SET status = 'payment_pending' 
			 WHERE hash = $1 AND status = 'confirmed'`, blockHash); err != nil {
			return errors.Wrapf(err, "failed to insert ledger")
		}

		return tx.Commit(ctx)
	})
}

func GetLedgerEntriesByStatus(ctx context.Context, status model.LedgerEntryStatus, limit uint64) ([]*model.LedgerEntry, error) {
	var fetched []*model.LedgerEntry
	return fetched, DoQuery(ctx, func(conn *pgx.Conn) error {
		cur, err := conn.Query(ctx,
			`SELECT payee, amount, daascore, status, tx_id
			 FROM ledger l where l.status = $1
			 ORDER BY daascore DESC LIMIT $2`, status, limit)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch blocks from database")
		}
		defer cur.Close()

		for {
			if !cur.Next() {
				break
			}
			entry := &model.LedgerEntry{}
			if err := cur.Scan(&entry.Wallet, &entry.Amount,
				&entry.Daascore, &entry.Status, &entry.TxId); err != nil {
				return errors.Wrap(err, "failed unmarshalling ledger entry")
			}

			fetched = append(fetched, entry)
		}
		return nil
	})
}

func GetPendingForWallet(ctx context.Context, wallet model.KaspaWalletAddr) (uint64, error) {
	var pendingBalance uint64
	return pendingBalance, DoQuery(ctx, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `SELECT  SUM(amount) from ledger  
										WHERE payee = $1 AND status != 'confirmed'`,
			wallet)
		return row.Scan(&pendingBalance)
	})
}

func GetWalletsPendingPayment(ctx context.Context, minPayout, limit uint64) ([]*model.LedgerEntry, error) {
	var fetched []*model.LedgerEntry
	return fetched, DoQuery(ctx, func(conn *pgx.Conn) error {
		cur, err := conn.Query(ctx,
			`SELECT payee, amount 
				FROM (SELECT payee, SUM(amount) as amount 
					FROM ledger l where l.status = 'owed' 
					GROUP BY 1) tt 
				WHERE amount > $1 ORDER BY 2 LIMIT $2`, minPayout*model.KasDigitMultipler, limit)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch blocks from database")
		}
		defer cur.Close()

		for {
			if !cur.Next() {
				break
			}
			entry := &model.LedgerEntry{}
			if err := cur.Scan(&entry.Wallet, &entry.Amount,
				&entry.Daascore, &entry.Status, &entry.TxId); err != nil {
				return errors.Wrap(err, "failed unmarshalling ledger entry")
			}

			fetched = append(fetched, entry)
		}
		return nil
	})
}
