package config

import "fmt"

type Environment string

const (
	Live Environment = "live"
	Beta             = "beta"
	Dev              = "dev"
)

type PonixConfig struct {
	Env      Environment
	Addr     string
	BaseUrl  string
	Firebird FirebirdConfig
}

type FirebirdConfig struct {
	User     string
	Password string
	Hostname string
	Port     int
	DbName   string
}

func (info FirebirdConfig) DSN() string {
	return fmt.Sprintf("%s:%s@%s:%d%s", info.User, info.Password, info.Hostname, info.Port, info.DbName)
}
