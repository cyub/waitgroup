package waitgroup

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type waitgroup struct {
	sync.WaitGroup
	start     chan struct{}
	meanwhile bool
}

func New(meanwhiles ...bool) *waitgroup {
	var meanwhile bool
	if len(meanwhiles) > 0 {
		meanwhile = meanwhiles[0]
	}
	return &waitgroup{
		start:     make(chan struct{}),
		meanwhile: meanwhile,
	}
}

func (wg *waitgroup) Wrap(cb func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if wg.meanwhile {
			<-wg.start
		}
		cb()
	}()
}

func (wg *waitgroup) Start() {
	select {
	case <-wg.start:
		return
	default:
		close(wg.start)
	}
}

func (wg *waitgroup) Finished() bool {
	underfly := *(*waitGroup)(unsafe.Pointer(&wg.WaitGroup))
	var statep *uint64
	if uintptr(unsafe.Pointer(&underfly.state1))%8 == 0 {
		statep = (*uint64)(unsafe.Pointer(&underfly.state1))
	} else {
		statep = (*uint64)(unsafe.Pointer(&underfly.state1[1]))
	}
	state := atomic.LoadUint64(statep)
	v := int32(state >> 32)
	if v < 0 {
		panic("sync: negative WaitGroup counter")
	}
	if v > 0 {
		return false
	}

	return true
}

type noCopy struct{}
type waitGroup struct {
	noCopy noCopy

	// 64-bit value: high 32 bits are counter, low 32 bits are waiter count.
	// 64-bit atomic operations require 64-bit alignment, but 32-bit
	// compilers do not ensure it. So we allocate 12 bytes and then use
	// the aligned 8 bytes in them as state, and the other 4 as storage
	// for the sema.
	state1 [3]uint32
}
