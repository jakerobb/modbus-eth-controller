package pkg

import (
	"context"
	"fmt"
)

func LogDebug(ctx context.Context, format string, args ...interface{}) {
	if ctx.Value("debug") == true {
		fmt.Printf(format, args...)
	}
}
