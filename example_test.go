package pool

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MyPoolObject struct {
}

func TestExample(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := NewObjectPoolWithDefaultConfig(
		ctx,
		NewPooledObjectFactorySimple(
			func(context.Context) (interface{}, error) {
				return &MyPoolObject{}, nil
			}))
	obj, _ := p.BorrowObject(ctx)
	p.ReturnObject(ctx, obj)
}

type MyObjectFactory struct {
}

func (f *MyObjectFactory) MakeObject(ctx context.Context) (*PooledObject, error) {
	return NewPooledObject(&MyPoolObject{}), nil
}

func (f *MyObjectFactory) DestroyObject(ctx context.Context, object *PooledObject) error {
	//do destroy
	return nil
}

func (f *MyObjectFactory) ValidateObject(ctx context.Context, object *PooledObject) bool {
	//do validate
	return true
}

func (f *MyObjectFactory) ActivateObject(ctx context.Context, object *PooledObject) error {
	//do activate
	return nil
}

func (f *MyObjectFactory) PassivateObject(ctx context.Context, object *PooledObject) error {
	//do passivate
	return nil
}

func TestCustomFactoryExample(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := NewObjectPoolWithDefaultConfig(ctx, new(MyObjectFactory))
	obj, _ := p.BorrowObject(ctx)
	p.ReturnObject(ctx, obj)
}

func TestStringExample(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := NewObjectPoolWithDefaultConfig(ctx, NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			var stringPointer = new(string)
			*stringPointer = "hello"
			return stringPointer, nil
		}))
	obj, _ := p.BorrowObject(ctx)
	fmt.Println(obj)
	assert.Equal(t, "hello", *obj.(*string))
	p.ReturnObject(ctx, obj)
}

//func TestStringExampleFail(t *testing.T) {
//	p := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return "hello", nil
//		}))
//	obj, _ := p.BorrowObject(ctx)
//	p.ReturnObject(obj)
//}
//
//func TestIntExampleFail(t *testing.T) {
//	p := NewObjectPoolWithDefaultConfig(NewPooledObjectFactorySimple(
//		func() (interface{}, error) {
//			return 1, nil
//		}))
//	obj, _ := p.BorrowObject(ctx)
//	p.ReturnObject(obj)
//}
