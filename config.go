package main

import (
	"flag"
	"os"
	"strings"

	"github.com/kelseyhightower/confd/backends"
	"github.com/kelseyhightower/confd/log"
)

const Version = "0.0.1"

type BackendsConfig = backends.Config

// A Config structure is used to configure confd.
type Config struct {
	BackendsConfig
	LogLevel     string
	PrintVersion bool
}

var config Config

func init() {
	flag.StringVar(&config.AuthToken, "auth-token", "", "Auth bearer token to use")
	flag.BoolVar(&config.BasicAuth, "basic-auth", false, "Use Basic Auth to authenticate (only used with -backend=consul and -backend=etcd)")
	flag.StringVar(&config.ClientCaKeys, "client-ca-keys", "", "client ca keys")
	flag.StringVar(&config.ClientCert, "client-cert", "", "the client cert")
	flag.StringVar(&config.ClientKey, "client-key", "", "the client key")
	flag.StringVar(&config.LogLevel, "log-level", "", "level which confd should log messages")
	flag.Var(&config.BackendNodes, "node", "list of backend nodes")
	flag.BoolVar(&config.PrintVersion, "version", false, "print version and exit")
	flag.StringVar(&config.Scheme, "scheme", "http", "the backend URI scheme for nodes retrieved from DNS SRV records (http or https)")
	flag.StringVar(&config.Separator, "separator", "", "the separator to replace '/' with when looking up keys in the backend, prefixed '/' will also be removed (only used with -backend=redis)")
	flag.StringVar(&config.Username, "username", "", "the username to authenticate as (only used with vault and etcd backends)")
	flag.StringVar(&config.Password, "password", "", "the password to authenticate with (only used with vault and etcd backends)")
}

// initConfig initializes the protoagent configuration by first setting defaults,
// then settings from environment variables, and finally overriding
// settings from flags set on the command line.
// It returns an error if any.
func initConfig() error {

	// Update config from environment variables.
	processEnv()

	if config.LogLevel != "" {
		log.SetLevel(config.LogLevel)
	}
	// Update BackendNodes from SRV records.
	if config.Backend != "etcdv3" {
		log.Info("Set backend to 'etcdv3'. This is an Etcdv3 only version.")
		config.Backend = "etcdv3"
	}
	if len(config.BackendNodes) == 0 {
		config.BackendNodes = []string{"127.0.0.1:2379"}
	}

	return nil
}

func processEnv() {
	user := os.Getenv("PROTOAGENT_ETCD_USER")
	if len(user) > 0 {
		userSlice := strings.Split(user, ":")
		config.Username = userSlice[0]
		if len(userSlice) >= 2 {
			config.Password = userSlice[1]
		}
	}

	endpoints := os.Getenv("PROTOAGENT_ETCD_ENDPOINTS")
	if len(endpoints) > 0 {
		config.BackendNodes = strings.Split(endpoints, ",")
	}
}
