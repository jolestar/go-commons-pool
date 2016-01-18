package collections

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

type HashableObject struct {
	str string
	i   int
	b   bool
}

type UnhashableObject struct {
	str  string
	i    int
	b    bool
	strs []string
	m    map[string]int
}

func TestSyncMapString(t *testing.T) {
	m := NewSyncMap()
	key := "key"
	//var key *interface{}
	//key = &"key1"
	m.Put(&key, "value1")
	assert.Equal(t, "value1", m.Get(&key))
	m.Remove(&key)
	assert.Nil(t, m.Get(&key))
}

func TestSyncMapValues(t *testing.T) {
	m := NewSyncMap()
	key := "key"
	key2 := "key2"
	m.Put(&key, "value1")
	m.Put(&key2, "value2")
	values := m.Values()
	assert.Equal(t, 2, len(values))
}

func TestSyncMapHashableObject(t *testing.T) {
	m := NewSyncMap()
	o1 := HashableObject{}
	m.Put(&o1, "value1")
	assert.Equal(t, "value1", m.Get(&o1))
	//change object
	o1.str = "str"
	o1.i = 6
	assert.Equal(t, "value1", m.Get(&o1))
}

func TestSyncMapHashableObject2(t *testing.T) {
	m := NewSyncMap()
	o1 := HashableObject{}
	m.Put(&o1, "value1")
	assert.Equal(t, "value1", m.Get(&o1))

	o2 := HashableObject{}
	assert.Nil(t, m.Get(&o2))
}

func TestSyncMapHashableObject3(t *testing.T) {
	m := NewSyncMap()
	o1 := HashableObject{}
	m.Put(&o1, &o1)
	o1.str = "h"
	assert.Equal(t, &o1, m.Get(&o1))
}

func TestSyncMapUnhashableObject(t *testing.T) {
	m := NewSyncMap()
	o1 := UnhashableObject{}
	m.Put(&o1, "value1")
	assert.Equal(t, "value1", m.Get(&o1))
	//change object
	o1.str = "str"
	o1.i = 6
	assert.Equal(t, "value1", m.Get(&o1))
}

func TestMultiThread(t *testing.T) {
	m := NewSyncMap()
	wait := sync.WaitGroup{}
	wait.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			o1 := HashableObject{}
			m.Put(&o1, &o1)
			wait.Done()
		}()
	}
	wait.Wait()
	assert.Equal(t, 1000, m.Size())
}
