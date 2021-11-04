package client

import (
	"bytes"
	"crypto/sha512"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/je4/HandleCreator/v2/pkg/server"
	"github.com/je4/utils/v2/pkg/JWTInterceptor"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HandleCreatorClient struct {
	client http.Client
	addr   string
	logger *logging.Logger
}

func NewHandleCreatorClient(service, addr string, jwtKey, jwtAlg string, certSkipVerify bool, logger *logging.Logger) (*HandleCreatorClient, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: certSkipVerify}
	tr, err := JWTInterceptor.NewJWTTransport(service, "Create", JWTInterceptor.Secure, nil, sha512.New(), jwtKey, jwtAlg, 30*time.Second)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create jwt transport")
	}

	hsc := &HandleCreatorClient{
		addr: addr,
		client: http.Client{
			Transport: tr,
		},
		logger: logger,
	}
	return hsc, nil
}

func (hsc *HandleCreatorClient) Ping() error {
	u := fmt.Sprintf("%s/ping", hsc.addr)
	resp, err := hsc.client.Get(u)
	if err != nil {
		hsc.logger.Errorf("cannot query %s: %v", u, err)
		return errors.Wrapf(err, "cannot query %s", u)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("error pinging HandleCreator Service: invalid status %s", resp.Status))
	}
	jdec := json.NewDecoder(resp.Body)
	result := &server.ApiResult{}
	if err := jdec.Decode(result); err != nil {
		hsc.logger.Errorf("cannot read and decode result: %v", err)
		return errors.Wrap(err, "cannot read and decode result")
	}
	if result.Status != "ok" {
		return errors.New(fmt.Sprintf("error pinging HandleCreator Service: %s", result.Message))
	}
	return nil
}

func (hsc *HandleCreatorClient) Create(handle string, URL *url.URL) error {
	createStruct := struct {
		Handle string `json:"handle"`
		Url    string `json:"url"`
	}{
		Handle: handle,
		Url:    URL.String(),
	}
	data, err := json.Marshal(createStruct)
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("cannot marshal create struct %v", createStruct))
	}
	u := fmt.Sprintf("%s/create", hsc.addr)
	hsc.logger.Infof("creating handle %s", handle)
	resp, err := hsc.client.Post(u,
		"application/json",
		bytes.NewBuffer(data))
	if err != nil {
		hsc.logger.Errorf("cannot query %s: %v", u, err)
		return errors.Wrapf(err, "cannot query %s", u)
	}
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		hsc.logger.Errorf("cannot read result: %v", err)
		return errors.Wrap(err, "cannot read result")
	}
	if resp.StatusCode != http.StatusCreated {
		hsc.logger.Errorf("handle %s not created: %s", handle, string(result))
		return errors.New(fmt.Sprintf("handle %s not created: %s", handle, string(result)))
	}
	return nil
}
