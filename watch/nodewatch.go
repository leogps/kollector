package watch

import (
	"container/list"
	"k8s.io/apimachinery/pkg/types"
	"runtime/debug"
	"strings"
	"time"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"golang.org/x/net/context"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type NodeData struct {
	// core.NodeSystemInfo
	core.NodeStatus `json:",inline"`
	Name            string    `json:"name"`
	UID             types.UID `json:"uid"`
	ResourceVersion string    `json:"resourceVersion"`
}

func (updateNode *NodeData) UpdateNodeData(node *core.Node) {
	updateNode.Name = node.ObjectMeta.Name
	updateNode.NodeStatus = node.Status
	updateNode.UID = node.ObjectMeta.UID
	updateNode.ResourceVersion = node.ResourceVersion
}

func UpdateNode(node *core.Node, ndm map[int]*list.List) *NodeData {

	var nd *NodeData
	for _, v := range ndm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.Name) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			logger.L().Debug("node updated", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nd = v.Front().Value.(*NodeData)
			break
		}
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.GenerateName) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			logger.L().Debug("node updated", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nd = v.Front().Value.(*NodeData)
			break
		}
	}
	return nd
}

func RemoveNode(node *core.Node, ndm map[int]*list.List) string {

	var nodeName string
	for _, v := range ndm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.Name) == 0 {
			logger.L().Debug("node removed", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nodeName = v.Front().Value.(*NodeData).Name
			v.Remove(v.Front())
			break
		}
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.GenerateName) == 0 {
			logger.L().Debug("node removed", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nodeName = v.Front().Value.(*NodeData).Name
			v.Remove(v.Front())
			break
		}
	}
	return nodeName
}

// NodeWatch Watching over nodes
func (wh *WatchHandler) NodeWatch(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER NodeWatch", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		wh.RetrieveClusterInfo()
		logger.L().Info("Watching over nodes starting")
		nodesWatcher, err := wh.RestAPIClient.CoreV1().Nodes().Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		wh.handleNodeWatch(nodesWatcher, newStateChan, &lastWatchEventCreationTime)

		if wh.cancelled {
			logger.L().Info("Watching over nodes cancelled")
			break
		}
	}
}
func (wh *WatchHandler) handleNodeWatch(nodesWatcher watch.Interface, newStateChan <-chan bool, lastWatchEventCreationTime *time.Time) {
	nodesChan := nodesWatcher.ResultChan()
	for {
		var event watch.Event
		select {
		case event = <-nodesChan:
		case <-newStateChan:
			nodesWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if event.Type == watch.Error {
			nodesWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if node, ok := event.Object.(*core.Node); ok {
			node.ManagedFields = []metav1.ManagedFieldsEntry{}
			switch event.Type {
			case watch.Added:
				if node.CreationTimestamp.Time.Before(*lastWatchEventCreationTime) {
					continue
				}
				id := CreateID()
				if wh.ndm[id] == nil {
					wh.ndm[id] = list.New()
				}
				nd := &NodeData{
					UID:             node.ObjectMeta.UID,
					ResourceVersion: node.ResourceVersion,
					Name:            node.ObjectMeta.Name,
					NodeStatus:      node.Status,
				}
				wh.ndm[id].PushBack(nd)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(nd, NODE, CREATED)
			case watch.Modified:
				updateNode := UpdateNode(node, wh.ndm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(updateNode, NODE, UPDATED)
			case watch.Deleted:
				name := RemoveNode(node, wh.ndm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(name, NODE, DELETED)
			case watch.Bookmark: //only the resource version is changed but it's the same workload
				continue
			case watch.Error:
				*lastWatchEventCreationTime = time.Now()
				return
			}
		} else {
			*lastWatchEventCreationTime = time.Now()
			return
		}
	}
}
