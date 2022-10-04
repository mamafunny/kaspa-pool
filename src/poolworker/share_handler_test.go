package poolworker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var pg *pgx.Conn
var rd *redis.Client
var logger *zap.Logger

func TestMain(m *testing.M) {
	logger, _ = configureZap(WorkerConfig{})
	var err error
	pg, err = pgx.Connect(pgx.ConnConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "kaspa-pool",
	})
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
	defer pg.Close()
	defer rd.Close()

	m.Run()
}

type testContext struct {
	ctx    *gostratum.StratumContext
	conn   *gostratum.MockConnection
	block  *appmessage.RPCBlock
	jobMan *JobManager
	job    *WorkJob
}

func loadExampleHeader() *appmessage.RPCBlockHeader {
	headerRaw, err := ioutil.ReadFile("example_header.json")
	if err != nil {
		panic(err)
	}
	header := appmessage.RPCBlockHeader{}
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		panic(err)
	}
	return &header
}

func newTestContext(t *testing.T) *testContext {
	ctx, conn := gostratum.NewMockContext(context.Background(), logger, MiningStateGenerator())

	headerRaw, err := ioutil.ReadFile("example_header.json")
	if err != nil {
		t.Fatal(err)
	}
	header := appmessage.RPCBlockHeader{}
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		t.Fatal(err)
	}

	jm := NewJobManager()
	jobIdx := jm.AddJob(&appmessage.RPCBlock{
		Header: &header,
	})

	return &testContext{
		ctx:    ctx,
		conn:   conn,
		block:  &appmessage.RPCBlock{Header: &header},
		jobMan: jm,
		job:    jobIdx,
	}
}

func TestShareLogging(t *testing.T) {
	tc := newTestContext(t)
	sh := newShareHandler(nil, tc.jobMan, rd, pg)

	// Submit a good share, should be recorded and respond w/ no errors
	tc.conn.AsyncReadTestDataFromBuffer(func(b []byte) {
		resp, err := gostratum.UnmarshalResponse(string(b))
		if err != nil {
			t.Fatal(err)
		}
		if resp.Error != nil {
			t.Fatalf("no error expected, got: %s", resp.Error...)
		}
	})
	nonce := time.Now().Unix()
	err := sh.HandleSubmit(tc.ctx, gostratum.NewEvent("1", "mining.submit", []any{
		"", fmt.Sprintf("%d", tc.job.Id), fmt.Sprintf("%d", nonce),
	}))
	if err != nil {
		t.Fatalf("submit failed, should have allowed share submission")
	}

	// Submit the same nonce again, should be a dupe and report back as so
	tc.conn.AsyncReadTestDataFromBuffer(func(b []byte) {
		resp, err := gostratum.UnmarshalResponse(string(b))
		if err != nil {
			t.Fatal(err)
		}
		if resp.Error == nil {
			t.Fatalf("dupe error expected")
		}
	})
	err = sh.HandleSubmit(tc.ctx, gostratum.NewEvent("1", "mining.submit", []any{
		"", fmt.Sprintf("%d", tc.job.Id), fmt.Sprintf("%d", nonce),
	}))
	if err != nil { // allow the submission but return error to the miner
		t.Fatalf("submit failed, should have allowed share submission")
	}
}
