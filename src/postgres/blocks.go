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

		_, err := conn.Exec(context.Background(), `INSERT into blocks(hash, timestamp, miner, payee, 
			round_time, block_json, bluescore, luck) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			blockhash.String(), time.Now().UTC(), miner, payee, roundTime, string(json), block.Header.BlueScore(), 100)
		if err != nil {
			return errors.Wrapf(err, "failed to record block to databse")
		}
		return nil
	})
}

type BlockStatusType string

const (
	BlockStatusUnconfirmed BlockStatusType = "unconfirmed"
	BlockStatusConfirmed   BlockStatusType = "confirmed"
	BlockStatusPaid        BlockStatusType = "paid"
	BlockStatusError       BlockStatusType = "error"
)

func GetBlocks(ctx context.Context, status BlockStatusType, limit int) ([]*model.Block, error) {
	var fetched []*model.Block
	return fetched, DoQuery(ctx, func(conn *pgx.Conn) error {
		cur, err := conn.Query(ctx, "SELECT block_json from blocks b where b.status = $1 ORDER BY timestamp LIMIT $2", status, limit)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch blocks from database")
		}
		defer cur.Close()

		for {
			if !cur.Next() {
				break
			}
			var data string
			cur.Scan(&data)

			block := &model.Block{}
			if err := json.Unmarshal([]byte(data), block); err != nil {
				continue
			}
			fetched = append(fetched, block)
		}
		return nil
	})
}
