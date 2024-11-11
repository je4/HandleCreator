package server

import (
	"context"
	"crypto/sha512"
	"crypto/tls"
	"database/sql"
	"emperror.dev/errors"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/je4/utils/v2/pkg/JWTInterceptor"
	dcert "github.com/je4/utils/v2/pkg/cert"
	"github.com/je4/utils/v2/pkg/zLogger"
	"io"
	"net"
	"net/http"
)

type Server struct {
	service     string
	host, port  string
	srv         *http.Server
	log         zLogger.ZLogger
	accessLog   io.Writer
	jwtKey      string
	jwtAlg      []string
	db          *sql.DB
	replaceStmt *sql.Stmt
	dbSchema    string
	adminBearer string
}

type ApiResult struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func NewServer(service, addr string, db *sql.DB, dbSchema string, log zLogger.ZLogger, accessLog io.Writer, jwtKey string, jwtAlg []string, adminBearer string) (*Server, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot split address %s", addr)
	}

	srv := &Server{
		service:     service,
		host:        host,
		port:        port,
		db:          db,
		dbSchema:    dbSchema,
		log:         log,
		accessLog:   accessLog,
		jwtKey:      jwtKey,
		jwtAlg:      jwtAlg,
		adminBearer: adminBearer,
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
	router.HandleFunc("/ping", s.pingHandler).Methods("GET")
	router.Handle(
		"/create",
		handlers.CompressHandler(JWTInterceptor.JWTInterceptor(
			s.service,
			"Create",
			JWTInterceptor.Secure,
			func() http.Handler { return http.HandlerFunc(s.createHandler) }(),
			s.jwtKey,
			[]string{"HS256", "HS384", "HS512"},
			sha512.New(),
			s.adminBearer,
			s.log,
		)),
	).Methods("POST")
	loggedRouter := handlers.CombinedLoggingHandler(s.accessLog, handlers.ProxyHeaders(router))
	addr := net.JoinHostPort(s.host, s.port)
	s.srv = &http.Server{
		Handler: loggedRouter,
		Addr:    addr,
	}

	if cert == "auto" || key == "auto" {
		s.log.Info().Msg("generating new certificate")
		cert, err := dcert.DefaultCertificate()
		if err != nil {
			return errors.Wrap(err, "cannot generate default certificate")
		}
		s.srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{*cert}}
		s.log.Info().Msgf("starting Handle Creator at https://%v/", addr)
		return s.srv.ListenAndServeTLS("", "")
	} else if cert != "" && key != "" {
		s.log.Info().Msgf("starting Handle Creator at https://%v", addr)
		return s.srv.ListenAndServeTLS(cert, key)
	} else {
		s.log.Info().Msgf("starting Handle Creator at http://%v", addr)
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

func (s *Server) pingHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	jenc := json.NewEncoder(w)
	if err := jenc.Encode(&ApiResult{
		Status:  "ok",
		Message: "pong",
		Result:  nil,
	}); err != nil {
		w.Write([]byte(fmt.Sprintf("error encoding json: %v", err)))
	}
}

func (s *Server) createHandler(w http.ResponseWriter, req *http.Request) {
	jsonDec := json.NewDecoder(req.Body)
	cp := &createParam{}
	if err := jsonDec.Decode(cp); err != nil {
		s.log.Error().Msgf("invalid request body for create: %v", err)
		http.Error(w, fmt.Sprintf("invalid request body for create: %v", err), http.StatusBadRequest)
		return
	}

	s.log.Info().Msgf("creating handle %s", cp.Handle)
	result, err := s.replaceStmt.Exec(cp.Handle, cp.Url)
	if err != nil {
		s.log.Error().Msgf("cannot execute query: %v", err)
		http.Error(w, fmt.Sprintf("cannot execute query: %v", err), http.StatusBadRequest)
		return
	}
	rows, err := result.RowsAffected()
	if err != nil {
		s.log.Error().Msgf("cannot get affected rows: %v", err)
		http.Error(w, fmt.Sprintf("cannot get affected rows: %v", err), http.StatusBadRequest)
		return
	}
	jenc := json.NewEncoder(w)
	switch rows {
	case 0:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		if err := jenc.Encode(&ApiResult{
			Status:  "ok",
			Message: fmt.Sprintf("%s written to handle server - no change", cp.Handle),
			Result:  nil,
		}); err != nil {
			http.Error(w, fmt.Sprintf("cannot encode result: %v", err), http.StatusInternalServerError)
		}
	case 1:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		if err := jenc.Encode(&ApiResult{
			Status:  "ok",
			Message: fmt.Sprintf("%s written to handle server", cp.Handle),
			Result:  nil,
		}); err != nil {
			http.Error(w, fmt.Sprintf("cannot encode result: %v", err), http.StatusInternalServerError)
		}
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		if err := jenc.Encode(&ApiResult{
			Status:  "error",
			Message: fmt.Sprintf("%s written to handle server - multiple handles - should not happen", cp.Handle),
			Result:  nil,
		}); err != nil {
			http.Error(w, fmt.Sprintf("cannot encode result: %v", err), http.StatusInternalServerError)
		}
	}
}
