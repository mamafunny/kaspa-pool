package kaspastratum

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx"
	"github.com/mattn/go-colorable"
	"github.com/onemorebsmith/kaspa-pool/src/gostratum"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const version = "v1.1"

type BridgeConfig struct {
	StratumPort     string `yaml:"stratum_port"`
	RPCServer       string `yaml:"kaspad_address"`
	PromPort        string `yaml:"prom_port"`
	HealthCheckPort string `yaml:"health_check_port"`
	RedisPort       string `yaml:"redis_port"`
	PostgresPort    string `yaml:"postgres_port"`
}

func configureZap(cfg BridgeConfig) (*zap.SugaredLogger, func()) {
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
	return zap.New(core).Sugar(), func() { logFile.Close() }
}

func ListenAndServe(cfg BridgeConfig) error {
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

	connstring := "postgres://postgres:postgres@localhost:5432/shares"
	pg, err := pgx.Connect(pgx.ConnConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "kaspa-pool",
	})
	if err != nil {
		return errors.Wrapf(err, "FATAL, failed to connect to postgres at %s", connstring)
	}
	defer pg.Close()

	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisPort,
		Password: "eSYzVUxnUzgkb0RLV28meDE0SlVJeDEqd2FwTCVYM05YQVJE",
		DB:       0, // use default DB
	})
	if err := redis.Ping(context.Background()); err.Err() != nil {
		return errors.Wrapf(err.Err(), "FATAL, failed to connect to redis at %s", cfg.RedisPort)
	}

	shareHandler := newShareHandler(ksApi.kaspad, redis, pg)
	clientHandler := newClientListener(logger, shareHandler)
	handlers := gostratum.DefaultHandlers()
	// override the submit handler with an actual useful handler
	handlers[string(gostratum.StratumMethodSubmit)] =
		func(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) error {
			return shareHandler.HandleSubmit(ctx, event)
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
		clientHandler.NewBlockAvailable(ksApi)
	})

	server := gostratum.NewListener(stratumConfig)
	server.Listen(context.Background())
	return nil
}
