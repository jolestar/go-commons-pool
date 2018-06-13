package pool

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestFactoryObject struct {
	status string
}

func TestDefaultPooledObjectFactorySimple(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	factory := NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			return &TestFactoryObject{status: "make"}, nil
		})

	assert.NotNil(t, factory)
	o, _ := factory.MakeObject(ctx)
	if debugTest {
		fmt.Println("object:", o.Object)
	}
	assert.NotNil(t, o)
	assert.Nil(t, factory.ActivateObject(ctx, o))
	assert.Nil(t, factory.PassivateObject(ctx, o))
	assert.True(t, factory.ValidateObject(ctx, o))
	assert.Nil(t, factory.DestroyObject(ctx, o))
}

func TestDefaultPooledObjectFactory(t *testing.T) {
	t.Parallel()

	factory := NewPooledObjectFactory(
		func(context.Context) (interface{}, error) {
			return &TestFactoryObject{status: "make"}, nil
		},
		func(ctx context.Context, object *PooledObject) error {
			object.Object.(*TestFactoryObject).status = "destory"
			return nil
		},
		func(ctx context.Context, object *PooledObject) bool {
			object.Object.(*TestFactoryObject).status = "validate"
			return true
		},
		func(ctx context.Context, object *PooledObject) error {
			object.Object.(*TestFactoryObject).status = "activate"
			return nil
		},
		func(ctx context.Context, object *PooledObject) error {
			object.Object.(*TestFactoryObject).status = "passivate"
			return nil
		})

	assert.NotNil(t, factory)

	ctx := context.Background()
	o, _ := factory.MakeObject(ctx)
	assert.NotNil(t, o)
	obj := o.Object.(*TestFactoryObject)
	assert.Equal(t, "make", obj.status)
	assert.Nil(t, factory.ActivateObject(ctx, o))
	assert.Equal(t, "activate", obj.status)
	assert.Nil(t, factory.PassivateObject(ctx, o))
	assert.Equal(t, "passivate", obj.status)
	assert.True(t, factory.ValidateObject(ctx, o))
	assert.Equal(t, "validate", obj.status)
	assert.Nil(t, factory.DestroyObject(ctx, o))
	assert.Equal(t, "destory", obj.status)
}
