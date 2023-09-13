package app

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"golang.ngrok.com/ngrok"
	"golang.org/x/net/websocket"
)

type ReverseProxy struct {
	TargetURL    string
	WebsocketURL string
	tunnel       *ngrok.Tunnel
}

func NewReverseProxy(tunnel *ngrok.Tunnel, targetUrl string, websocketUrl string) *ReverseProxy {
	return &ReverseProxy{
		TargetURL:    targetUrl,
		WebsocketURL: websocketUrl,
		tunnel:       tunnel,
	}
}

func (rp *ReverseProxy) ProxyWebSocket(w http.ResponseWriter, r *http.Request, target *url.URL) {
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Header.Set("Host", target.Host)
	}

	proxy := &httputil.ReverseProxy{
		Director: director,
	}

	proxy.ServeHTTP(w, r)
}

func (rp *ReverseProxy) Launch() {
	targetURL, _ := url.Parse(rp.TargetURL)
	websocketURL, _ := url.Parse(rp.WebsocketURL)

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" && r.URL.Path == "/socket" {
			rp.ProxyWebSocket(w, r, websocketURL)
		} else {
			proxy.ServeHTTP(w, r)
		}
	})

	http.Handle("/socket", websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		backendWS, err := websocket.Dial(rp.WebsocketURL, "", "http://localhost/")

		if err != nil {
			log.Fatal(err)
			return
		}
		defer backendWS.Close()
	}))

	http.Serve(*rp.tunnel, handler)
}
