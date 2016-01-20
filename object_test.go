package pool

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPooledObject(t *testing.T) {
	object := &TestObject{Num: 1}
	pooledObject := NewPooledObject(object)
	pooledObject.MarkReturning()
	assert.Equal(t, StateReturning, pooledObject.GetState())

	pooledObject.MarkAbandoned()
	assert.Equal(t, StateAbandoned, pooledObject.GetState())
}

type TrackedUseObject struct {
	lastUsed int64
}

func (o *TrackedUseObject) GetLastUsed() int64 {
	return o.lastUsed
}

func TestTrackedUse(t *testing.T) {
	time := currentTimeMillis()
	object := &TrackedUseObject{lastUsed: time}
	var trackedUse TrackedUse
	trackedUse = object
	assert.Equal(t, time, trackedUse.GetLastUsed())

	pooledObject := NewPooledObject(object)
	sleep(20)
	pooledObject.Allocate()
	time2 := pooledObject.GetLastUsedTime()
	assert.True(t, time != time2)
	object.lastUsed = currentTimeMillis()
	time3 := pooledObject.GetLastUsedTime()
	assert.Equal(t, object.lastUsed, time3)
}

func TestActiveTimeMillis(t *testing.T) {
	object := &TrackedUseObject{}
	pooledObject := NewPooledObject(object)
	pooledObject.Allocate()
	sleep(20)
	pooledObject.Deallocate()
	assert.True(t, pooledObject.GetActiveTimeMillis() >= 20)
}
