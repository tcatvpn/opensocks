package counter

import "sync/atomic"

// TotalReadBytes is the total number of bytes read
var TotalReadBytes uint64 = 0

// TotalWrittenBytes is the total number of bytes written
var TotalWrittenBytes uint64 = 0

// IncrReadBytes increments the number of bytes read
func IncrReadBytes(n int) {
	atomic.AddUint64(&TotalReadBytes, uint64(n))
}

// IncrWrittenBytes increments the number of bytes written
func IncrWrittenBytes(n int) {
	atomic.AddUint64(&TotalWrittenBytes, uint64(n))
}

// Clean clean the counter
func Clean() {
	TotalReadBytes = 0
	TotalWrittenBytes = 0
}
