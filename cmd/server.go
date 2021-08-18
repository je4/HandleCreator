package main

import (
	"context"
	"crypto/sha512"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/je4/utils/v2/pkg/JWTInterceptor"
	dcert "github.com/je4/utils/v2/pkg/cert"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"io"
	"net"
	"net/http"
)

type Server struct {
	host, port  string
	srv         *http.Server
	log         *logging.Logger
	accessLog   io.Writer
	jwtKey      string
	jwtAlg      []string
	db          *sql.DB
	replaceStmt *sql.Stmt
	dbSchema    string
}

func NewServer(addr string, db *sql.DB, dbSchema string, log *logging.Logger, accessLog io.Writer, jwtKey string, jwtAlg []string) (*Server, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot split address %s", addr)
	}

	srv := &Server{
		host:      host,
		port:      port,
		db:        db,
		dbSchema:  dbSchema,
		log:       log,
		accessLog: accessLog,
		jwtKey:    jwtKey,
		jwtAlg:    jwtAlg,
	}

	return srv, srv.Init()
}
func (s *Server) Init() error {
	var err error
	sqlstr := fmt.Sprintf("REPLACE INTO %s.handles_handle(handle, data) VALUES(?, ?)", s.dbSchema)
	s.replaceStmt, err = s.db.Prepare(sqlstr)
	if err != nil {
		return errors.Wrapf(err, "cannot create statement - %s", sqlstr)
	}
	return nil
}

func (s *Server) ListenAndServe(cert, key string) (err error) {
	router := mux.NewRouter()
	router.Handle(
		"/create",
		handlers.CompressHandler(JWTInterceptor.JWTInterceptor(
			func() http.Handler { return http.HandlerFunc(s.createHandler) }(),
			"",
			s.jwtKey,
			[]string{"HS256", "HS384", "HS512"},
			sha512.New())),
	).Methods("POST")
	loggedRouter := handlers.CombinedLoggingHandler(s.accessLog, handlers.ProxyHeaders(router))
	addr := net.JoinHostPort(s.host, s.port)
	s.srv = &http.Server{
		Handler: loggedRouter,
		Addr:    addr,
	}

	if cert == "auto" || key == "auto" {
		s.log.Info("generating new certificate")
		cert, err := dcert.DefaultCertificate()
		if err != nil {
			return errors.Wrap(err, "cannot generate default certificate")
		}
		s.srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{*cert}}
		s.log.Infof("starting Handle Creator at https://%v/", addr)
		return s.srv.ListenAndServeTLS("", "")
	} else if cert != "" && key != "" {
		s.log.Infof("starting Handle Creator at https://%v", addr)
		return s.srv.ListenAndServeTLS(cert, key)
	} else {
		s.log.Infof("starting Handle Creator at http://%v", addr)
		return s.srv.ListenAndServe()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.replaceStmt.Close()
	return s.srv.Shutdown(ctx)
}

type createParam struct {
	Handle string `json:"handle"`
	Url    string `json:"url"`
}

func (s *Server) createHandler(w http.ResponseWriter, req *http.Request) {
	jsonDec := json.NewDecoder(req.Body)
	cp := &createParam{}
	if err := jsonDec.Decode(cp); err != nil {
		s.log.Errorf("invalid request body for create: %v", err)
		http.Error(w, fmt.Sprintf("invalid request body for create: %v", err), http.StatusBadRequest)
		return
	}

	s.log.Info("creating handle %s", cp.Handle)
	result, err := s.replaceStmt.Exec(cp.Handle, cp.Url)
	if err != nil {
		s.log.Errorf("cannot execute query: %v", err)
		http.Error(w, fmt.Sprintf("cannot execute query: %v", err), http.StatusBadRequest)
		return
	}
	rows, err := result.RowsAffected()
	if err != nil {
		s.log.Errorf("cannot get affected rows: %v", err)
		http.Error(w, fmt.Sprintf("cannot get affected rows: %v", err), http.StatusBadRequest)
		return
	}
	switch rows {
	case 0:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf("%s written to handle server - no change", cp.Handle)))
	case 1:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf("%s written to handle server", cp.Handle)))
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%s written to handle server - multiple handles - should not happen", cp.Handle)))
	}
}
