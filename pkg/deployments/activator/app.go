package activator

import (
	"net/http/httputil"
	"net/url"
)

type appKind string

const (
	appKindDeployment  appKind = "Deployment"
	appKindStatefulSet appKind = "StatefulSet"
)

type app struct {
	namespace           string
	serviceName         string
	name                string
	kind                appKind
	targetURL           *url.URL
	proxyRequestHandler *httputil.ReverseProxy
	dependencies        []*app
}
