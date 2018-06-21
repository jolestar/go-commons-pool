package pool_test

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/jolestar/go-commons-pool"
)

func Example_multipleBorrowers() {
	type myPoolObject struct {
		s string
	}

	var v uint64
	factory := pool.NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			return &myPoolObject{
					s: strconv.FormatUint(atomic.AddUint64(&v, 1), 10),
				},
				nil
		})

	ctx := context.Background()
	p := pool.NewObjectPoolWithDefaultConfig(ctx, factory)

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
