package concurrent

import "sync/atomic"

type AtomicInteger int32

func (i *AtomicInteger) IncrementAndGet() int32 {
	return atomic.AddInt32((*int32)(i), int32(1))
}

func (i *AtomicInteger) GetAndIncrement() int32 {
	ret := int32(*i)
	atomic.AddInt32((*int32)(i), int32(1))
	return ret
}

func (i *AtomicInteger) DecrementAndGet() int32 {
	return atomic.AddInt32((*int32)(i), int32(-1))
}

func (i *AtomicInteger) GetAndDecrement() int32 {
	ret := int32(*i)
	atomic.AddInt32((*int32)(i), int32(-1))
	return ret
}

func (i AtomicInteger) Get() int32 {
	return int32(i)
}
