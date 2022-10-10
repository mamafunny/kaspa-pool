package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/pkg/errors"
)

func PutBlock(ctx context.Context, block *externalapi.DomainBlock, miner string, payee string, roundTime time.Duration) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		blockhash := consensushashing.BlockHash(block)
		json, _ := json.Marshal(appmessage.DomainBlockToRPCBlock(block)) // domain blocks don't have public members

		_, err := conn.Exec(context.Background(),
			`INSERT into blocks(hash, timestamp, miner, payee, round_time, block_json, bluescore, daascore, luck) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			blockhash.String(), time.Now().UTC(), miner, payee,
			roundTime, string(json), block.Header.BlueScore(), block.Header.DAAScore(), 100)
		if err != nil {
			return errors.Wrapf(err, "failed to record block to databse")
		}
		return nil
	})
}

func GetUnconfirmedBlocks(ctx context.Context, limit int) ([]*model.UnconfirmedBlock, error) {
	var fetched []*model.UnconfirmedBlock
	return fetched, DoQuery(ctx, func(conn *pgx.Conn) error {
		cur, err := conn.Query(ctx,
			`SELECT hash, payee, block_json 
			 FROM blocks b where b.status = 'unconfirmed' 
			 ORDER BY timestamp LIMIT $1`, limit)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch blocks from database")
		}
		defer cur.Close()

		for {
			if !cur.Next() {
				break
			}
			var hash, wallet, data string
			cur.Scan(&hash, &wallet, &data)

			block := &model.UnconfirmedBlock{
				Hash:   hash,
				Wallet: wallet,
			}
			if err := json.Unmarshal([]byte(data), block); err != nil {
				continue
			}
			fetched = append(fetched, block)
		}
		return nil
	})
}

func GetConfirmedBlocks(ctx context.Context, limit int) ([]*model.ConfirmedBlock, error) {
	var fetched []*model.ConfirmedBlock
	return fetched, DoQuery(ctx, func(conn *pgx.Conn) error {
		cur, err := conn.Query(ctx,
			`SELECT hash, coinbase_reward, cp.amount, cp.wallet, block_json from blocks b 
				JOIN coinbase_payments cp ON cp.tx = b.coinbase_reward
				WHERE b.status = 'confirmed' 
				ORDER BY timestamp LIMIT $1`, limit)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch blocks from database")
		}
		defer cur.Close()

		for {
			if !cur.Next() {
				break
			}
			var hash, txHash, wallet, data string
			payment := uint64(0)
			cur.Scan(&hash, &txHash, &payment, &wallet, &data)

			block := &model.ConfirmedBlock{
				UnconfirmedBlock: model.UnconfirmedBlock{Hash: hash},
				CoinbasePayment: &model.CoinbasePayment{
					TxId:   txHash,
					Wallet: wallet,
					Amount: payment,
				},
			}
			if err := json.Unmarshal([]byte(data), block); err != nil {
				continue
			}
			fetched = append(fetched, block)
		}
		return nil
	})
}

func UpdateBlockCoinbaseTransaction(ctx context.Context, block *model.ConfirmedBlock) error {
	return DoQuery(ctx, func(conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `UPDATE blocks b SET coinbase_reward = $1, status = 'confirmed'
					WHERE status = 'unconfirmed' AND hash = $2`, block.CoinbasePayment.TxId, block.UnconfirmedBlock.Hash)
		return err
	})
}
