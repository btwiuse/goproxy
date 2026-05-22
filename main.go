package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// followTransport is an http.RoundTripper that uses an http.Client
// configured to follow redirects. This ensures the reverse proxy
// always returns the final response (e.g. after proxy.golang.org
// redirects to Google Cloud Storage), without the client needing
// to follow redirects itself.
type followTransport struct {
	inner http.RoundTripper
}

func (t followTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Don't follow redirects ourselves; let the client do it.
	client := &http.Client{
		Transport: t.inner,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	return client.Do(req)
}

func newFollowTransport() followTransport {
	return followTransport{inner: http.DefaultTransport}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Encoding")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	target := os.Getenv("GOPROXY_TARGET")
	if target == "" {
		target = "https://proxy.golang.org"
	}
	addr := os.Getenv("GOPROXY_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid target URL %q: %v", target, err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.Out.Host = targetURL.Host
			r.Out.RequestURI = "" // http.Client rejects requests with RequestURI set
		},
		Transport: newFollowTransport(),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("ERROR: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
		},
	}

	log.Printf("Listening on %s, forwarding to %s", addr, target)
	log.Fatal(http.ListenAndServe(addr, cors(proxy)))
}
