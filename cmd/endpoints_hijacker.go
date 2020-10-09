package cmd

import (
	"context"

	"github.com/golang/glog"

	endpoints "github.com/dailymotion-oss/osiris/pkg/endpoints/hijacker"
	"github.com/dailymotion-oss/osiris/pkg/version"
)

func RunEndpointsHijacker(ctx context.Context) {
	glog.Infof(
		"Starting Osiris Endpoints Hijacker -- version %s -- commit %s",
		version.Version(),
		version.Commit(),
	)

	cfg, err := endpoints.GetConfigFromEnvironment()
	if err != nil {
		glog.Fatalf(
			"Error retrieving proxy endpoints hijacker webhook server "+
				"configuration: %s",
			err,
		)
	}

	// Run the server
	endpoints.NewHijacker(cfg).Run(ctx)
}
