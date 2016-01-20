package pool

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MyPoolObject struct {
}

func TestExample(t *testing.T) {
	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			return &MyPoolObject{}, nil
		}))
	obj, _ := pool.BorrowObject()
	pool.ReturnObject(obj)
}

type MyObjectFactory struct {
}

func (f *MyObjectFactory) MakeObject() (*PooledObject, error) {
	return NewPooledObject(&MyPoolObject{}), nil
}

func (f *MyObjectFactory) DestroyObject(object *PooledObject) error {
	//do destroy
	return nil
}

func (f *MyObjectFactory) ValidateObject(object *PooledObject) bool {
	//do validate
	return true
}

func (f *MyObjectFactory) ActivateObject(object *PooledObject) error {
	//do activate
	return nil
}

func (f *MyObjectFactory) PassivateObject(object *PooledObject) error {
	//do passivate
	return nil
}

func TestCustomFactoryExample(t *testing.T) {
	pool := NewObjectPoolWithDefaultConfig(new(MyObjectFactory))
	obj, _ := pool.BorrowObject()
	pool.ReturnObject(obj)
}

func TestStringExample(t *testing.T) {
	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			var stringPointer = new(string)
			*stringPointer = "hello"
			return stringPointer, nil
		}))
	obj, _ := pool.BorrowObject()
	fmt.Println(obj)
	assert.Equal(t, "hello", *obj.(*string))
	pool.ReturnObject(obj)
}

//func TestStringExampleFail(t *testing.T) {
//	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return "hello", nil
//		}))
//	obj, _ := pool.BorrowObject()
//	pool.ReturnObject(obj)
//}

//func TestIntExampleFail(t *testing.T) {
//	pool := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return 1, nil
//		}))
//	obj, _ := pool.BorrowObject()
//	pool.ReturnObject(obj)
//}
