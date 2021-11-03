package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/je4/HandleCreator/v2/pkg/client"
	"github.com/je4/HandleCreator/v2/pkg/server"
	lm "github.com/je4/utils/v2/pkg/logger"
	"github.com/phayes/freeport"
	"net/url"
	"os"
	"testing"
	"time"
)

var hcClient *client.HandleCreatorClient
var replStmt *sqlmock.ExpectedPrepare
var mock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	logger, lf := lm.CreateLogger(
		"HandleCreator",
		"",
		nil,
		"DEBUG",
		"%{time:2006-01-02T15:04:05.000} %{module}::%{shortfunc} [%{shortfile}] > %{level:.5s} - %{message}",
	)
	defer lf.Close()

	// get database connection handle
	var db *sql.DB
	var err error
	db, mock, err = sqlmock.New()
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

	mock.ExpectPrepare("REPLACE INTO test.handles_handle\\(handle, data\\)").
		ExpectExec().
		WithArgs("0000.test", "https://test.org").WillReturnResult(sqlmock.NewResult(1, 1))

	port, err := freeport.GetFreePort()
	if err != nil {
		os.Exit(1)
	}

	addr := fmt.Sprintf("localhost:%v", port)
	srv, err := server.NewServer("test", addr, db, "test", logger, os.Stdout, "swordfish", []string{"HS512"})
	if err != nil {
		logger.Panicf("error initializing server: %v", err)
	}
	time.Sleep(2 * time.Second)

	go func() {
		if err := srv.ListenAndServe("", ""); err != nil {
			logger.Fatalf("server died: %v", err)
		}
	}()
	defer srv.Shutdown(context.Background())

	hcClient, err = client.NewHandleCreatorClient("test", fmt.Sprintf("http://%s", addr), "swordfish", "HS512", false, logger)
	if err != nil {
		logger.Fatalf("cannot create handlecreatorclient: %v", err)
	}

	os.Exit(m.Run())
}

func TestPing(t *testing.T) {
	if hcClient == nil {
		t.Error("hcClient is nil")
		return
	}
	if err := hcClient.Ping(); err != nil {
		t.Errorf("ping error: %v", err)
	}
}

func TestCreate(t *testing.T) {
	if hcClient == nil {
		t.Error("hcClient is nil")
		return
	}

	url, _ := url.Parse("https://test.org")
	hcClient.Create("0000.test", url)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
