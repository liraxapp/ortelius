// (c) 2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package services

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/go-redis/redis/v8"
	"github.com/gocraft/health"
	"github.com/gocraft/health/sinks/bugsnag"

	"github.com/ava-labs/ortelius/cfg"
	"github.com/ava-labs/ortelius/services/cache"
	"github.com/ava-labs/ortelius/services/db"
)

type Connections struct {
	stream *health.Stream
	logger logging.Logger

	db    *db.Conn
	redis *redis.Client
	cache *cache.Cache
}

func NewConnectionsFromConfig(servicesConf cfg.Services, healthConf cfg.Health) (*Connections, error) {
	// Always create a stream and log
	stream := NewStream(healthConf)
	log, err := logging.New(servicesConf.Logging)
	if err != nil {
		return nil, err
	}

	// Create db and redis connections if configured
	var (
		dbConn      *db.Conn
		redisClient *redis.Client
	)

	if servicesConf.Redis != nil && servicesConf.Redis.Addr != "" {
		kvs := health.Kvs{"addr": servicesConf.Redis.Addr, "db": strconv.Itoa(servicesConf.Redis.DB)}
		redisClient, err = NewRedisConn(&redis.Options{
			DB:       servicesConf.Redis.DB,
			Addr:     servicesConf.Redis.Addr,
			Password: servicesConf.Redis.Password,
		})
		if err != nil {
			return nil, stream.EventErrKv("connect.redis", err, kvs)
		}
		stream.EventKv("connect.redis", kvs)
	} else {
		stream.Event("connect.redis.skip")
	}

	if servicesConf.DB != nil || servicesConf.DB.Driver == db.DriverNone {
		// Setup logging kvs
		kvs := health.Kvs{"driver": servicesConf.DB.Driver}
		loggableDSN, err := db.SanitizedDSN(servicesConf.DB)
		if err != nil {
			return nil, stream.EventErrKv("connect.db.sanitize_dsn", err, kvs)
		}
		kvs["dsn"] = loggableDSN

		// Create connection
		dbConn, err = db.New(stream, *servicesConf.DB)
		if err != nil {
			return nil, stream.EventErrKv("connect.db", err, kvs)
		}
		stream.EventKv("connect.db", kvs)
	} else {
		stream.Event("connect.db.skip")
	}

	return NewConnections(log, stream, dbConn, redisClient), nil
}

func NewConnections(l logging.Logger, s *health.Stream, db *db.Conn, r *redis.Client) *Connections {
	var c *cache.Cache
	if r != nil {
		c = cache.New(r)
	}

	return &Connections{
		logger: l,
		stream: s,

		db:    db,
		redis: r,
		cache: c,
	}
}

func (c Connections) Stream() *health.Stream { return c.stream }
func (c Connections) Logger() logging.Logger { return c.logger }
func (c Connections) DB() *db.Conn           { return c.db }
func (c Connections) Redis() *redis.Client   { return c.redis }
func (c Connections) Cache() *cache.Cache    { return c.cache }

func (c Connections) Close() error {
	errs := wrappers.Errs{}
	errs.Add(c.db.Close(context.Background()))
	if c.redis != nil {
		errs.Add(c.redis.Close())
	}
	return errs.Err
}

func NewStream(conf cfg.Health) *health.Stream {
	s := health.NewStream()

	switch conf.Writer {
	case "none":
	case "":
	case "stdout":
		s.AddSink(&health.WriterSink{Writer: os.Stdout})
		s.EventKv("sink.added.writer", health.Kvs{"file": "stdout"})
	}

	if conf.Bugsnag != nil {
		s.AddSink(bugsnag.NewSink(&bugsnag.Config{
			APIKey:       conf.Bugsnag.Key,
			ReleaseStage: conf.Bugsnag.Env,
		}))
		s.EventKv("sink.added.bugsnag", health.Kvs{"stage": conf.Bugsnag.Env})
	}

	return s
}

func NewRedisConn(opts *redis.Options) (*redis.Client, error) {
	client := redis.NewClient(opts)

	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	return client, nil
}
