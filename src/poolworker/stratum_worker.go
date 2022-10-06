package poolworker

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/go-redis/redis/v8"
	"github.com/onemorebsmith/kaspa-pool/src/common"
	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
	"github.com/onemorebsmith/kaspa-pool/src/postgres"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func configureRedis(path string) (*redis.Client, error) {
	rd := redis.NewClient(&redis.Options{
		Addr: path,
		DB:   0, // use default DB
	})
	if err := rd.Ping(context.Background()); err.Err() != nil {
		return nil, errors.Wrap(err.Err(), "failed to ping redis")
	}

	return rd, nil
}

func ListenAndServe(cfg WorkerConfig) error {
	logger := common.ConfigureZap(zap.InfoLevel)
	if cfg.PromPort != "" {
		StartPromServer(logger, cfg.PromPort)
	}
	ksApi, err := NewKaspaAPI(cfg.RPCServer, logger)
	if err != nil {
		return err
	}
	postgres.ConfigurePostgres(cfg.PostgresConfig)
	rd, err := configureRedis(cfg.RedisConfig)
	if err != nil {
		return errors.Wrap(err, "failed connecting to redis")
	}

	if cfg.HealthCheckPort != "" {
		logger.Info("enabling health check on port " + cfg.HealthCheckPort)
		beginReadyzHandler(cfg, rd)
	}

	jobManager := NewJobManager()
	shareHandler := newShareHandler(ksApi, jobManager, rd)
	clientHandler := newClientListener(logger, shareHandler)
	handlers := gostratum.DefaultHandlers()
	// override the submit handler with an actual useful handler
	handlers[string(gostratum.StratumMethodSubmit)] =
		func(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) error {
			if err := shareHandler.HandleSubmit(ctx, event); err != nil {
				ctx.Logger.Error("error during submit", zap.Error(err)) // sink error
			}
			return nil
		}

	stratumConfig := gostratum.StratumListenerConfig{
		Port:           cfg.StratumPort,
		HandlerMap:     handlers,
		StateGenerator: MiningStateGenerator,
		ClientListener: clientHandler,
		Logger:         logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ksApi.Start(ctx, func() {
		template, err := ksApi.GetBlockTemplate(cfg.PoolWallet)
		if err != nil {
			logger.Error("failed fetching block template", zap.Error(err))
			return
		}
		job := jobManager.AddJob(template.Block)
		clientHandler.NewJobAvailable(job)
	})

	server := gostratum.NewListener(stratumConfig)

	return server.Listen(context.Background())
}

func beginReadyzHandler(cfg WorkerConfig, rd *redis.Client) {
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
		if err := rd.Ping(r.Context()); err.Err() != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errors.Wrap(err.Err(), "failed pinging redis").Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(cfg.HealthCheckPort, nil)
}
