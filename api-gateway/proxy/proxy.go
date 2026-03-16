package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Setup creates a reverse proxy to forward requests to target URLs
func Setup(targetURL string) (http.HandlerFunc, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL %s: %w", targetURL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// Optionally modify the director to adjust paths or headers further
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Ensure correct host is sent downstream
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = parsedURL.Host
	}

	return func(w http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(w, req)
	}, nil
}
