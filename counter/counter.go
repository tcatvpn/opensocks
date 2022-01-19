package counter

import "sync/atomic"

var TotalReadBytes uint64 = 0
var TotalWrittenBytes uint64 = 0
var TotalConnections uint64 = 0
var CurrentConnections uint64 = 0

func IncrReadBytes(n int) {
	atomic.AddUint64(&TotalReadBytes, uint64(n))
}

func IncrWrittenBytes(n int) {
	atomic.AddUint64(&TotalWrittenBytes, uint64(n))
}

func IncrConnections(n int) {
	atomic.AddUint64(&TotalConnections, uint64(n))
}

func IncrCurrentConnections(n int) {
	atomic.AddUint64(&CurrentConnections, uint64(n))
}

func DecrCurrentConnections(n int) {
	atomic.AddUint64(&CurrentConnections, -uint64(n))
}
