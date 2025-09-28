package util

import (
	"fmt"
	"log/slog"
)

type HexBytes []byte

func (hb HexBytes) LogValue() slog.Value {
	return slog.StringValue(fmt.Sprintf("% X", hb))
}
