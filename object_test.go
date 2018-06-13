package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	lastUsed time.Time
}

func (o *TrackedUseObject) GetLastUsed() time.Time {
	return o.lastUsed
}

func TestTrackedUse(t *testing.T) {
	now := time.Now()
	object := &TrackedUseObject{lastUsed: now}
	var trackedUse TrackedUse
	trackedUse = object
	assert.Equal(t, now, trackedUse.GetLastUsed())

	pooledObject := NewPooledObject(object)
	time.Sleep(20 * time.Millisecond)
	pooledObject.Allocate()
	time2 := pooledObject.GetLastUsedTime()
	assert.True(t, now != time2)
	object.lastUsed = time.Now()
	time3 := pooledObject.GetLastUsedTime()
	assert.Equal(t, object.lastUsed, time3)
}

func TestActiveTime(t *testing.T) {
	object := &TrackedUseObject{}
	pooledObject := NewPooledObject(object)
	pooledObject.Allocate()
	time.Sleep(20 * time.Millisecond)
	pooledObject.Deallocate()
	assert.True(t, pooledObject.GetActiveTime() >= 20*time.Millisecond)
}
