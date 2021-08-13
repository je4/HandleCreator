package JWTInterceptor

import (
	"bytes"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"hash"
	"io"
	"net/http"
	"strings"
)

var allowedContentTypes = []string{
	"text/xml",
	"application/xml",
	"application/json",
}

func checkToken(tokenStr string, jwtKey string, jwtAlg []string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		talg := token.Method.Alg()
		algOK := false
		for _, a := range jwtAlg {
			if talg == a {
				algOK = true
				break
			}
		}
		if !algOK {
			return false, fmt.Errorf("unexpected signing method (allowed are %v): %v", jwtAlg, token.Header["alg"])
		}

		return []byte(jwtKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("Token not valid: %s", tokenStr)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Cannot get claims from token %s", tokenStr)
	}
	return claims, nil
}

func JWTInterceptor(handler http.Handler, ignorePrefix string, jwtKey string, jwtAlg []string, h hash.Hash) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check for allowed content-type
		contentType := r.Header.Get("Content-type")
		var allowed = false
		for _, val := range allowedContentTypes {
			if val == contentType {
				allowed = true
				break
			}
		}
		// read the full body
		body, err := r.GetBody()
		if err != nil {
			http.Error(w, fmt.Sprintf("JWTInterceptor: cannot get body: %v", err), http.StatusInternalServerError)
			return
		}
		bs, err := io.ReadAll(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("JWTInterceptor: cannot read body: %v", err), http.StatusInternalServerError)
			return
		}

		if !allowed && len(bs) > 0 {
			http.Error(w, fmt.Sprintf("JWTInterceptor: invalid content-type for body: %s", contentType), http.StatusInternalServerError)
			return
		}

		// calculate checksum
		h.Reset()
		// checksum from method + rawquery + body
		if _, err := h.Write([]byte(strings.ToUpper(r.Method))); err != nil {
			http.Error(w, fmt.Sprintf("JWTInterceptor: cannot write to checksum: %v", err), http.StatusInternalServerError)
			return
		}
		if _, err := h.Write([]byte(strings.TrimPrefix(r.URL.RawQuery, ignorePrefix))); err != nil {
			http.Error(w, fmt.Sprintf("JWTInterceptor: cannot write to checksum: %v", err), http.StatusInternalServerError)
			return
		}
		if _, err := h.Write(bs); err != nil {
			http.Error(w, fmt.Sprintf("JWTInterceptor: cannot write to checksum: %v", err), http.StatusInternalServerError)
			return
		}
		checksumBytes := h.Sum(nil)
		checksum := fmt.Sprintf("%x", checksumBytes)

		// extract authorization bearer
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")
		if len(splitToken) != 2 {
			http.Error(w, fmt.Sprintf("JWTInterceptor: no Bearer in Authorization Token: %v", reqToken), http.StatusForbidden)
			return
		}
		reqToken = splitToken[1]

		claims, err := checkToken(reqToken, jwtKey, jwtAlg)
		if err != nil {
			http.Error(w, fmt.Sprintf("JWTInterceptor: error in authorization token: %v", err), http.StatusForbidden)
			return
		}

		if checksum != claims["checksum"] {
			http.Error(w, fmt.Sprintf("JWTInterceptor: invalid checksum: %s != %s", checksum, claims["checksum"]), http.StatusForbidden)
			return
		}

		// rewind body
		r.Body.Close()
		buf := bytes.NewBuffer(bs)
		r.Body = io.NopCloser(buf)

		//serve
		handler.ServeHTTP(w, r)
	})
}
