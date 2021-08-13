package main

import (
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"io"
	"net"
	"net/http"
)

type Server struct {
	host, port string
	srv        *http.Server
	log        *logging.Logger
	accessLog  io.Writer
	jwtKey     string
	jwtAlg     []string
}

func NewServer(addr string, log *logging.Logger, accessLog io.Writer, jwtKey string, jwtAlg []string) (*Server, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot split address %s", addr)
	}

	srv := &Server{
		host:      host,
		port:      port,
		log:       log,
		accessLog: accessLog,
		jwtKey:    jwtKey,
		jwtAlg:    jwtAlg,
	}

	return srv, nil
}
