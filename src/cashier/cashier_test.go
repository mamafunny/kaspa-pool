package cashier

import (
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/onemorebsmith/kaspa-pool/src/common"
	"github.com/onemorebsmith/kaspa-pool/src/kaspaapi"
	"github.com/onemorebsmith/kaspa-pool/src/model"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var rd *redis.Client
var logger *zap.Logger

func TestMain(m *testing.M) {
	logger = common.ConfigureZap(zap.DebugLevel)
	postgres.ConfigureDockerConnection()
	rd = redis.NewClient(&redis.Options{
		Addr: ":6379",
		DB:   0, // use default DB
	})
	if err := rd.Ping(context.Background()); err.Err() != nil {
		panic(errors.Wrapf(err.Err(), "FATAL, failed to connect to redis at %s", ":6379"))
	}
	rd.Del(context.Background(), "share_buffer")
	defer rd.Close()

	m.Run()
}

var testEffort = []model.LedgerEntry{
	{Wallet: "A", Amount: 100, Status: model.LedgerEntryStatusOwed, TxId: nil},
	{Wallet: "B", Amount: 500, Status: model.LedgerEntryStatusOwed, TxId: nil},
	{Wallet: "C", Amount: 399, Status: model.LedgerEntryStatusOwed, TxId: nil},
	{Wallet: "D", Amount: 1, Status: model.LedgerEntryStatusOwed, TxId: nil},
}

func TestPayoutCalculation(t *testing.T) {
	postgres.DoExecOrDie(context.Background(), "DELETE from ledger")
	postgres.DoExecOrDie(context.Background(), "DELETE from shares")
	postgres.DoExecOrDie(context.Background(), "DELETE from blocks")
	postgres.DoExecOrDie(context.Background(), blockData)

	blockhash := "9d6e8049dc0c78499b034981d305541d65d39c5c3ba560eca52febebac06caa6"

	sort.Slice(testEffort, func(i, j int) bool { return testEffort[i].Wallet < testEffort[j].Wallet })
	// Generate a buffer of shares for each worker
	totalEffort := uint64(0)
	shares := []string{}
	for _, v := range testEffort {
		totalEffort += v.Amount
		for i := uint64(0); i < v.Amount; i++ {
			shares = append(shares, string(v.Wallet))
		}
	}

	// shuffle the shares in a random, but reproducable way
	rand.Seed(12345678) // reproducable seed
	rand.Shuffle(len(shares), func(i, j int) { shares[i], shares[j] = shares[j], shares[i] })

	// clear any residual data
	if err := postgres.DoExec(context.Background(), "DELETE FROM shares"); err != nil {
		t.Fatalf("failed to clean shares table before test execution: %s", err)
	}

	// put the shares in the db
	i := uint64(0)
	for _, v := range shares {
		i++
		if err := postgres.PutShare(context.Background(), v, 12345, i, 1); err != nil {
			t.Fatal(err)
		}
	}

	expectedEffort := model.EffortMap{"A": 100, "B": 500, "C": 399, "D": 1}
	// fetch the aggregated shares back from pg, and validate it's what we expect
	effort, err := postgres.GetSharesByWallet(context.Background(), time.Now(), totalEffort)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(expectedEffort, effort); d != "" {
		t.Fatalf("calculated incorrect effort: %s", d)
	}

	payout := DeterminePayouts(totalEffort, 0, effort) // payout should be 1 to 1 share to $
	sort.Slice(payout, func(i, j int) bool { return payout[i].Wallet < payout[j].Wallet })
	if d := cmp.Diff(&testEffort, &payout); d != "" {
		t.Fatalf("calculated incorrect payout: %s", d)
	}
	if err := postgres.PutPayable(context.Background(), blockhash, payout); err != nil {
		t.Fatal(err)
	}

	for _, v := range testEffort {
		pending, err := postgres.GetPendingForWallet(context.Background(), v.Wallet)
		if err != nil {
			t.Fatal(err)
		}
		if v.Amount != pending {
			t.Fatalf("incorrect pending balance, expected %d, got %d", v.Amount, pending)
		}
	}
}

var blockData = `INSERT INTO "public"."blocks"("hash","daascore","bluescore","timestamp","miner","payee","round_time","block_json","luck","status","coinbase_reward","pool")
VALUES
(E'9d6e8049dc0c78499b034981d305541d65d39c5c3ba560eca52febebac06caa6',28554656,27031522,E'2022-10-05 21:44:48.110018',E'kaspa:qqkrl0er5ka5snd55gr9rcf6rlpx8nln8gf3jxf83w4dc0khfqmauy6qs83zm',E'kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau',E'05:32:56.927973',E'{"Header": {"Bits": 453082126, "Nonce": 0, "Parents": [{"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["6e5336490f804198b7a4907ba6c18353fee59d8db2068cfb5d744d057b8b3c38"]}, {"ParentHashes": ["f0f8b810dc60866425453cafb6617f311a155c226fe26f9a2a0ade5752a217d3"]}, {"ParentHashes": ["1cec0bc930aad946154c27a6ee187121e363edc84f22b95e456fe7803583ca97"]}, {"ParentHashes": ["1cec0bc930aad946154c27a6ee187121e363edc84f22b95e456fe7803583ca97"]}, {"ParentHashes": ["1cec0bc930aad946154c27a6ee187121e363edc84f22b95e456fe7803583ca97"]}, {"ParentHashes": ["dc92ab2c182c4e975d16d578b37d2364581a654a3232f92f0c077c4583c120dd"]}, {"ParentHashes": ["8aaf0fedbb1c182c6c6bf8b46d99c72dde5ac78a396b16444f5f4c3bc2339974"]}, {"ParentHashes": ["8aaf0fedbb1c182c6c6bf8b46d99c72dde5ac78a396b16444f5f4c3bc2339974"]}, {"ParentHashes": ["d70f6476df038c3644c5a425fdef6e28520b7d3007f0a64472030b578675e5f2"]}, {"ParentHashes": ["d70f6476df038c3644c5a425fdef6e28520b7d3007f0a64472030b578675e5f2"]}, {"ParentHashes": ["d70f6476df038c3644c5a425fdef6e28520b7d3007f0a64472030b578675e5f2"]}, {"ParentHashes": ["d70f6476df038c3644c5a425fdef6e28520b7d3007f0a64472030b578675e5f2"]}, {"ParentHashes": ["98de8dd2696a558c160824611f8351eb0557198b64e8cec60bd22f151786ebc5"]}, {"ParentHashes": ["98de8dd2696a558c160824611f8351eb0557198b64e8cec60bd22f151786ebc5"]}, {"ParentHashes": ["e2bad4c15afcccda57aa89b4cab219af96560d0dddbc8563e17f8f5f0290eb11"]}, {"ParentHashes": ["e2bad4c15afcccda57aa89b4cab219af96560d0dddbc8563e17f8f5f0290eb11"]}, {"ParentHashes": ["36161178fac70818390a3ecb743d041d4edf3e8e5f1d7564984d5de14263ac0f"]}, {"ParentHashes": ["36161178fac70818390a3ecb743d041d4edf3e8e5f1d7564984d5de14263ac0f"]}, {"ParentHashes": ["00eca12dc79701af7b105c98a58a7f58a02eda88730c53ecf604ba7d4b1afdab"]}, {"ParentHashes": ["00eca12dc79701af7b105c98a58a7f58a02eda88730c53ecf604ba7d4b1afdab"]}, {"ParentHashes": ["00eca12dc79701af7b105c98a58a7f58a02eda88730c53ecf604ba7d4b1afdab"]}], "Version": 1, "BlueWork": "258e1385384dfc4240", "DAAScore": 28554656, "BlueScore": 27031522, "Timestamp": 1665006287119, "PruningPoint": "24519ddc72168bae722ee9b42bfa0b9c22b578797c9bf7fad8e552320a1c303d", "HashMerkleRoot": "281798f34b7473e1f736dba3bcfbcf829dcc0baa60ed1663a796a8994c6be5d4", "UTXOCommitment": "3be9607e3f223f58ade69a4f2e4b9d8678fbdad66b0a73636841ec4908746d86", "AcceptedIDMerkleRoot": "c96b416e8238231568dc556ac1cabc7ae84985c3ed7f66899f2ead41ec0f9a74"}, "VerboseData": null, "Transactions": [{"Gas": 0, "Inputs": [], "Outputs": [{"Amount": 34922823143, "VerboseData": null, "ScriptPublicKey": {"Script": "20c62cf30e4e57c5922086460235d5157a62f2666aa22707cb9b588f6211fc0ed6ac", "Version": 0}}], "Payload": "e2779c0100000000e7fd8f210800000000002220e0bf89fc75b0380d30711c916612cdaaf6ab65aed00544ce70e57eb14040b050ac302e31322e372f6f6e656d6f726562736d6974682f6b617370612d706f6f6c5f76302e31", "Version": 0, "LockTime": 0, "VerboseData": null, "SubnetworkID": "0100000000000000000000000000000000000000"}]}',100,E'unconfirmed',NULL,NULL),
(E'f2b36edcfaceaba4a21b24e0ce58c6264b51c577aa5bc8fa4c452f52b5d80d1f',28564862,27041704,E'2022-10-06 00:34:48.100663',E'kaspa:qzk3uh2twkhu0fmuq50mdy3r2yzuwqvstq745hxs7tet25hfd4egcafcdmpdl',E'kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau',E'02:35:01.827759',E'{"Header": {"Bits": 453080112, "Nonce": 0, "Parents": [{"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["3514ef1f84bfb35ef81ad3c682cc28aa9b5226b65a353e7906f5342c8ba66fd4"]}, {"ParentHashes": ["bc2a71cc0299469b2a76e0c3b0b320776326c9397be493d25c3e683146b36511"]}, {"ParentHashes": ["0bd83108544570ab13e218419971e38a44c2b4587fe5f4c92d2d0327668d287b"]}, {"ParentHashes": ["6c584072a7f091bb78f1c192d29c26574970be9de39ed492baa057648e39e1d0"]}, {"ParentHashes": ["6c584072a7f091bb78f1c192d29c26574970be9de39ed492baa057648e39e1d0"]}, {"ParentHashes": ["0ddda15dbeb1dc196d562f9a66c2801fff8603c3f5e9baaebc22c860b6d424b2"]}, {"ParentHashes": ["f720725efc3d1c050abbb1c23389912fede132b324221023270db3922869b180"]}, {"ParentHashes": ["d771b2fec9f28de118cc2feb70e623edf3a616b4d91b04ebc47aa6700e2692b9"]}, {"ParentHashes": ["d771b2fec9f28de118cc2feb70e623edf3a616b4d91b04ebc47aa6700e2692b9"]}, {"ParentHashes": ["6cf2b0a372505c12dd8051dd7cbd5349809e8937db909d9c7464b0163b8d536a"]}, {"ParentHashes": ["6cf2b0a372505c12dd8051dd7cbd5349809e8937db909d9c7464b0163b8d536a"]}, {"ParentHashes": ["6cf2b0a372505c12dd8051dd7cbd5349809e8937db909d9c7464b0163b8d536a"]}, {"ParentHashes": ["6cf2b0a372505c12dd8051dd7cbd5349809e8937db909d9c7464b0163b8d536a"]}, {"ParentHashes": ["6cf2b0a372505c12dd8051dd7cbd5349809e8937db909d9c7464b0163b8d536a"]}, {"ParentHashes": ["6cf2b0a372505c12dd8051dd7cbd5349809e8937db909d9c7464b0163b8d536a"]}, {"ParentHashes": ["e2bad4c15afcccda57aa89b4cab219af96560d0dddbc8563e17f8f5f0290eb11"]}, {"ParentHashes": ["36161178fac70818390a3ecb743d041d4edf3e8e5f1d7564984d5de14263ac0f"]}, {"ParentHashes": ["36161178fac70818390a3ecb743d041d4edf3e8e5f1d7564984d5de14263ac0f"]}, {"ParentHashes": ["00eca12dc79701af7b105c98a58a7f58a02eda88730c53ecf604ba7d4b1afdab"]}, {"ParentHashes": ["00eca12dc79701af7b105c98a58a7f58a02eda88730c53ecf604ba7d4b1afdab"]}, {"ParentHashes": ["00eca12dc79701af7b105c98a58a7f58a02eda88730c53ecf604ba7d4b1afdab"]}], "Version": 1, "BlueWork": "25a97c873eca689054", "DAAScore": 28564862, "BlueScore": 27041704, "Timestamp": 1665016487339, "PruningPoint": "24519ddc72168bae722ee9b42bfa0b9c22b578797c9bf7fad8e552320a1c303d", "HashMerkleRoot": "b306f30af4c9b4f28cae645e813b34da31564b42fd4fbaf6e88d80e8e8f74357", "UTXOCommitment": "f7519aee8d0f1c40d86e58b641b5557bbbf54f85dc5f379de9c4c5ab153840de", "AcceptedIDMerkleRoot": "df3ff59dca4a0a3ccfbef3559e1b43174c7308a90d2b443e7500ab57c902077b"}, "VerboseData": null, "Transactions": [{"Gas": 0, "Inputs": [], "Outputs": [{"Amount": 34922823143, "VerboseData": null, "ScriptPublicKey": {"Script": "20c62cf30e4e57c5922086460235d5157a62f2666aa22707cb9b588f6211fc0ed6ac", "Version": 0}}], "Payload": "a89f9c0100000000e7fd8f210800000000002220e0bf89fc75b0380d30711c916612cdaaf6ab65aed00544ce70e57eb14040b050ac302e31322e372f6f6e656d6f726562736d6974682f6b617370612d706f6f6c5f76302e31", "Version": 0, "LockTime": 0, "VerboseData": null, "SubnetworkID": "0100000000000000000000000000000000000000"}]}',100,E'unconfirmed',NULL,NULL);`
var transactionData = `INSERT INTO "public"."coinbase_payments"("tx","wallet","amount","daascore")
VALUES
(E'2301e103768b658e5b007fca72d227e3f2e29908b464500b4821d4809e423400',E'kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau',34922823143,28564864),
(E'e5022c2f0e1d49471e0eb6091e7431b3ec90a607f9ad47c92c22d88214bdb16c',E'kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau',34922823143,28554661);
`

func TestResolveBlock(t *testing.T) {
	ctx := context.Background()
	logger := common.ConfigureZap(zap.DebugLevel)
	postgres.DoExecOrDie(context.Background(), "DELETE from blocks")
	postgres.DoExecOrDie(context.Background(), "DELETE from coinbase_payments")
	postgres.DoExecOrDie(context.Background(), blockData)
	postgres.DoExecOrDie(context.Background(), transactionData)

	resolved, err := ResolveBlocks(ctx, "kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau", logger)
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 2 {
		t.Fatal("expected 2 blocks to be resolved")
	}
	CommitResolvedBlocks(ctx, logger, resolved)

	blocks, err := postgres.GetConfirmedBlocks(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	mapped := model.ConfirmedBlockArrayToMap(blocks)
	if mapped["9d6e8049dc0c78499b034981d305541d65d39c5c3ba560eca52febebac06caa6"].CoinbasePayment.TxId != "e5022c2f0e1d49471e0eb6091e7431b3ec90a607f9ad47c92c22d88214bdb16c" {
		t.Fatal("incorrectly mapped block")
	}
	if mapped["f2b36edcfaceaba4a21b24e0ce58c6264b51c577aa5bc8fa4c452f52b5d80d1f"].CoinbasePayment.TxId != "2301e103768b658e5b007fca72d227e3f2e29908b464500b4821d4809e423400" {
		t.Fatal("incorrectly mapped block")
	}
}

func TestCoinbase(t *testing.T) {
	ctx := context.Background()

	postgres.DoExecOrDie(context.Background(), "DELETE from blocks")
	postgres.DoExecOrDie(context.Background(), "DELETE from coinbase_payments")
	postgres.DoExecOrDie(context.Background(), transactionData)
	tip, err := postgres.GetCoinbaseTip(ctx, "kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau")
	if err != nil {
		t.Fatal(err)
	}
	if tip != 28564864 {
		t.Fatalf("wrong value for daa tip, got %d, expected 28564864", tip)
	}

}

func TestTransactionListener(t *testing.T) {
	//ctx := context.Background()
	ka, err := kaspaapi.NewKaspaAPI(":16110", logger)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	StartListener(ctx, ka, logger)
	time.Sleep(5 * time.Minute) // obviously don't check this in
}

func TestTransactionBackfill(t *testing.T) {
	//ctx := context.Background()
	ka, err := kaspaapi.NewKaspaAPI(":16110", logger)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = Backfill(ctx, ka, logger, "bbf5f762cb20cfd73f34422f7a003e01c3e19f6ce080396044b5df3af42a9104")
	if err != nil {
		t.Fatal(err)
	}
}
