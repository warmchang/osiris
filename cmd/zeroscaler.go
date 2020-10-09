package cmd

import (
	"context"

	"github.com/golang/glog"

	deployments "github.com/dailymotion-oss/osiris/pkg/deployments/zeroscaler"
	"github.com/dailymotion-oss/osiris/pkg/kubernetes"
	"github.com/dailymotion-oss/osiris/pkg/version"
)

func RunZeroScaler(ctx context.Context) {
	glog.Infof(
		"Starting Osiris Zeroscaler -- version %s -- commit %s",
		version.Version(),
		version.Commit(),
	)

	client, err := kubernetes.Client()
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	cfg, err := deployments.GetConfigFromEnvironment()
	if err != nil {
		glog.Fatalf("Error getting zeroscaler envconfig: %s", err.Error())
	}

	// Run the zeroscaler
	deployments.NewZeroscaler(cfg, client).Run(ctx)
}
