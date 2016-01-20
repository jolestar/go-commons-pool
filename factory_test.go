package pool

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestFactoryObject struct {
	status string
}

func TestDefaultPooledObjectFactorySimple(t *testing.T) {
	factory := NewPooledObjectFactorySimple(
		func() (interface{}, error) {
			return &TestFactoryObject{status: "make"}, nil
		})

	assert.NotNil(t, factory)
	o, _ := factory.MakeObject()
	if debugTest {
		fmt.Println("object:", o.Object)
	}
	assert.NotNil(t, o)
	assert.Nil(t, factory.ActivateObject(o))
	assert.Nil(t, factory.PassivateObject(o))
	assert.True(t, factory.ValidateObject(o))
	assert.Nil(t, factory.DestroyObject(o))
}

func TestDefaultPooledObjectFactory(t *testing.T) {
	factory := NewPooledObjectFactory(
		func() (interface{}, error) {
			return &TestFactoryObject{status: "make"}, nil
		},
		func(object *PooledObject) error {
			object.Object.(*TestFactoryObject).status = "destory"
			return nil
		},
		func(object *PooledObject) bool {
			object.Object.(*TestFactoryObject).status = "validate"
			return true
		},
		func(object *PooledObject) error {
			object.Object.(*TestFactoryObject).status = "activate"
			return nil
		},
		func(object *PooledObject) error {
			object.Object.(*TestFactoryObject).status = "passivate"
			return nil
		})

	assert.NotNil(t, factory)
	o, _ := factory.MakeObject()
	assert.NotNil(t, o)
	obj := o.Object.(*TestFactoryObject)
	assert.Equal(t, "make", obj.status)
	assert.Nil(t, factory.ActivateObject(o))
	assert.Equal(t, "activate", obj.status)
	assert.Nil(t, factory.PassivateObject(o))
	assert.Equal(t, "passivate", obj.status)
	assert.True(t, factory.ValidateObject(o))
	assert.Equal(t, "validate", obj.status)
	assert.Nil(t, factory.DestroyObject(o))
	assert.Equal(t, "destory", obj.status)
}
