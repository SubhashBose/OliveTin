package httpservers

/*
This file implements a very simple, lightweight reverse proxy so that REST and
the webui can be accessed from a single endpoint.

This makes external reverse proxies (treafik, haproxy, etc) easier, CORS goes
away, and several other issues.
*/

import (
	config "github.com/OliveTin/OliveTin/internal/config"
	"github.com/OliveTin/OliveTin/internal/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// StartSingleHTTPFrontend will create a reverse proxy that proxies the API
// and webui internally.
func StartSingleHTTPFrontend(cfg *config.Config) {
	log.WithFields(log.Fields{
		"address": cfg.ListenAddressSingleHTTPFrontend+cfg.ProxyBaseURL,
	}).Info("Starting single HTTP frontend")

	apiURL, _ := url.Parse("http://" + cfg.ListenAddressRestActions)
	apiProxy := httputil.NewSingleHostReverseProxy(apiURL)

	webuiURL, _ := url.Parse("http://" + cfg.ListenAddressWebUI)
	webuiProxy := httputil.NewSingleHostReverseProxy(webuiURL)

	mux := http.NewServeMux()

	mux.HandleFunc(cfg.ProxyBaseURL+"api/", func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("api req: %q", r.URL)
		r.URL.Path = strings.Replace(r.URL.Path,cfg.ProxyBaseURL,"/",1)
		apiProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc(cfg.ProxyBaseURL, func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.Replace(r.URL.Path,cfg.ProxyBaseURL,"/",1)
		if strings.Contains(r.Header.Get("Connection"), "Upgrade") {
			websocket.HandleWebsocket(w, r)
		} else {
			log.Debugf("ui req: %q", r.URL)
			webuiProxy.ServeHTTP(w, r)
		}
	})

	MakeProxy := func (base, target string) {
		appURL, _ := url.Parse(target)
		appProxy := httputil.NewSingleHostReverseProxy(appURL)
		mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
						r.URL.Path = strings.Replace(r.URL.Path,base,"/",1)
						appProxy.ServeHTTP(w, r)
		})
	}

	for _, extproxy := range cfg.ExternalProxies {
		log.Info("Setting up external Proxy: " + extproxy.BaseURL + " -> " + extproxy.Target)
		MakeProxy(extproxy.BaseURL, extproxy.Target)
	}

	srv := &http.Server{
		Addr:    cfg.ListenAddressSingleHTTPFrontend,
		Handler: mux,
	}

	log.Fatal(srv.ListenAndServe())
}
