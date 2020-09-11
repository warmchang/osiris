package cmd

import (
	"context"

	"github.com/golang/glog"

	endpoints "github.com/dailymotion/osiris/pkg/endpoints/controller"
	"github.com/dailymotion/osiris/pkg/kubernetes"
	"github.com/dailymotion/osiris/pkg/version"
)

func RunEndpointsController(ctx context.Context) {
	glog.Infof(
		"Starting Osiris Endpoints Controller -- version %s -- commit %s",
		version.Version(),
		version.Commit(),
	)

	client, err := kubernetes.Client()
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err)
	}

	controllerCfg, err := endpoints.GetConfigFromEnvironment()
	if err != nil {
		glog.Fatalf(
			"Error retrieving endpoints controller configuration: %s",
			err,
		)
	}

	// Run the controller
	endpoints.NewController(controllerCfg, client).Run(ctx)
}
