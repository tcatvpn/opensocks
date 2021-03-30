package osutil

import (
	"log"
	"syscall"
)

func SetSysMaxLimit() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Panicf("[server] Getrlimit error:%v", err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Panicf("[server] Setrlimit error:%v", err)
	}
}
