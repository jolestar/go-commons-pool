package collections

import (
	"fmt"
	"github.com/jolestar/go-commons-pool/concurrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

var ONE = 1
var TWO = 2
var THREE = 3

type LinkedBlockDequeTestSuite struct {
	suite.Suite
	deque *LinkedBlockDeque
}

func (this *LinkedBlockDequeTestSuite) NoErrorWithResult(object interface{}, err error) interface{} {
	//this.NotNil(object)
	this.Nil(err)
	return object
}

func (this *LinkedBlockDequeTestSuite) ErrorWithResult(object interface{}, err error) error {
	this.Nil(object)
	this.NotNil(err)
	return err
}

func TestLinkedBlockQueueTestSuite(t *testing.T) {
	suite.Run(t, new(LinkedBlockDequeTestSuite))
}

func (this *LinkedBlockDequeTestSuite) SetupTest() {
	this.deque = NewDeque(2)
}

func (this *LinkedBlockDequeTestSuite) TestAdd() {
	this.deque = NewDeque(3)
	this.deque.Add(ONE)
	this.deque.Add(TWO)
	this.deque.Add(THREE)
	//fmt.Println(deque.Size())
	assert.Equal(this.T(), 3, this.deque.Size(), "deque size != 3")
}

func (this *LinkedBlockDequeTestSuite) TestAddFirst() {
	this.deque.AddFirst(ONE)
	this.deque.AddFirst(TWO)
	//fmt.Println(deque.Size())
	assert.Equal(this.T(), 2, this.deque.Size(), "deque size != 2")
	e := this.deque.AddFirst(THREE)
	assert.NotNil(this.T(), e, "deque can not add three element")
	assert.Equal(this.T(), TWO, this.deque.Pop())
}

func (this *LinkedBlockDequeTestSuite) TestAddLast() {
	this.deque.AddLast(ONE)
	this.deque.AddLast(TWO)
	assert.Equal(this.T(), 2, this.deque.Size())
	e := this.deque.AddLast(THREE)
	assert.NotNil(this.T(), e, "deque can not add three element")
	assert.Equal(this.T(), ONE, this.deque.Pop())
}

func (this *LinkedBlockDequeTestSuite) TestOfferFirst() {
	this.deque.OfferFirst(ONE)
	this.deque.OfferFirst(TWO)
	assert.Equal(this.T(), 2, this.deque.Size())
	this.deque.OfferFirst(nil)
	assert.Equal(this.T(), TWO, this.deque.Pop())
}

func (this *LinkedBlockDequeTestSuite) TestOfferLast() {
	this.deque.OfferLast(ONE)
	this.deque.OfferLast(TWO)
	assert.Equal(this.T(), 2, this.deque.Size())
	this.deque.OfferLast(nil)
	assert.Equal(this.T(), ONE, this.deque.Pop())
}

func (this *LinkedBlockDequeTestSuite) TestPutFirst() {
	this.deque.PutFirst(nil)
	this.deque.PutFirst(ONE)
	this.deque.PutFirst(TWO)
	assert.Equal(this.T(), 2, this.deque.Size())
	assert.Equal(this.T(), TWO, this.deque.Pop())
}

func (this *LinkedBlockDequeTestSuite) TestPutLast() {
	this.deque.PutLast(nil)
	this.deque.PutLast(ONE)
	this.deque.PutLast(TWO)
	assert.Equal(this.T(), 2, this.deque.Size())
	assert.Equal(this.T(), ONE, this.deque.Pop())
}

func (this *LinkedBlockDequeTestSuite) TestPollFirst() {
	assert.Nil(this.T(), this.deque.PollFirst())
	assert.True(this.T(), this.deque.OfferFirst(ONE))
	assert.True(this.T(), this.deque.OfferFirst(TWO))
	assert.Equal(this.T(), TWO, this.deque.PollFirst())
}

func (this *LinkedBlockDequeTestSuite) TestPollLast() {
	assert.Nil(this.T(), this.deque.PollLast())
	assert.True(this.T(), this.deque.OfferFirst(ONE))
	assert.True(this.T(), this.deque.OfferFirst(TWO))
	assert.Equal(this.T(), ONE, this.deque.PollLast())
}

func (this *LinkedBlockDequeTestSuite) TestTakeFirst() {
	assert.True(this.T(), this.deque.OfferFirst(ONE))
	assert.True(this.T(), this.deque.OfferFirst(TWO))
	assert.Equal(this.T(), TWO, this.NoErrorWithResult(this.deque.TakeFirst()))
}

func (this *LinkedBlockDequeTestSuite) TestTakeLast() {
	assert.True(this.T(), this.deque.OfferFirst(ONE))
	assert.True(this.T(), this.deque.OfferFirst(TWO))
	assert.Equal(this.T(), ONE, this.NoErrorWithResult(this.deque.TakeLast()))
}

func (this *LinkedBlockDequeTestSuite) TestRemoveLastOccurence() {
	assert.False(this.T(), this.deque.removeLastOccurrence(nil))
	assert.False(this.T(), this.deque.removeLastOccurrence(ONE))
	this.deque.Add(ONE)
	this.deque.Add(ONE)
	fmt.Println(this.deque.Size())
	assert.True(this.T(), this.deque.removeLastOccurrence(ONE))
	fmt.Println(this.deque.Size())
	assert.True(this.T(), this.deque.Size() == 1)
}

func (this *LinkedBlockDequeTestSuite) TestPollFirstWithTimeout() {
	assert.Nil(this.T(), this.deque.PollFirst())
	assert.Nil(this.T(), this.NoErrorWithResult(this.deque.PollFirstWithTimeout(50*time.Millisecond)))
}

func (this *LinkedBlockDequeTestSuite) TestPollLastWithTimeout() {
	assert.Nil(this.T(), this.deque.PollLast())
	assert.Nil(this.T(), this.NoErrorWithResult(this.deque.PollLastWithTimeout(50*time.Millisecond)))
}

func (this *LinkedBlockDequeTestSuite) TestInterrupt() {
	wait := sync.WaitGroup{}
	wait.Add(2)
	go func() {
		for i := 0; i < 2; i++ {
			time.Sleep(time.Duration(1000) * time.Millisecond)
			this.deque.InterruptTakeWaiters()
			fmt.Println("TestInterrupt this.deque.InterruptTakeWaiters")
			wait.Done()
		}
	}()
	for i := 0; i < 2; i++ {
		_, e := this.deque.TakeFirst()
		_, ok := e.(*InterruptedErr)
		this.True(ok, "expect InterruptedErr bug get %v", reflect.TypeOf(e))
	}
	wait.Wait()
}

func (this *LinkedBlockDequeTestSuite) TestIterator() {
	this.deque.Add(ONE)
	this.deque.Add(TWO)
	iterator := this.deque.Iterator()
	var list []int
	for iterator.HasNext() {
		item := iterator.Next().(int)
		list = append(list, item)
	}
	//fmt.Println("list:",list)
	assert.True(this.T(), reflect.DeepEqual(list, []int{ONE, TWO}))
}

func (this *LinkedBlockDequeTestSuite) TestDescendingIterator() {
	this.deque.Add(ONE)
	this.deque.Add(TWO)
	iterator := this.deque.DescendingIterator()
	var list []int
	for iterator.HasNext() {
		item := iterator.Next().(int)
		list = append(list, item)
	}
	//fmt.Println("list:",list)
	assert.True(this.T(), reflect.DeepEqual(list, []int{TWO, ONE}))
}

func (this *LinkedBlockDequeTestSuite) TestIteratorRemove() {
	count := 100
	this.deque = NewDeque(count)

	for i := 0; i < count; i++ {
		this.deque.Add(i)
	}
	assert.Equal(this.T(), count, this.deque.Size())
	startWait := sync.WaitGroup{}
	startWait.Add(1)

	endWait := sync.WaitGroup{}
	endWait.Add(count + 1)

	counts := make(map[int]concurrent.AtomicInteger, count)
	var hasErr concurrent.AtomicInteger = concurrent.AtomicInteger(0)
	for i := 0; i < count; i++ {
		go func(idx int) {
			startWait.Wait()
			iterator := this.deque.Iterator()
			for iterator.HasNext() {
				item := iterator.Next()
				if item == nil {
					hasErr.IncrementAndGet()
				} else {
					c := counts[idx]
					c.IncrementAndGet()
				}
			}
			endWait.Done()
		}(i)
	}
	go func() {
		startWait.Wait()
		iterator := this.deque.Iterator()
		c := 0
		for iterator.HasNext() {
			iterator.Next()
			if c%2 == 1 {
				iterator.Remove()
			}
			c = c + 1
		}
		endWait.Done()
	}()
	startWait.Done()
	endWait.Wait()
	iterator := this.deque.Iterator()
	var list []int
	for iterator.HasNext() {
		item := iterator.Next().(int)
		list = append(list, item)
	}
	//fmt.Println("list:",list)
	//fmt.Println("counts:", counts)
	assert.Equal(this.T(), count/2, this.deque.Size())
	assert.Equal(this.T(), count/2, len(list))
	assert.Equal(this.T(), int32(0), hasErr.Get())
}

func (this *LinkedBlockDequeTestSuite) TestQueueLock() {
	this.deque = NewDeque(1)
	ch := make(chan int)
	go func() {
		ch <- this.NoErrorWithResult(this.deque.TakeFirst()).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	//time.Sleep(time.Duration(1)*time.Second)
	go func() {
		this.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	this.Equal(1, val)
}

func (this *LinkedBlockDequeTestSuite) TestQueueConcurrent() {
	this.deque = NewDeque(10)
	count := 100
	ch := make(chan int, count)
	for i := 0; i < count; i++ {
		go func() {
			ch <- this.NoErrorWithResult(this.deque.TakeFirst()).(int)
		}()
	}
	sleep(100)
	for i := 0; i < count; i++ {
		go func(val int) {
			this.deque.AddFirst(val)
		}(i)
	}
	values := make([]int, count)
	valueset := make(map[int]int)
	for i := 0; i < count; i++ {
		val := <-ch
		values[i] = val
		valueset[val] = val
	}
	fmt.Println("TestQueueConcurrent", values)
	this.Equal(count, len(valueset))
	close(ch)
	//this.Equal(1, val)
}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func sleep(millisecond int) {
	time.Sleep(time.Duration(millisecond) * time.Millisecond)
}

func (this *LinkedBlockDequeTestSuite) TestQueueConcurrentTimeout() {
	this.deque = NewDeque(10)
	count := 20
	ch := make(chan int, count/2)
	timeoutChan := make(chan int, count/2)
	timeout := 1000
	for i := 0; i < count; i++ {
		go func() {
			startWait := currentTimeMillis()
			r := this.NoErrorWithResult(this.deque.PollFirstWithTimeout(time.Duration(timeout) * time.Millisecond))
			endWait := currentTimeMillis()
			//timeout
			if r == nil {
				timeoutChan <- int((endWait - startWait))
			} else {
				ch <- r.(int)
			}
		}()
	}
	for i := 0; i < count/2; i++ {
		go func(val int) {
			sleep(int(rand.Int31n(int32(timeout))))
			this.deque.AddFirst(val)
		}(i)
	}
	for i := 0; i < count/2; i++ {
		t := <-timeoutChan
		//fmt.Println(t)
		if t < timeout {
			this.Fail(fmt.Sprintf("%v timeout %v < 1000", t, i))
		}
	}
	close(timeoutChan)
	valueset := make(map[int]int)
	for i := 0; i < count/2; i++ {
		val := <-ch
		valueset[val] = val
	}
	close(ch)
	this.Equal(count/2, len(valueset))
}

func (this *LinkedBlockDequeTestSuite) TestQueuePutAndPullTimeout() {
	this.deque = NewDeque(1)
	ch := make(chan int)
	timeout := time.Duration(500) * time.Millisecond
	go func() {
		ch <- this.NoErrorWithResult(this.deque.PollFirstWithTimeout(timeout)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(time.Duration(100) * time.Millisecond)
	go func() {
		this.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	this.Equal(1, val)
}

func (this *LinkedBlockDequeTestSuite) TestHasTakeWaitersWithTimeout() {
	this.deque = NewDeque(1)
	this.False(this.deque.HasTakeWaiters())
	timeout := time.Duration(500) * time.Millisecond
	ch := make(chan int)
	go func() {
		ch <- this.NoErrorWithResult(this.deque.PollFirstWithTimeout(timeout)).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	this.True(this.deque.HasTakeWaiters())
	go func() {
		this.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	this.Equal(1, val)
	this.False(this.deque.HasTakeWaiters())
}

func (this *LinkedBlockDequeTestSuite) TestHasTakeWaiters() {
	this.deque = NewDeque(1)
	this.False(this.deque.HasTakeWaiters())
	ch := make(chan int)
	go func() {
		ch <- this.NoErrorWithResult(this.deque.TakeFirst()).(int)
		fmt.Printf("TestQueueLock take finish.\n")
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	this.True(this.deque.HasTakeWaiters())
	go func() {
		this.deque.PutFirst(1)
		fmt.Printf("TestQueueLock put finish.\n")
	}()
	val := <-ch
	close(ch)
	this.Equal(1, val)
	this.False(this.deque.HasTakeWaiters())
}
