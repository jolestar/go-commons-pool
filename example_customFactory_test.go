package pool_test

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/jolestar/go-commons-pool"
)

type MyPoolObject struct {
	s string
}

type MyCustomFactory struct {
	v uint64
}

func (f *MyCustomFactory) MakeObject(ctx context.Context) (*pool.PooledObject, error) {
	return pool.NewPooledObject(
			&MyPoolObject{
				s: strconv.FormatUint(atomic.AddUint64(&f.v, 1), 10),
			}),
		nil
}

func (f *MyCustomFactory) DestroyObject(ctx context.Context, object *pool.PooledObject) error {
	// do destroy
	return nil
}

func (f *MyCustomFactory) ValidateObject(ctx context.Context, object *pool.PooledObject) bool {
	// do validate
	return true
}

func (f *MyCustomFactory) ActivateObject(ctx context.Context, object *pool.PooledObject) error {
	// do activate
	return nil
}

func (f *MyCustomFactory) PassivateObject(ctx context.Context, object *pool.PooledObject) error {
	// do passivate
	return nil
}

func Example_customFactory() {
	ctx := context.Background()
	p := pool.NewObjectPoolWithDefaultConfig(ctx, &MyCustomFactory{})

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
