package kaspastratum

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type ZSet struct {
	client *redis.Client
	key    string
}

func NewZSet(cache *redis.Client, key string) ZSet {
	return ZSet{
		key:    key,
		client: cache,
	}
}

type ZSetKVP = redis.Z

func (zz *ZSet) AddValues(ctx context.Context, keys ...ZSetKVP) (int64, error) {
	cmd := zz.client.ZAddArgs(ctx, zz.key, redis.ZAddArgs{
		GT:      true,
		NX:      true,
		Members: keys,
	})
	return cmd.Result()
}

func (zz *ZSet) AddValuesWithScore(ctx context.Context, score float64, keys ...string) error {
	zArgs := make([]redis.Z, 0, len(keys))
	for _, k := range keys {
		zArgs = append(zArgs, redis.Z{Member: k, Score: score})
	}

	cmd := zz.client.ZAddArgs(ctx, zz.key, redis.ZAddArgs{
		GT:      true,
		NX:      true,
		Members: zArgs,
	})
	return cmd.Err()
}

func (zz *ZSet) GetValuesByScore(ctx context.Context, min, max int64, limit int64) ([]string, error) {
	data := zz.client.ZRangeByScore(ctx, zz.key, &redis.ZRangeBy{
		Min:   fmt.Sprintf("%d", min),
		Max:   fmt.Sprintf("%d", max),
		Count: limit,
	})
	if data.Err() != nil {
		return nil, data.Err()
	}
	return data.Val(), nil
}

func (zz *ZSet) Count(ctx context.Context) (int64, error) {
	cmd := zz.client.ZCount(ctx, zz.key, "-inf", "+inf")
	return cmd.Val(), cmd.Err()
}

func (zz *ZSet) CountRange(ctx context.Context, min, max float64) (int64, error) {
	cmd := zz.client.ZCount(ctx, zz.key, fmt.Sprintf("%f", min), fmt.Sprintf("%f", max))
	return cmd.Val(), cmd.Err()
}

func (zz *ZSet) Remove(ctx context.Context) (int64, error) {
	cmd := zz.client.ZRem(ctx, zz.key)
	return cmd.Val(), cmd.Err()
}

func (zz *ZSet) RemoveByScore(ctx context.Context, min, max int64) (int64, error) {
	cmd := zz.client.ZRemRangeByScore(ctx, zz.key, fmt.Sprintf("%d", min), fmt.Sprintf("%d", max))
	return cmd.Val(), cmd.Err()
}
