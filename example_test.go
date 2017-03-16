package pool

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MyPoolObject struct {
}

func TestExample(t *testing.T) {
	p := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			return &MyPoolObject{}, nil
		}))
	obj, _ := p.BorrowObject()
	p.ReturnObject(obj)
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
	p := NewObjectPoolWithDefaultConfig(new(MyObjectFactory))
	obj, _ := p.BorrowObject()
	p.ReturnObject(obj)
}

func TestStringExample(t *testing.T) {
	p := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			var stringPointer = new(string)
			*stringPointer = "hello"
			return stringPointer, nil
		}))
	obj, _ := p.BorrowObject()
	fmt.Println(obj)
	assert.Equal(t, "hello", *obj.(*string))
	p.ReturnObject(obj)
}

//func TestStringExampleFail(t *testing.T) {
//	p := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return "hello", nil
//		}))
//	obj, _ := p.BorrowObject()
//	p.ReturnObject(obj)
//}
//
//func TestIntExampleFail(t *testing.T) {
//	p := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return 1, nil
//		}))
//	obj, _ := p.BorrowObject()
//	p.ReturnObject(obj)
//}
