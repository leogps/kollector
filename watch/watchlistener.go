package watch

import (
	"context"
	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"runtime/debug"
)

func (wh *WatchHandler) ListenAndProcess(ctx context.Context, processor EventProcessor) {
	wh.ListenAndProcessWithStartSignal(ctx, processor, make(chan struct{}))
}

func (wh *WatchHandler) ListenAndProcessWithStartSignal(ctx context.Context, processor EventProcessor, startSignal chan<- struct{}) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER ListenAndProcess", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	wh.SetFirstReportFlag(true)

	close(startSignal)
	for {
		jsonData := prepareDataToSend(ctx, wh)
		if jsonData == nil || isEmptyFirstReport(jsonData) {
			continue // skip (usually first) report in case it is empty
		}
		if jsonData != nil {
			logger.L().Ctx(ctx).Debug("sending report to websocket", helpers.String("report", string(jsonData)))
			processor.ProcessEventData(jsonData)
		}
		if wh.getFirstReportFlag() {
			wh.SetFirstReportFlag(false)
		}
		if WaitTillNewDataArrived(wh) {
			continue
		}
	}
}
