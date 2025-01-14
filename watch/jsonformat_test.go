package watch

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/utils-k8s-go/armometadata"
)

var (
	wh WatchHandler
)

func TestJson(test *testing.T) {
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), NODE, CREATED)
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), SERVICES, DELETED)
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), PODS, UPDATED)

	if !bytes.Equal(wh.jsonReport.Nodes.Created[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) {
		test.Errorf("NODE")
	}
	if !bytes.Equal(wh.jsonReport.Services.Deleted[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) {
		test.Errorf("SERVICES")
	}
	if !bytes.Equal(wh.jsonReport.Pods.Updated[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) {
		test.Errorf("PODS")
	}
}

func TestIsEmptyFirstReport(test *testing.T) {
	jsonReport := &jsonFormat{FirstReport: true}
	jsonReportToSend, _ := json.Marshal(jsonReport)
	if !isEmptyFirstReport(jsonReportToSend) {
		test.Errorf("First report is empty")
	}
	jsonReport.CloudVendor = "aws"
	jsonReportToSend, _ = json.Marshal(jsonReport)
	if isEmptyFirstReport(jsonReportToSend) {
		test.Errorf("First report is not empty")
	}
}

func TestSetInstallationData(t *testing.T) {
	trueBool := true
	falseBool := false
	testCases := []struct {
		name   string
		config armometadata.ClusterConfig
	}{

		{
			name: "all true",
			config: armometadata.ClusterConfig{
				InstallationData: armotypes.InstallationData{
					Namespace:                           "test-namespace",
					RelevantImageVulnerabilitiesEnabled: &trueBool,
					StorageEnabled:                      &trueBool,
					ImageVulnerabilitiesScanningEnabled: &trueBool,
					PostureScanEnabled:                  &trueBool,
					OtelCollectorEnabled:                &trueBool,
					ClusterName:                         "test-cluster",
				},
			},
		},
		{
			name: "all false",
			config: armometadata.ClusterConfig{
				InstallationData: armotypes.InstallationData{
					Namespace:                           "test-namespace",
					RelevantImageVulnerabilitiesEnabled: &falseBool,
					StorageEnabled:                      &falseBool,
					ImageVulnerabilitiesScanningEnabled: &falseBool,
					PostureScanEnabled:                  &falseBool,
					OtelCollectorEnabled:                &falseBool,
					ClusterName:                         "test-cluster"},
			},
		},
		{
			name: "half true half false",
			config: armometadata.ClusterConfig{
				InstallationData: armotypes.InstallationData{
					Namespace:                           "test-namespace",
					RelevantImageVulnerabilitiesEnabled: &falseBool,
					StorageEnabled:                      &trueBool,
					ImageVulnerabilitiesScanningEnabled: &falseBool,
					PostureScanEnabled:                  &trueBool,
					OtelCollectorEnabled:                &falseBool,
					ClusterName:                         "test-cluster"},
			},
		},
		{
			name: "empty",
			config: armometadata.ClusterConfig{
				InstallationData: armotypes.InstallationData{
					Namespace:   "test-namespace",
					ClusterName: "test-cluster"},
			},
		},
		{
			name: "nil pointer",
			config: armometadata.ClusterConfig{
				InstallationData: armotypes.InstallationData{
					OtelCollectorEnabled: nil,
				},
			},
		},
		{
			name: "cluster provider",
			config: armometadata.ClusterConfig{
				InstallationData: armotypes.InstallationData{
					ClusterProvider: "eks",
				},
			},
		},
		{
			name: "relevancy configuration",
			config: armometadata.ClusterConfig{
				InstallationData: armotypes.InstallationData{
					RelevantImageVulnerabilitiesConfiguration: "disable",
				},
			},
		},
	}

	for _, tc := range testCases {
		jsonReport := &jsonFormat{}
		setInstallationData(jsonReport, tc.config)

		if jsonReport.InstallationData.Namespace != tc.config.Namespace {
			t.Errorf("Namespace is not equal")
		}
		if jsonReport.InstallationData.RelevantImageVulnerabilitiesEnabled != tc.config.RelevantImageVulnerabilitiesEnabled {
			t.Errorf("RelevantImageVulnerabilitiesEnabled is not equal")
		}
		if jsonReport.InstallationData.StorageEnabled != tc.config.StorageEnabled {
			t.Errorf("StorageEnabled is not equal")
		}
		if jsonReport.InstallationData.ImageVulnerabilitiesScanningEnabled != tc.config.ImageVulnerabilitiesScanningEnabled {
			t.Errorf("ImageVulnerabilitiesScanningEnabled is not equal")
		}
		if jsonReport.InstallationData.PostureScanEnabled != tc.config.PostureScanEnabled {
			t.Errorf("PostureScanEnabled is not equal")
		}
		if jsonReport.InstallationData.OtelCollectorEnabled != tc.config.OtelCollectorEnabled {
			t.Errorf("OtelCollectorEnabled is not equal")
		}
		if jsonReport.InstallationData.ClusterName != tc.config.ClusterName {
			t.Errorf("ClusterName is not equal")
		}
		if jsonReport.InstallationData.RelevantImageVulnerabilitiesConfiguration != tc.config.RelevantImageVulnerabilitiesConfiguration {
			t.Errorf("RelevantImageVulnerabilitiesConfiguration is not equal")
		}

		if jsonReport.InstallationData.ClusterProvider != tc.config.ClusterProvider {
			t.Errorf("ClusterProvider is not equal")
		}
	}
}
