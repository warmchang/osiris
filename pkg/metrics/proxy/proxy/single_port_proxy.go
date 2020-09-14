package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
)

type singlePortProxy struct {
	appPort             int
	requestCount        *uint64
	srv                 *http.Server
	proxyRequestHandler *httputil.ReverseProxy
	ignoredPaths        map[string]struct{}
}

func newSinglePortProxy(
	proxyPort int,
	appPort int,
	requestCount *uint64,
	ignoredPaths map[string]struct{},
) (*singlePortProxy, error) {
	targetURL, err := url.Parse(fmt.Sprintf("http://localhost:%d", appPort))
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	s := &singlePortProxy{
		appPort:      appPort,
		requestCount: requestCount,
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", proxyPort),
			Handler: mux,
		},
		proxyRequestHandler: httputil.NewSingleHostReverseProxy(targetURL),
		ignoredPaths:        ignoredPaths,
	}
	mux.HandleFunc("/", s.handleRequest)
	return s, nil
}

func (s *singlePortProxy) run(ctx context.Context) {
	doneCh := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done(): // Context was canceled or expired
			glog.Infof(
				"Proxy listening on %s proxying application port %d is shutting down",
				s.srv.Addr,
				s.appPort,
			)
			// Allow up to five seconds for requests in progress to be completed
			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*5,
			)
			defer cancel()
			s.srv.Shutdown(shutdownCtx) // nolint: errcheck
		case <-doneCh: // The server shut down on its own, perhaps due to error
		}
	}()

	glog.Infof(
		"Proxy listening on %s is proxying application port %d",
		s.srv.Addr,
		s.appPort,
	)
	err := s.srv.ListenAndServe()
	if err != http.ErrServerClosed {
		glog.Errorf(
			"Error from proxy listening on %s is proxying application port %d: %s",
			s.srv.Addr,
			s.appPort,
			err,
		)
	}
	close(doneCh)
}

func (s *singlePortProxy) handleRequest(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer r.Body.Close()

	if !s.isIgnoredRequest(r) {
		atomic.AddUint64(s.requestCount, 1)
	}

	s.proxyRequestHandler.ServeHTTP(w, r)
}

func (s *singlePortProxy) isIgnoredRequest(r *http.Request) bool {
	return s.isIgnoredPath(r) || isKubeProbe(r)
}

func (s *singlePortProxy) isIgnoredPath(r *http.Request) bool {
	if r.URL == nil || len(r.URL.Path) == 0 {
		return false
	}
	_, found := s.ignoredPaths[r.URL.Path]
	return found
}

func isKubeProbe(r *http.Request) bool {
	return strings.Contains(r.Header.Get("User-Agent"), "kube-probe")
}
