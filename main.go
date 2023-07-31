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
	"time"
)

var (
	Repository *db.Repository
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05.000"})
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("TRACE") != "" {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if os.Getenv("DEBUG") != "" {
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
		go startProxy(&wg, &proxy)
	}

	wg.Wait()
	Repository.Close()
}
