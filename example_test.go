package pool

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
)

func Example_multiple_borrowers() {
	type myPoolObject struct {
		s string
	}

	var v uint64
	factory := NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			return &myPoolObject{
					s: strconv.FormatUint(atomic.AddUint64(&v, 1), 10),
				},
				nil
		})

	ctx := context.Background()
	p := NewObjectPoolWithDefaultConfig(ctx, factory)

	// Borrows #1
	obj1, err := p.BorrowObject(ctx)
	if err != nil {
		panic(err)
	}

	o := obj1.(*myPoolObject)
	fmt.Println(o.s)

	// Borrowing again while the first object is borrowed will cause a new object to be made, if
	// the pool configuration allows it. If the pull is full, this will block until the context
	// is cancelled or an object is returned to the pool.
	//
	// Borrows #2
	obj2, err := p.BorrowObject(ctx)
	if err != nil {
		panic(err)
	}

	// Returning the object to the pool makes it available to another borrower.
	err = p.ReturnObject(ctx, obj1)
	if err != nil {
		panic(err)
	}

	// Since there's an object available in the pool, this gets that rather than creating a new one.
	//
	// Borrows #1 again (since it was returned earlier)
	obj3, err := p.BorrowObject(ctx)
	if err != nil {
		panic(err)
	}

	o = obj2.(*myPoolObject)
	fmt.Println(o.s)

	err = p.ReturnObject(ctx, obj2)
	if err != nil {
		panic(err)
	}

	o = obj3.(*myPoolObject)
	fmt.Println(o.s)

	err = p.ReturnObject(ctx, obj3)
	if err != nil {
		panic(err)
	}

	// Output:
	// 1
	// 2
	// 1
}

func ExampleSimpleFactory() {
	type myPoolObject struct {
		s string
	}

	v := uint64(0)
	factory := NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			return &myPoolObject{
					s: strconv.FormatUint(atomic.AddUint64(&v, 1), 10),
				},
				nil
		})

	ctx := context.Background()
	p := NewObjectPoolWithDefaultConfig(ctx, factory)
	obj, err := p.BorrowObject(ctx)
	if err != nil {
		panic(err)
	}

	o := obj.(*myPoolObject)
	fmt.Println(o.s)

	err = p.ReturnObject(ctx, obj)
	if err != nil {
		panic(err)
	}

	// Output: 1
}

type MyPoolObject struct {
	s string
}

type MyCustomFactory struct {
	v uint64
}

func (f *MyCustomFactory) MakeObject(ctx context.Context) (*PooledObject, error) {
	return NewPooledObject(
			&MyPoolObject{
				s: strconv.FormatUint(atomic.AddUint64(&f.v, 1), 10),
			}),
		nil
}

func (f *MyCustomFactory) DestroyObject(ctx context.Context, object *PooledObject) error {
	// do destroy
	return nil
}

func (f *MyCustomFactory) ValidateObject(ctx context.Context, object *PooledObject) bool {
	// do validate
	return true
}

func (f *MyCustomFactory) ActivateObject(ctx context.Context, object *PooledObject) error {
	// do activate
	return nil
}

func (f *MyCustomFactory) PassivateObject(ctx context.Context, object *PooledObject) error {
	// do passivate
	return nil
}

func ExamplePooledObjectFactory() {
	ctx := context.Background()
	// MyCustomFactory is a type that implements PooledObjectFactory
	// and is defined earlier in this file (example_test.go).
	p := NewObjectPoolWithDefaultConfig(ctx, &MyCustomFactory{})

	obj1, err := p.BorrowObject(ctx)
	if err != nil {
		panic(err)
	}

	o := obj1.(*MyPoolObject)
	fmt.Println(o.s)

	err = p.ReturnObject(ctx, obj1)
	if err != nil {
		panic(err)
	}

	// Output: 1
}
