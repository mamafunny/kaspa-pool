package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/onemorebsmith/kaspa-pool/src/cashier"
	"github.com/onemorebsmith/kaspa-pool/src/common"
	"github.com/onemorebsmith/kaspa-pool/src/kaspaapi"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func main() {
	pwd, _ := os.Getwd()
	fullPath := path.Join(pwd, "config.yaml")
	log.Printf("loading config @ `%s`", fullPath)
	rawCfg, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Printf("config file not found: %s", err)
		os.Exit(1)
	}
	cfg := cashier.CashierConfig{}
	if err := yaml.Unmarshal(rawCfg, &cfg); err != nil {
		log.Printf("failed parsing config file: %s", err)
		os.Exit(1)
	}

	flag.StringVar(&cfg.RPCServer, "kaspa", cfg.RPCServer, "address of the kaspad node, default `localhost:16110`")
	flag.StringVar(&cfg.PromPort, "prom", cfg.PromPort, "address to serve prom stats, default `:2112`")
	flag.StringVar(&cfg.HealthCheckPort, "hcp", cfg.HealthCheckPort, `(rarely used) if defined will expose a health check on /readyz, default ""`)
	flag.StringVar(&cfg.PoolWallet, "wallet", cfg.PoolWallet, `pool wallet to use for all block payouts"`)
	flag.StringVar(&cfg.PostgresConfig, "pg", cfg.PostgresConfig, `config string for the postgres connection"`)
	flag.Uint64Var(&cfg.PPLNSWindow, "pplns", cfg.PPLNSWindow, `number of shares in the pplns window"`)

	flag.Parse()

	log.Println("----------------------------------")
	log.Printf("initializing cashier")
	log.Printf("\tkaspad:        %s", cfg.RPCServer)
	log.Printf("\tkaspawallet:   %s", cfg.DaemonAddress)
	log.Printf("\tprom:          %s", cfg.PromPort)
	log.Printf("\thealth check:  %s", cfg.HealthCheckPort)
	log.Printf("\twallet:  		 %s", cfg.PoolWallet)
	log.Printf("\tmock:  		 %t", cfg.Mock)
	log.Printf("\tpplns_window   %d", cfg.PPLNSWindow)
	log.Println("----------------------------------")

	postgres.ConfigurePostgres(cfg.PostgresConfig)

	logger := common.ConfigureZap(zap.InfoLevel)
	kapi, err := kaspaapi.NewKaspaAPI(cfg.RPCServer, logger)
	if err != nil {
		panic(err)
	}
	// kwallet := kaspaapi.NewKaspawalletApi(cfg.DaemonAddress, logger)
	// _, err = ka.SendKas(context.Background(),
	// 	"kaspa:qqkrl0er5ka5snd55gr9rcf6rlpx8nln8gf3jxf83w4dc0khfqmauy6qs83zm",
	// 	"kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau",
	// 	1, "")
	if err != nil {
		panic(err)
	}

	go beginReadyzHandler(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cashier.StartListener(ctx, kapi, logger)
	go cashier.StartPruner(ctx, 5*time.Minute, logger)
	cashier.StartPipeline(ctx, cfg, logger)
}

func beginReadyzHandler(cfg cashier.CashierConfig) {
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		pg, err := postgres.GetConnection(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errors.Wrap(err, "failed pinging postgres").Error()))
			return
		}
		defer pg.Close(r.Context())
		if err := pg.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errors.Wrap(err, "failed pinging postgres").Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(cfg.HealthCheckPort, nil)
}
