package router_test

import (
	"io"
	"log/slog"
	"strconv"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func itoa(id uint64) string {
	return strconv.FormatUint(id, 10)
}
