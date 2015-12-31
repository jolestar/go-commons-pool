package pool
import "sync/atomic"

type AtomicInteger int32

func (this *AtomicInteger) IncrementAndGet() int32 {
	return atomic.AddInt32((*int32)(this), int32(1))
}

func (this *AtomicInteger) GetAndIncrement() int32 {
	ret := int32(*this)
	atomic.AddInt32((*int32)(this), int32(1))
	return ret
}

func (this *AtomicInteger) DecrementAndGet() int32 {
	return atomic.AddInt32((*int32)(this), int32(-1))
}

func (this *AtomicInteger) GetAndDecrement() int32 {
	ret := int32(*this)
	atomic.AddInt32((*int32)(this), int32(-1))
	return ret
}

func (this AtomicInteger) Get() int32  {
	return int32(this)
}

