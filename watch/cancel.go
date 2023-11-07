package watch

import (
	"context"
	"github.com/kubescape/go-logger"
)

func (wh *WatchHandler) Cancel() {
	logger.L().Info("Cancelling watch context...")
	wh.cancelled = true
	globalHTTPContextCancelFunc()
	wh.informNewDataChannel <- 1
	logger.L().Info("Cancelling watch context done.")
}

func (wh *WatchHandler) Reset() {
	wh.cancelled = false
	globalHTTPContext, globalHTTPContextCancelFunc = context.WithCancel(context.Background())
}
