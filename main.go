package main

import (
	"context"
	"flag"
	"fmt"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"users/base"
	"users/db"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/prometheus"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/gorilla/handlers"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	var (
		serviceName   = flag.String("service.name", "users", "Name of microservice")
		basePath      = flag.String("service.base.path", "users", "Name of microservice")
		version       = flag.String("service.version", "v1", "Version of microservice")
		httpAddr      = flag.String("http.addr", "localhost", "This is the addr at which http requests are accepted (Default localhost)")
		httpPort      = flag.Int("http.port", 8081, "This is the port at which http requests are accepted (Default :8080)")
		metricsPort   = flag.Int("metrics.port", 8083, "HTTP metrics listen address (Default 8082)")
		dataType      = flag.String("service.datatype", "test", "default Test/qa")
		consulAddr    = flag.String("consul.addr", "localhost:8500", "consul address (Default localhost:8500)")
		serverTimeout = flag.Int64("service.timeout", 2000, "service timeout in milliseconds")
		sysLogAddress = flag.String("syslog.address", "localhost:514", "default location for the syslogger")
		dbusername    = flag.String("db.username", "admin", "couch db username")
		dbpwd         = flag.String("db.pwd", "limited", "couch db password")
		dbType        = flag.String("db.type", "couch", "persistance type")
		queryLimit    = flag.Int("db.limit", 10000, "max db entries to pull on query")
	)
	flag.Parse()
	errs := make(chan error)

	sysLogger, err := syslog.Dial("udp", *sysLogAddress, syslog.LOG_EMERG|syslog.LOG_LOCAL6, *serviceName)
	if err != nil {
		fmt.Printf("exit: %v\n", err)
		return
	}
	defer sysLogger.Close()

	var logger log.Logger
	{
		logger = log.NewJSONLogger(sysLogger)
		logger = log.With(logger, "serviceName", *serviceName)
		logger = log.With(logger, "ip", *httpAddr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	consulClient, registrar, err := base.Register(*serviceName, *consulAddr, *httpAddr, *httpPort, []string{}, logger)
	if err != nil || registrar == nil {
		logger.Log("exit", err)
		return
	}
	registrar.Register()

	dbRegisterName := "db-" + *basePath + "-" + *dataType
	dbInstancer := consulsd.NewInstancer(consulsd.NewClient(consulClient), logger, dbRegisterName, []string{}, true)
	dbs := db.NewRoundRobin(dbInstancer, *dbusername, *dbpwd, *basePath, *dbType, *queryLimit)

	var s base.Service
	{
		labelNames := []string{"method"}
		constLabels := map[string]string{"serviceName": *serviceName, "version": *version, "dataType": *dataType}
		s = base.NewService(logger, dbs)
		s = base.NewLoggingMiddleware(logger)(s)
		s = base.NewInstrumentingService(labelNames, prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Name:        "request_count",
			Help:        "Number of requests received.",
			ConstLabels: constLabels,
		}, labelNames),
			prometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Name:        "err_count",
				Help:        "Number of errors.",
				ConstLabels: constLabels,
			}, labelNames),
			prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
				Name:        "request_latency_seconds",
				Help:        "Total duration of requests in request_latency_seconds.",
				ConstLabels: constLabels,
			}, labelNames),
			s)
	}

	h := base.MakeHTTPHandler(s, logger, *version, *basePath)
	h = http.TimeoutHandler(h, time.Duration(*serverTimeout)*time.Millisecond, "")

	httpServer := http.Server{
		Addr:    ":" + strconv.Itoa(*httpPort),
		Handler: handlers.RecoveryHandler(handlers.RecoveryLogger(base.NewPanicLogger(logger)))(h),
	}
	logger.Log("httpport", "HTTP", "addr", *httpPort)
	go func() {
		errs <- httpServer.ListenAndServe()
	}()

	metricsServer := http.Server{
		Addr:    ":" + strconv.Itoa(*metricsPort),
		Handler: promhttp.Handler(),
	}

	go func() {
		logger.Log("transport", "HTTP", "addr", *metricsPort)
		errs <- metricsServer.ListenAndServe()
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	errMain := <-errs
	//exit gracefully
	errMetricsServer := metricsServer.Shutdown(context.Background())
	errHTTPServer := httpServer.Shutdown(context.Background())
	logger.Log("exit", errMain, "httpErr", errHTTPServer, "metricsErr", errMetricsServer)

}
