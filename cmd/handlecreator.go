package main

import (
	"database/sql"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/je4/sshtunnel/v2/pkg/sshtunnel"
	lm "github.com/je4/utils/v2/pkg/logger"
	_ "github.com/lib/pq"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	var configfile = flag.String("cfg", "/etc/tbbs.toml", "configuration file")

	flag.Parse()
	config := &Config{}
	if err := LoadConfig(*configfile, config); err != nil {
		log.Printf("cannot load config file: %v", err)
	}

	// create logger instance
	logger, lf := lm.CreateLogger("HandleCreator", config.Logfile, nil, config.Loglevel, config.Logformat)
	defer lf.Close()

	var accesslog io.Writer
	if config.AccessLog == "" {
		accesslog = os.Stdout
	} else {
		f, err := os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			logger.Panicf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accesslog = f
	}

	for name, tunnel := range config.Tunnel {
		logger.Infof("starting tunnel %s", name)

		forwards := make(map[string]*sshtunnel.SourceDestination)
		for fwname, fw := range tunnel.Forward {
			forwards[fwname] = &sshtunnel.SourceDestination{
				Local: &sshtunnel.Endpoint{
					Host: fw.Local.Host,
					Port: fw.Local.Port,
				},
				Remote: &sshtunnel.Endpoint{
					Host: fw.Remote.Host,
					Port: fw.Remote.Port,
				},
			}
		}

		t, err := sshtunnel.NewSSHTunnel(
			tunnel.User,
			tunnel.PrivateKey,
			&sshtunnel.Endpoint{
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
			logger.Panicf("cannot create sshtunnel %v - %v", t.String(), err)
		}
		defer t.Close()
	}
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

	_, err = NewServer(config.Addr, logger, accesslog, config.JWTKey, config.JWTAlg)
	if err != nil {
		logger.Panicf("error initializing server: %v", err)
	}

}
