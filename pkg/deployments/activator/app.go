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
	Namespace           string
	ServiceName         string
	Name                string
	Kind                appKind
	Dependencies        []*app
	TargetURL           *url.URL
	proxyRequestHandler *httputil.ReverseProxy
}
