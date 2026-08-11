package stats

import (
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type Stats struct {
	HeapAlloc    uint64
	HeapSys      uint64
	Sys          uint64
	NumGoroutine int
	RSSBytes     uint64
}

func Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return c.JSON(http.StatusOK, Stats{
			HeapAlloc:    m.HeapAlloc,
			HeapSys:      m.HeapSys,
			Sys:          m.Sys,
			NumGoroutine: runtime.NumGoroutine(),
			RSSBytes:     rssBytes(),
		})
	}
}

func rssBytes() uint64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return pages * uint64(os.Getpagesize())
}
