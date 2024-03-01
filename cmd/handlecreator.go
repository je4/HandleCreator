package main

import (
	"context"
	"database/sql"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/je4/HandleCreator/v2/pkg/server"
	"github.com/je4/utils/v2/pkg/zLogger"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var configfile = flag.String("cfg", "/etc/handlecreator.toml", "configuration file")

	flag.Parse()
	config := &Config{}
	if err := LoadConfig(*configfile, config); err != nil {
		log.Printf("cannot load config file: %v", err)
	}

	var out io.Writer = os.Stdout
	if config.Logfile != "" {
		fp, err := os.OpenFile(config.Logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("cannot open logfile %s: %v", config.Logfile, err)
		}
		defer fp.Close()
		out = fp
	}

	output := zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
	_logger := zerolog.New(output).With().Timestamp().Logger()
	_logger.Level(zLogger.LogLevel(config.Loglevel))
	var logger zLogger.ZLogger = &_logger

	var accessLog io.Writer
	var f *os.File
	var err error
	if config.AccessLog == "" {
		accessLog = os.Stdout
	} else {
		f, err = os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			logger.Panic().Msgf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accessLog = f
	}

	// get database connection handle
	db, err := sql.Open(config.DB.ServerType, config.DB.DSN)
	if err != nil {
		logger.Panic().Msgf("error opening database: %v", err)
	}
	// close on shutdown
	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		logger.Panic().Msgf("error pinging database: %v", err)
	}

	srv, err := server.NewServer(config.ServiceName, config.Addr, db, config.DB.Schema, logger, accessLog, config.JWTKey, config.JWTAlg)
	if err != nil {
		logger.Panic().Msgf("error initializing server: %v", err)
	}

	go func() {
		if err := srv.ListenAndServe(config.CertPEM, config.KeyPEM); err != nil {
			log.Fatalf("server died: %v", err)
		}
	}()

	end := make(chan bool, 1)

	// process waiting for interrupt signal (TERM or KILL)
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)

		signal.Notify(sigint, syscall.SIGTERM)
		signal.Notify(sigint, syscall.SIGKILL)

		<-sigint

		// We received an interrupt signal, shut down.
		logger.Info().Msg("shutdown requested")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)

		end <- true
	}()

	<-end
	logger.Info().Msg("server stopped")
}
