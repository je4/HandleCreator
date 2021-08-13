package JWTInterceptor

import (
	"crypto/sha512"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	hello := func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { fmt.Fprintf(w, "hello\n") })
	}

	mux := http.NewServeMux()
	mux.Handle("/test", JWTInterceptor(hello(), "/test", "secret", []string{"HS256", "HS384", "HS512"}, sha512.New()))
	srv := &http.Server{
		Handler: mux,
		Addr:    ":7788",
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server died: %v", err)
		}
	}()
	defer srv.Close()

	client := &http.Client{}
	resp, err := client.Get("http://localhost:7788/test")
	if err != nil {
		t.Fatalf("webserver not running: %v", err)
	}
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("cannot read result: %v", err)
	}
	resultStr := strings.TrimSpace(string(result))
	if resultStr != "hello" {
		t.Fatalf("result: %s != hello", resultStr)
	}
}
