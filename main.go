package main

import (
	"flag"
	log "github.com/inconshreveable/log15"
	"github.com/tomochain/proxy/cache"
	"github.com/tomochain/proxy/cache/lru"
	"github.com/tomochain/proxy/config"
	"github.com/tomochain/proxy/healthcheck"
	"net/http"
	"net/url"
	"time"
)

var (
	NWorkers        = flag.Int("n", 16, "The number of workers to start")
	HTTPAddr        = flag.String("http", "0.0.0.0:3000", "Address to listen for HTTP requests on")
	ConfigFile      = flag.String("config", "./config/default.json", "Path to config file")
	CacheLimit      = flag.Int("cacheLimit", 100000, "Cache limit")
	CacheExpiration = flag.String("cacheExpiration", "2s", "Cache expiration")
)

var storage cache.Storage

func main() {
	// Parse the command-line flags.
	flag.Parse()

	// setup config
	config.Init(*ConfigFile)
	c := config.GetConfig()
	var urls []*url.URL
	for i := 0; i < len(c.Masternode); i++ {
		url, _ := url.Parse(c.Masternode[i])
		urls = append(urls, url)
	}
	backend.Masternode = urls

	urls = []*url.URL{}
	for i := 0; i < len(c.Fullnode); i++ {
		url, _ := url.Parse(c.Fullnode[i])
		urls = append(urls, url)
	}
	backend.Fullnode = urls

	// setup log
	log.Debug("Starting the proxy", "workers", *NWorkers, "config", *ConfigFile)

	// Cache
	storage, _ = lrucache.NewStorage(*CacheLimit)

	// Healthcheck
    go func() {
        for {
            <-time.After(2 * time.Second)
            for i := 0; i < len(c.Fullnode); i++ {
                url, _ := url.Parse(c.Fullnode[i])
                go healthcheck.Run(url)
            }
            for i := 0; i < len(c.Masternode); i++ {
                url, _ := url.Parse(c.Masternode[i])
                go healthcheck.Run(url)
            }
        }
    }()

	StartDispatcher(*NWorkers)

	http.HandleFunc("/proxystatus", proxystatus)
	http.HandleFunc("/endpointstatus", healthcheck.GetEndpointStatus)

	http.HandleFunc("/", Collector)

	log.Info("HTTP server listening on", "addr", *HTTPAddr)
	if err := http.ListenAndServe(*HTTPAddr, nil); err != nil {
		log.Error("Failed start http server", "error", err.Error())
	}
}
