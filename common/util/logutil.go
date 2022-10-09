package util

import (
	"log"
	"time"

	"github.com/net-byte/opensocks/counter"
)

// PrintLog returns the log
func PrintLog(enableVerbose bool, format string, v ...any) {
	if !enableVerbose {
		return
	}
	log.Printf("[info] "+format, v)
}

// PrintStats returns the stats info
func PrintStats(enableVerbose bool, serverMode bool) {
	if !enableVerbose {
		return
	}
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if serverMode {
				log.Printf("stats:%v", counter.PrintServerBytes())
			} else {
				log.Printf("stats:%v", counter.PrintClientBytes())
			}
		}
	}()
}
