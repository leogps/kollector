package watch

import (
	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
)

func (wh *WatchHandler) RetrieveClusterInfo() {
	wh.clusterAPIServerVersion = wh.ClusterVersion()
	wh.cloudVendor = wh.CheckInstanceMetadataAPIVendor()
	if wh.cloudVendor != "" {
		wh.clusterAPIServerVersion.GitVersion += ";" + wh.cloudVendor
	}
	logger.L().Info("K8s Cloud Vendor", helpers.String("cloudVendor", wh.cloudVendor))
}
