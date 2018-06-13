package pool

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jolestar/go-commons-pool/concurrent"
)

type BenchObject struct {
	Num int32
}

func BenchmarkPoolBorrowReturn(b *testing.B) {
	ctx := context.Background()
	pool := NewObjectPoolWithDefaultConfig(ctx, NewPooledObjectFactorySimple(func(context.Context) (interface{}, error) {
		return &BenchObject{Num: rand.Int31()}, nil
	}))
	defer pool.Close(ctx)
	for i := 0; i < b.N; i++ {
		o, err := pool.BorrowObject(ctx)
		if err != nil {
			b.Fail()
		}
		err = pool.ReturnObject(ctx, o)
		if err != nil {
			b.Fail()
		}
	}
}

func BenchmarkPoolBorrowReturnParallel(b *testing.B) {
	ctx := context.Background()
	pool := NewObjectPoolWithDefaultConfig(ctx, NewPooledObjectFactorySimple(func(context.Context) (interface{}, error) {
		return &BenchObject{}, nil
	}))
	pool.Config.MaxTotal = 100
	defer pool.Close(ctx)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			o, err := pool.BorrowObject(ctx)
			//fmt.Println("borrow:",reflect.ValueOf(o).Pointer())
			if err != nil {
				fmt.Println(err)
				b.Fail()
			}
			err = pool.ReturnObject(ctx, o)
			//fmt.Println("return:",reflect.ValueOf(o).Pointer())
			if err != nil {
				fmt.Println(err)
				b.Fail()
			}
		}
	})
}

type SleepingObjectFactory struct {
	counter concurrent.AtomicInteger
}

func NewSleepingObjectFactory() *SleepingObjectFactory {
	return &SleepingObjectFactory{counter: concurrent.AtomicInteger(0)}
}

func (f *SleepingObjectFactory) MakeObject(context.Context) (*PooledObject, error) {
	if debugTest {
		fmt.Println("factory MakeObject", f.counter.Get())
	}
	sleep(500)
	return NewPooledObject(getNthObject(int(f.counter.Get()))), nil
}

func (f *SleepingObjectFactory) DestroyObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory DestroyObject", object)
	}
	sleep(250)
	return nil
}

func (f *SleepingObjectFactory) ValidateObject(ctx context.Context, object *PooledObject) bool {
	if debugTest {
		fmt.Println("factory ValidateObject", object)
	}
	sleep(30)
	return true
}

func (f *SleepingObjectFactory) ActivateObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory ActivateObject", object)
		defer fmt.Println("factory ActivateObject end")
	}
	sleep(10)
	return nil
}

func (f *SleepingObjectFactory) PassivateObject(ctx context.Context, object *PooledObject) error {
	if debugTest {
		fmt.Println("factory PassivateObject", object)
	}
	sleep(10)
	return nil
}

type TaskStats struct {
	waiting         int
	complete        int
	totalBorrowTime time.Duration
	totalReturnTime time.Duration
	nrSamples       int
}

func runOnce(ctx context.Context, pool *ObjectPool, taskStats *TaskStats) (time.Duration, time.Duration) {
	taskStats.waiting++
	if debugTest {
		fmt.Println("   waiting: ", taskStats.waiting, "   complete: ", taskStats.complete)
	}
	begin := time.Now()
	o, _ := pool.BorrowObject(ctx)
	borrowTime := time.Since(begin)
	taskStats.waiting--

	if debugTest {
		fmt.Println(
			"    waiting: ", taskStats.waiting,
			"   complete: ", taskStats.complete)
	}

	begin = time.Now()
	pool.ReturnObject(ctx, o)
	returnTime := time.Since(begin)
	taskStats.complete++

	return borrowTime, returnTime
}

func prefTask(ctx context.Context, pool *ObjectPool, nrIterations int) chan TaskStats {
	ch := make(chan TaskStats, 1)
	go func() {
		taskStats := TaskStats{}
		runOnce(ctx, pool, &taskStats) // warmup
		for i := 0; i < nrIterations; i++ {
			borrowTime, returnTime := runOnce(ctx, pool, &taskStats)
			taskStats.totalBorrowTime += borrowTime
			taskStats.totalReturnTime += returnTime
			taskStats.nrSamples++
			if debugTest {
				fmt.Println("result ", taskStats.nrSamples, "borrow time: ", borrowTime, "\t"+
					"return time: ", returnTime, "\t", "waiting: ",
					taskStats.waiting, "\t", "complete: ",
					taskStats.complete)
			}
		}
		ch <- taskStats
	}()
	return ch
}

func perfRun(ctx context.Context, iterations int, nrThreads int, maxTotal int, maxIdle int) {
	factory := NewSleepingObjectFactory()

	pool := NewObjectPoolWithDefaultConfig(ctx, factory)
	pool.Config.MaxTotal = maxTotal
	pool.Config.MaxIdle = maxIdle
	pool.Config.TestOnBorrow = true
	chs := make([]chan TaskStats, nrThreads)
	for i := 0; i < nrThreads; i++ {
		chs[i] = prefTask(ctx, pool, iterations)
	}

	aggregate := TaskStats{}
	for i := 0; i < nrThreads; i++ {
		taskStats := <-chs[i]
		close(chs[i])
		aggregate.complete += taskStats.complete
		aggregate.nrSamples += taskStats.nrSamples
		aggregate.totalBorrowTime += taskStats.totalBorrowTime
		aggregate.totalReturnTime += taskStats.totalReturnTime
		aggregate.waiting += taskStats.waiting
	}

	fmt.Println("-----------------------------------------")
	fmt.Println("nrIterations: ", iterations)
	fmt.Println("nrThreads: ", nrThreads)
	fmt.Println("maxTotal: ", maxTotal)
	fmt.Println("maxIdle: ", maxIdle)
	fmt.Println("nrSamples: ", aggregate.nrSamples)
	fmt.Println("totalBorrowTime: ", aggregate.totalBorrowTime)
	fmt.Println("totalReturnTime: ", aggregate.totalReturnTime)
	fmt.Println("avg BorrowTime: ",
		aggregate.totalBorrowTime/time.Duration(aggregate.nrSamples))
	fmt.Println("avg ReturnTime: ",
		aggregate.totalReturnTime/time.Duration(aggregate.nrSamples))
}

func perfMain(ctx context.Context) {
	fmt.Println("Increase threads")
	perfRun(ctx, 1, 50, 5, 5)
	perfRun(ctx, 1, 100, 5, 5)
	perfRun(ctx, 1, 200, 5, 5)
	perfRun(ctx, 1, 400, 5, 5)

	fmt.Println("Increase threads & poolsize")
	perfRun(ctx, 1, 50, 5, 5)
	perfRun(ctx, 1, 100, 10, 10)
	perfRun(ctx, 1, 200, 20, 20)
	perfRun(ctx, 1, 400, 40, 40)

	fmt.Println("Increase maxIdle")
	perfRun(ctx, 1, 400, 40, 5)
	perfRun(ctx, 1, 400, 40, 40)
}
