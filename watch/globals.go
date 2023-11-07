package watch

import (
	"context"
)

var (
	globalHTTPContext, globalHTTPContextCancelFunc = context.WithCancel(context.Background())
)
