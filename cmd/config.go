package main

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"time"
)

type Endpoint struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type SSHTunnel struct {
	User           string   `toml:"user"`
	PrivateKey     string   `toml:"privatekey"`
	LocalEndpoint  Endpoint `toml:"localendpoint"`
	ServerEndpoint Endpoint `toml:"serverendpoint"`
	RemoteEndpoint Endpoint `toml:"remoteendpoint"`
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type DB struct {
	ServerType string
	DSN        string
	ConnMax    int `toml:"connection_max"`
	Schema     string
}

type Config struct {
	Logfile   string               `toml:"logfile"`
	Loglevel  string               `toml:"loglevel"`
	AccessLog string               `toml:"accesslog"`
	Logformat string               `toml:"logformat"`
	Addr      string               `toml:"addr"`
	AddrExt   string               `toml:"addrext"`
	CertPEM   string               `toml:"certpem"`
	KeyPEM    string               `toml:"keypem"`
	JWTKey    string               `toml:"jwtkey"`
	JWTAlg    []string             `toml:"jwtalg"`
	Tunnel    map[string]SSHTunnel `toml:"tunnel"`
	DB        DB                   `toml:"db"`
}

func LoadConfig(fp string, conf *Config) error {
	_, err := toml.DecodeFile(fp, conf)
	if err != nil {
		return errors.Wrapf(err, "error loading config file %v", fp)
	}
	return nil
}