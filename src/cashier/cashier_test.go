package cashier

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/onemorebsmith/kaspa-pool/src/common"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var pg *pgx.Conn
var rd *redis.Client
var logger *zap.Logger

func TestMain(m *testing.M) {
	logger = common.ConfigureZap(zap.DebugLevel)
	var err error
	pg, err = postgres.DefaultDockerConnection()
	if err != nil {
		panic(err)
	}
	rd = redis.NewClient(&redis.Options{
		Addr:     ":6379",
		Password: "eSYzVUxnUzgkb0RLV28meDE0SlVJeDEqd2FwTCVYM05YQVJE",
		DB:       0, // use default DB
	})
	if err := rd.Ping(context.Background()); err.Err() != nil {
		panic(errors.Wrapf(err.Err(), "FATAL, failed to connect to redis at %s", ":6379"))
	}
	rd.Del(context.Background(), "share_buffer")
	defer pg.Close(context.Background())
	defer rd.Close()

	m.Run()
}

var testEffort = postgres.EffortMap{
	"A": 100,
	"B": 500,
	"C": 399,
	"D": 1,
}

func TestPayoutCalculation(t *testing.T) {
	// Generate a buffer of shares for each worker
	totalEffort := uint64(0)
	shares := []string{}
	for k, v := range testEffort {
		totalEffort += v
		for i := uint64(0); i < v; i++ {
			shares = append(shares, k)
		}
	}

	// shuffle the shares in a random, but reproducable way
	rand.Seed(12345678) // reproducable seed
	rand.Shuffle(len(shares), func(i, j int) { shares[i], shares[j] = shares[j], shares[i] })

	// clear any residual data
	if _, err := pg.Exec(context.Background(), "DELETE FROM shares"); err != nil {
		t.Fatalf("failed to clean shares table before test execution: %s", err)
	}

	// put the shares in the db
	i := uint64(0)
	for _, v := range shares {
		i++
		if err := postgres.PutShare(pg, v, 12345, i, 1); err != nil {
			t.Fatal(err)
		}
	}

	// fetch the aggregated shares back from pg, and validate it's what we expect
	effort, err := postgres.GetSharesByWallet(pg, time.Now(), int(totalEffort))
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(testEffort, effort); d != "" {
		t.Fatalf("calculated incorrect effort: %s", d)
	}

	payout := DeterminePayouts(totalEffort, effort) // payout should be 1 to 1 share to $
	if d := cmp.Diff((map[string]uint64)(testEffort), (map[string]uint64)(payout)); d != "" {
		t.Fatalf("calculated incorrect payout: %s", d)
	}
	if err := postgres.PutPayable(pg, payout, 12345); err != nil {
		t.Fatal(err)
	}
}
