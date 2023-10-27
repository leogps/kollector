package watch

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"sync"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/utils-k8s-go/armometadata"
	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"k8s.io/apimachinery/pkg/version"
)

type JsonType int
type StateType int

const (
	NODE          JsonType = 1
	SERVICES      JsonType = 2
	MICROSERVICES JsonType = 3
	PODS          JsonType = 4
	SECRETS       JsonType = 5
	NAMESPACES    JsonType = 6
)

const (
	CREATED StateType = 1
	DELETED StateType = 2
	UPDATED StateType = 3
)

var (
	FirstReportEmptyBytes  = []byte("{\"firstReport\":true}")
	FirstReportEmptyLength = len(FirstReportEmptyBytes)
)

type EventObjectData struct {
	sync.Map
	ProcessedData []string `json:"-"`
}

func (e *EventObjectData) process() {
	e.Range(func(key, value interface{}) bool {
		e.ProcessedData = append(e.ProcessedData, key.(string))
		return true
	})
}

func (e *EventObjectData) deleteProcessed() {
	for _, key := range e.ProcessedData {
		e.Delete(key)
	}
}

func (e *EventObjectData) MarshalJSON() ([]byte, error) {
	var values []interface{}

	e.Range(func(key, value interface{}) bool {
		values = append(values, value)
		return true
	})

	// Marshal the values to JSON
	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func NewEventObjectData() EventObjectData {
	return EventObjectData{
		Map:           sync.Map{},
		ProcessedData: []string{},
	}
}

type ObjectData struct {
	Created EventObjectData `json:"create,omitempty"`
	Deleted EventObjectData `json:"delete,omitempty"`
	Updated EventObjectData `json:"update,omitempty"`
}

func (obj *ObjectData) process() {
	if obj == nil {
		return
	}
	obj.Created.process()
	obj.Updated.process()
	obj.Deleted.process()
}

func NewObjectData() *ObjectData {
	return &ObjectData{
		Created: NewEventObjectData(),
		Deleted: NewEventObjectData(),
		Updated: NewEventObjectData(),
	}
}

type jsonFormat struct {
	FirstReport             bool                        `json:"firstReport"`
	ClusterAPIServerVersion *version.Info               `json:"clusterAPIServerVersion,omitempty"`
	CloudVendor             string                      `json:"cloudVendor,omitempty"`
	Nodes                   *ObjectData                 `json:"node,omitempty"`
	Services                *ObjectData                 `json:"service,omitempty"`
	MicroServices           *ObjectData                 `json:"microservice,omitempty"`
	Pods                    *ObjectData                 `json:"pod,omitempty"`
	Secret                  *ObjectData                 `json:"secret,omitempty"`
	Namespace               *ObjectData                 `json:"namespace,omitempty"`
	InstallationData        *armotypes.InstallationData `json:"installationData,omitempty"`
	//sync.Mutex
}

func NewJsonFormat() jsonFormat {
	return jsonFormat{
		FirstReport:   true,
		Nodes:         NewObjectData(),
		Services:      NewObjectData(),
		MicroServices: NewObjectData(),
		Pods:          NewObjectData(),
		Secret:        NewObjectData(),
		Namespace:     NewObjectData(),
	}
}

// Type alias for the recursive call
type threadSafeJson jsonFormat

func (jsonReport *jsonFormat) MarshalJSON() ([]byte, error) {
	logger.L().Info("marshalling jsonReport now")

	jsonReport.Nodes.process()
	jsonReport.Services.process()
	jsonReport.MicroServices.process()
	jsonReport.Pods.process()
	jsonReport.Secret.process()
	jsonReport.Namespace.process()

	return json.Marshal(threadSafeJson(*jsonReport))
}

func (obj *ObjectData) AddToJsonFormatByState(NewData interface{}, stype StateType) {
	key := uuid.New().String()
	switch stype {
	case CREATED:
		obj.Created.Store(key, NewData)
	case DELETED:
		obj.Deleted.Store(key, NewData)
	case UPDATED:
		obj.Updated.Store(key, NewData)
	}
}

func (jsonReport *jsonFormat) AddToJsonFormat(data interface{}, jtype JsonType, stype StateType) {

	switch jtype {
	case NODE:
		if jsonReport.Nodes == nil {
			jsonReport.Nodes = NewObjectData()
		}
		jsonReport.Nodes.AddToJsonFormatByState(data, stype)
	case SERVICES:
		if jsonReport.Services == nil {
			jsonReport.Services = NewObjectData()
		}
		jsonReport.Services.AddToJsonFormatByState(data, stype)
	case MICROSERVICES:
		if jsonReport.MicroServices == nil {
			jsonReport.MicroServices = NewObjectData()
		}
		jsonReport.MicroServices.AddToJsonFormatByState(data, stype)
	case PODS:
		if jsonReport.Pods == nil {
			jsonReport.Pods = NewObjectData()
		}
		jsonReport.Pods.AddToJsonFormatByState(data, stype)
	case SECRETS:
		if jsonReport.Secret == nil {
			jsonReport.Secret = NewObjectData()
		}
		jsonReport.Secret.AddToJsonFormatByState(data, stype)
	case NAMESPACES:
		if jsonReport.Namespace == nil {
			jsonReport.Namespace = NewObjectData()
		}
		jsonReport.Namespace.AddToJsonFormatByState(data, stype)
	}
}

func prepareDataToSend(ctx context.Context, wh *WatchHandler) []byte {
	logger.L().Ctx(ctx).Info("prepareDataToSend...")
	jsonReport := wh.jsonReport
	if wh.clusterAPIServerVersion == nil {
		return nil
	}
	if *wh.getAggregateFirstDataFlag() {
		setInstallationData(&jsonReport, *wh.config.ClusterConfig())

		jsonReport.ClusterAPIServerVersion = wh.clusterAPIServerVersion
		jsonReport.CloudVendor = wh.cloudVendor
	} else {
		jsonReport.ClusterAPIServerVersion = nil
		jsonReport.CloudVendor = ""
	}

	logger.L().Ctx(ctx).Info("Marshalling jsonReport...")
	jsonReportToSend, err := json.Marshal(&jsonReport)

	if nil != err {
		logger.L().Ctx(ctx).Error("In PrepareDataToSend json.Marshal", helpers.Error(err))
		return nil
	}
	deleteJsonData(wh)
	if *wh.getAggregateFirstDataFlag() && !isEmptyFirstReport(jsonReportToSend) {
		wh.aggregateFirstDataFlag = false
	}
	return jsonReportToSend
}

func isEmptyFirstReport(jsonReportToSend []byte) bool {
	// len==0 is for empty json, len==2 is for "{}"
	if len(jsonReportToSend) == 0 || len(jsonReportToSend) == 2 || len(jsonReportToSend) == FirstReportEmptyLength {
		return true
	}

	return false
}

// WaitTillNewDataArrived -
func WaitTillNewDataArrived(wh *WatchHandler) bool {
	<-wh.informNewDataChannel
	return true
}

func informNewDataArrive(wh *WatchHandler) {
	if !wh.aggregateFirstDataFlag || wh.clusterAPIServerVersion != nil {
		wh.informNewDataChannel <- 1
	}
}

func deleteObjectData(e *EventObjectData) {
	e.deleteProcessed()
}

func deleteJsonData(wh *WatchHandler) {
	jsonReport := &wh.jsonReport
	// DO NOT DELETE jsonReport.ClusterAPIServerVersion data. it's not a subject to change

	if jsonReport.Nodes != nil {
		deleteObjectData(&jsonReport.Nodes.Created)
		deleteObjectData(&jsonReport.Nodes.Deleted)
		deleteObjectData(&jsonReport.Nodes.Updated)
	}

	if jsonReport.Pods != nil {
		deleteObjectData(&jsonReport.Pods.Created)
		deleteObjectData(&jsonReport.Pods.Deleted)
		deleteObjectData(&jsonReport.Pods.Updated)
	}

	if jsonReport.Services != nil {
		deleteObjectData(&jsonReport.Services.Created)
		deleteObjectData(&jsonReport.Services.Deleted)
		deleteObjectData(&jsonReport.Services.Updated)
	}

	if jsonReport.MicroServices != nil {
		deleteObjectData(&jsonReport.MicroServices.Created)
		deleteObjectData(&jsonReport.MicroServices.Deleted)
		deleteObjectData(&jsonReport.MicroServices.Updated)
	}

	if jsonReport.Secret != nil {
		deleteObjectData(&jsonReport.Secret.Created)
		deleteObjectData(&jsonReport.Secret.Deleted)
		deleteObjectData(&jsonReport.Secret.Updated)
	}

	if jsonReport.Namespace != nil {
		deleteObjectData(&jsonReport.Namespace.Created)
		deleteObjectData(&jsonReport.Namespace.Deleted)
		deleteObjectData(&jsonReport.Namespace.Updated)
	}
}

func setInstallationData(jsonReport *jsonFormat, config armometadata.ClusterConfig) {
	jsonReport.InstallationData = &armotypes.InstallationData{}
	jsonReport.InstallationData.Namespace = config.Namespace
	jsonReport.InstallationData.RelevantImageVulnerabilitiesEnabled = config.RelevantImageVulnerabilitiesEnabled
	jsonReport.InstallationData.StorageEnabled = config.StorageEnabled
	jsonReport.InstallationData.ImageVulnerabilitiesScanningEnabled = config.ImageVulnerabilitiesScanningEnabled
	jsonReport.InstallationData.PostureScanEnabled = config.PostureScanEnabled
	jsonReport.InstallationData.OtelCollectorEnabled = config.OtelCollectorEnabled
	jsonReport.InstallationData.ClusterName = config.ClusterName
	jsonReport.InstallationData.ClusterProvider = config.ClusterProvider
	jsonReport.InstallationData.RelevantImageVulnerabilitiesConfiguration = config.RelevantImageVulnerabilitiesConfiguration

	logger.L().Debug("setting installation data", helpers.Interface("installation data", jsonReport.InstallationData))
}
