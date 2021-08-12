package main

import (
	"database/sql"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	lm "github.com/je4/utils/v2/pkg/logger"
	_ "github.com/lib/pq"
	"log"
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

	// todo: ssh tunnel

	// get database connection handle
	db, err := sql.Open(config.DB.ServerType, config.DB.DSN)
	if err != nil {
		logger.Fatalf("error opening database: %v", err)
	}
	// close on shutdown
	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		logger.Fatalf("error pinging database: %v", err)
	}

}
