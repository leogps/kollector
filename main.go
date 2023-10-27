package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kubescape/backend/pkg/utils"
	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"

	"github.com/leogps/kollector/config"
	"github.com/leogps/kollector/consts"
	"github.com/leogps/kollector/watch"

	"github.com/armosec/utils-k8s-go/armometadata"
	"github.com/armosec/utils-k8s-go/probes"
)

func main() {
	ctx := context.Background()

	isServerReady := false
	go probes.InitReadinessV1(&isServerReady)
	displayBuildTag()

	//clusterConfig, err := armometadata.LoadConfig(os.Getenv(consts.ConfigEnvironmentVariable))
	//if err != nil {
	//	logger.L().Ctx(ctx).Fatal("failed to load config", helpers.Error(err))
	//}
	//
	//services, err := servicediscovery.GetServices(
	//	v1.NewServiceDiscoveryFileV1("/etc/config/services.json"),
	//)
	//if err != nil {
	//	logger.L().Ctx(ctx).Fatal("failed to load services", helpers.Error(err))
	//}
	//
	//logger.L().Info("loaded event receiver websocket url (service discovery)", helpers.String("url", services.GetReportReceiverWebsocketUrl()))
	//
	//var credentials *utils.Credentials
	//if credentials, err = utils.LoadCredentialsFromFile("/etc/credentials"); err != nil {
	//	logger.L().Ctx(ctx).Error("failed to load credentials", helpers.Error(err))
	//	credentials = &utils.Credentials{}
	//} else {
	//	logger.L().Info("credentials loaded",
	//		helpers.Int("accessKeyLength", len(credentials.AccessKey)),
	//		helpers.Int("accountLength", len(credentials.Account)))
	//}
	//
	//kollectorConfig := config.NewKollectorConfig(clusterConfig, *credentials, services.GetReportReceiverWebsocketUrl())
	//
	//// to enable otel, set OTEL_COLLECTOR_SVC=otel-collector:4317
	//if otelHost, present := os.LookupEnv(consts.OtelCollectorSvcEnvironmentVariable); present {
	//	ctx = logger.InitOtel("kollector",
	//		os.Getenv(consts.ReleaseBuildTagEnvironmentVariable),
	//		kollectorConfig.AccountID(),
	//		kollectorConfig.ClusterName(),
	//		url.URL{Host: otelHost})
	//	defer logger.ShutdownOtel(ctx)
	//}

	credentials := &utils.Credentials{}
	kollectorConfig := config.NewKollectorConfig(&armometadata.ClusterConfig{
		//ClusterName: clusterName,
	}, *credentials, "")

	wh, err := watch.CreateWatchHandler(kollectorConfig)
	if err != nil {
		logger.L().Ctx(ctx).Fatal("failed to initialize the WatchHandler", helpers.Error(err))
	}

	//websocketEventProcessor := watch.WebSocketProcessor{
	//	WatchHandler: wh,
	//}

	debugProcessor := DebugProcessor{
		ctx,
	}

	wh.RetrieveClusterInfo()
	signal := make(chan struct{})
	go func() {
		for {
			logger.L().Ctx(ctx).Info("Invoking wh.ListenAndProcess...")
			wh.ListenAndProcess(ctx, &debugProcessor)
		}
		close(signal)
	}()

	go func() {
		for {
			wh.NodeWatch(ctx)
		}
	}()

	go func() {
		for {
			wh.PodWatch(ctx)
		}
	}()

	go func() {
		for {
			wh.ServiceWatch(ctx)
		}
	}()

	go func() {
		for {
			wh.SecretWatch(ctx)
		}
	}()
	go func() {
		for {
			wh.NamespaceWatch(ctx)
		}
	}()
	go func() {
		for {
			wh.CronJobWatch(ctx)
		}
	}()
	//	logger.L().Ctx(ctx).Fatal(wh.WebSocketHandle.SendReportRoutine(ctx, &isServerReady, wh.SetFirstReportFlag).Error())
	<-signal
}

func displayBuildTag() {
	flag.Parse()
	logger.L().Info(fmt.Sprintf("Image version: %s", os.Getenv(consts.ReleaseBuildTagEnvironmentVariable)))
}

type DebugProcessor struct {
	Context context.Context
}

func (d DebugProcessor) ProcessEventData(jsonData []byte) {
	event := string(jsonData)
	var data map[string]interface{}
	err := json.Unmarshal([]byte(event), &data)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}
	msg := fmt.Sprintf("Length: %d", len(data))
	logger.L().Ctx(d.Context).Info(msg)
	n := data["namespace"]
	if n != nil {
		c := n.(map[string]interface{})["create"]
		if c != nil {
			s := c.([]interface{})
			l := len(s)
			createCount += l
			m := fmt.Sprintf("Total create count: %d", createCount)
			logger.L().Ctx(d.Context).Info(m)
		}
	}

	logger.L().Ctx(d.Context).Info(event)
}

var (
	createCount int
)
