package collections

import (
	"reflect"
	"sync"
)

type Iterator interface {
	HasNext() bool
	Next() interface{}
	Remove()
}

type SyncIdentityMap struct {
	sync.RWMutex
	m map[uintptr]interface{}
}

func NewSyncMap() *SyncIdentityMap {
	return &SyncIdentityMap{m: make(map[uintptr]interface{})}
}
func (this *SyncIdentityMap) Get(key interface{}) interface{} {
	this.RLock()
	keyPtr := genKey(key)
	value := this.m[keyPtr]
	this.RUnlock()
	return value
}

func genKey(key interface{}) uintptr {
	keyValue := reflect.ValueOf(key)
	return keyValue.Pointer()
}

func (this *SyncIdentityMap) Put(key interface{}, value interface{}) {
	this.Lock()
	keyPtr := genKey(key)
	this.m[keyPtr] = value
	this.Unlock()
}

func (this *SyncIdentityMap) Remove(key interface{}) {
	this.Lock()
	keyPtr := genKey(key)
	delete(this.m, keyPtr)
	this.Unlock()
}

func (this *SyncIdentityMap) Size() int {
	this.RLock()
	defer this.RUnlock()
	return len(this.m)
}

/**
 * for support multi thread, just copy all map value to slice
 */
func (this *SyncIdentityMap) Values() []interface{} {
	this.RLock()
	defer this.RUnlock()
	var list []interface{}
	for _, v := range this.m {
		list = append(list, v)
	}
	return list
}
