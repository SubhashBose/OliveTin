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

	AuthFunc := func (w http.ResponseWriter, r *http.Request) bool {
		u, p, ok := r.BasicAuth()
		if !(cfg.AuthUser == "" && cfg.AuthPass == "") && !(ok && u == cfg.AuthUser && p ==cfg.AuthPass ){
				w.Header().Set("WWW-Authenticate", "Basic realm=\"Control Server\", charset=\"UTF-8\"")
				w.WriteHeader(401)
				return false
		}
		return true
	}
	
	mux.HandleFunc(cfg.ProxyBaseURL+"api/", func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("api req: %q", r.URL)
		if (!AuthFunc(w,r)) {
			return
		}
		r.URL.Path = strings.Replace(r.URL.Path,cfg.ProxyBaseURL,"/",1)
		apiProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc(cfg.ProxyBaseURL+"websocket", func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("websocket req: %q", r.URL)
		if (!AuthFunc(w,r)) {
			return
		}
		r.URL.Path = strings.Replace(r.URL.Path,cfg.ProxyBaseURL,"/",1)
		websocket.HandleWebsocket(w, r)
	})

	mux.HandleFunc(cfg.ProxyBaseURL, func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("ui req: %q", r.URL)
		if (!AuthFunc(w,r)) {
			return
		}
		r.URL.Path = strings.Replace(r.URL.Path,cfg.ProxyBaseURL,"/",1)
		webuiProxy.ServeHTTP(w, r)
	})

	MakeProxy := func (proxy config.ExternalProxy) {
		log.Info("Setting up external Proxy: " + proxy.BaseURL + " -> " + proxy.Target)
		appURL, _ := url.Parse(proxy.Target)
		appProxy := httputil.NewSingleHostReverseProxy(appURL)
		mux.HandleFunc(proxy.BaseURL, func(w http.ResponseWriter, r *http.Request) {
										if (!proxy.NoAuth && !AuthFunc(w,r)) {
												return
										}
										r.URL.Path = strings.Replace(r.URL.Path,proxy.BaseURL,"/",1)
										appProxy.ServeHTTP(w, r)
		})
	}

	for _, extproxy := range cfg.ExternalProxies {
			MakeProxy(extproxy)
	}

	srv := &http.Server{
		Addr:    cfg.ListenAddressSingleHTTPFrontend,
		Handler: mux,
	}

	if (cfg.SSLCertFile != "" && cfg.SSLKeyFile != ""){
		log.Info("Using SSL to serve")
		log.Fatal(srv.ListenAndServeTLS(cfg.SSLCertFile, cfg.SSLKeyFile))
	} else {
		log.Fatal(srv.ListenAndServe())
	}
}
