package services

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"riseact/internal/app"
	"riseact/internal/utils/logger"

	"golang.ngrok.com/ngrok"
)

func ProxyApp(port string) error {
	logger.Debug("Starting proxy on port " + port)
	var a *app.Application

	// start ngrok tunnel
	tun, err := app.StartNgrokTunnel()

	if err != nil {
		return err
	}

	// initialize app
	if a == nil {
		a, err = initApp(tun.URL())

		if err != nil {
			logger.Debugf("Error initializing app: %s", err.Error())
			return err
		}
	}

	// print infos
	logger.Info("")
	logger.Infof("App url: %s", tun.URL())
	logger.Info("")

	// start reverse proxy server
	launch("http://localhost:"+port, tun)

	return nil
}

func launch(destUrl string, tun ngrok.Tunnel) error {
	targetURL, _ := url.Parse(destUrl)

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	http.Serve(tun, handler)

	return nil
}
