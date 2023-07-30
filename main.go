package main

import (
	"fmt"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/toml"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"hll-rcon-proxy/db"
	"os"
	"sync"
)

var (
	Repository *db.Repository
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	config.AddDriver(toml.Driver)
	err := config.LoadFiles("config.toml")
	if err != nil {
		log.Panic().Err(err)
	}

	Repository = db.NewRepository()

	var proxies []Proxy
	proxyConf := config.SubDataMap("proxies")
	for _, key := range proxyConf.Keys() {
		var proxy Proxy
		config.BindStruct(fmt.Sprintf("proxies.%s", key), &proxy)
		proxy.ServerName = key

		proxies = append(proxies, proxy)
	}

	var wg sync.WaitGroup
	for _, proxy := range proxies {
		wg.Add(1)
		go startProxy(&wg, proxy)
	}

	wg.Wait()
	Repository.Close()
}
