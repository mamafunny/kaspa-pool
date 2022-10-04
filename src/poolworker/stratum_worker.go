package poolworker

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/mattn/go-colorable"
	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const version = "v0.1"

func configureZap(cfg WorkerConfig) (*zap.Logger, func()) {
	pe := zap.NewProductionEncoderConfig()
	pe.EncodeTime = zapcore.RFC3339TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(pe)
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	// log file fun
	logFile, err := os.OpenFile("bridge.log", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zap.InfoLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(colorable.NewColorableStdout()), zap.InfoLevel),
	)
	return zap.New(core), func() { logFile.Close() }
}

func configureRedis(cfg RedisConfig) (*redis.Client, error) {
	return nil, nil
}

func ListenAndServe(cfg WorkerConfig) error {
	logger, logCleanup := configureZap(cfg)
	defer logCleanup()

	if cfg.PromPort != "" {
		StartPromServer(logger, cfg.PromPort)
	}

	ksApi, err := NewKaspaAPI(cfg.RPCServer, logger)
	if err != nil {
		return err
	}

	if cfg.HealthCheckPort != "" {
		logger.Info("enabling health check on port " + cfg.HealthCheckPort)
		http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		go http.ListenAndServe(cfg.HealthCheckPort, nil)
	}

	pg, err := configurePostgres(cfg.PostgresConfig)
	if err != nil {
		return errors.Wrap(err, "failed connecting to postgres")
	}
	rd, err := configureRedis(cfg.RedisConfig)
	if err != nil {
		return errors.Wrap(err, "failed connecting to redis")
	}

	jobManager := NewJobManager()
	shareHandler := newShareHandler(ksApi, jobManager, rd, pg)
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
	server.Listen(context.Background())
	return nil
}
