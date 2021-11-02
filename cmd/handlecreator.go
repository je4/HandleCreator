package main

import (
	"context"
	"database/sql"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/je4/HandleCreator/v2/pkg/server"
	lm "github.com/je4/utils/v2/pkg/logger"
	"github.com/je4/utils/v2/pkg/ssh"
	_ "github.com/lib/pq"
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

	// create logger instance
	logger, lf := lm.CreateLogger("HandleCreator", config.Logfile, nil, config.Loglevel, config.Logformat)
	defer lf.Close()

	var accessLog io.Writer
	var f *os.File
	var err error
	if config.AccessLog == "" {
		accessLog = os.Stdout
	} else {
		f, err = os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			logger.Panicf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accessLog = f
	}

	var tunnels []*ssh.SSHtunnel
	for name, tunnel := range config.Tunnel {
		logger.Infof("starting tunnel %s", name)

		forwards := make(map[string]*ssh.SourceDestination)
		for fwname, fw := range tunnel.Forward {
			forwards[fwname] = &ssh.SourceDestination{
				Local: &ssh.Endpoint{
					Host: fw.Local.Host,
					Port: fw.Local.Port,
				},
				Remote: &ssh.Endpoint{
					Host: fw.Remote.Host,
					Port: fw.Remote.Port,
				},
			}
		}

		t, err := ssh.NewSSHTunnel(
			tunnel.User,
			tunnel.PrivateKey,
			&ssh.Endpoint{
				Host: tunnel.Endpoint.Host,
				Port: tunnel.Endpoint.Port,
			},
			forwards,
			logger,
		)
		if err != nil {
			logger.Panicf("cannot create tunnel %v@%v:%v - %v", tunnel.User, tunnel.Endpoint.Host, tunnel.Endpoint.Port, err)
		}
		if err := t.Start(); err != nil {
			logger.Panicf("cannot create ssh tunnel %v - %v", t.String(), err)
		}
		tunnels = append(tunnels, t)
	}
	defer func() {
		for _, t := range tunnels {
			t.Close()
		}
	}()
	// if tunnels are made, wait until connection is established
	if len(config.Tunnel) > 0 {
		time.Sleep(2 * time.Second)
	}

	// get database connection handle
	db, err := sql.Open(config.DB.ServerType, config.DB.DSN)
	if err != nil {
		logger.Panicf("error opening database: %v", err)
	}
	// close on shutdown
	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		logger.Panicf("error pinging database: %v", err)
	}

	srv, err := server.NewServer(config.Addr, db, config.DB.Schema, logger, accessLog, config.JWTKey, config.JWTAlg)
	if err != nil {
		logger.Panicf("error initializing server: %v", err)
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
		logger.Infof("shutdown requested")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)

		end <- true
	}()

	<-end
	logger.Info("server stopped")
}
