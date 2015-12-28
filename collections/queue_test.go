package collections

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"fmt"
	"time"
	"sync"
	"reflect"
	"sync/atomic"
)

var ONE = 1
var TWO = 2
var THREE = 3

type LinkedBlockDequeTestSuite struct {
	suite.Suite
	deque *LinkedBlockDeque
}

func TestLinkedBlockQueueTestSuite(t *testing.T) {
	suite.Run(t, new(LinkedBlockDequeTestSuite))
}

func (suite *LinkedBlockDequeTestSuite) SetupTest() {
	suite.deque, _ = NewDeque(2)
}


func (suite *LinkedBlockDequeTestSuite) TestAdd() {
	suite.deque, _ = NewDeque(3)
	suite.deque.Add(ONE)
	suite.deque.Add(TWO)
	suite.deque.Add(THREE)
	//fmt.Println(deque.Size())
	assert.Equal(suite.T(), 3, suite.deque.Size(), "deque size != 3")
}


func (suite *LinkedBlockDequeTestSuite) TestAddFirst() {
	suite.deque.AddFirst(ONE)
	suite.deque.AddFirst(TWO)
	//fmt.Println(deque.Size())
	assert.Equal(suite.T(), 2, suite.deque.Size(), "deque size != 2")
	e := suite.deque.AddFirst(THREE)
	assert.NotNil(suite.T(), e, "deque can not add three element")
	assert.Equal(suite.T(), TWO, suite.deque.Pop())
}

func (suite *LinkedBlockDequeTestSuite) TestAddLast() {
	suite.deque.AddLast(ONE)
	suite.deque.AddLast(TWO)
	assert.Equal(suite.T(), 2, suite.deque.Size())
	e := suite.deque.AddLast(THREE)
	assert.NotNil(suite.T(), e, "deque can not add three element")
	assert.Equal(suite.T(), ONE, suite.deque.Pop())
}

func (suite *LinkedBlockDequeTestSuite) TestOfferFirst() {
	suite.deque.OfferFirst(ONE)
	suite.deque.OfferFirst(TWO)
	assert.Equal(suite.T(), 2, suite.deque.Size())
	suite.deque.OfferFirst(nil)
	assert.Equal(suite.T(), TWO, suite.deque.Pop())
}

func (suite *LinkedBlockDequeTestSuite) TestOfferLast() {
	suite.deque.OfferLast(ONE)
	suite.deque.OfferLast(TWO)
	assert.Equal(suite.T(), 2, suite.deque.Size())
	suite.deque.OfferLast(nil)
	assert.Equal(suite.T(), ONE, suite.deque.Pop())
}

func (suite *LinkedBlockDequeTestSuite) TestPutFirst() {
	suite.deque.PutFirst(nil)
	suite.deque.PutFirst(ONE)
	suite.deque.PutFirst(TWO)
	assert.Equal(suite.T(), 2, suite.deque.Size())
	assert.Equal(suite.T(), TWO, suite.deque.Pop())
}

func (suite *LinkedBlockDequeTestSuite) TestPutLast() {
	suite.deque.PutLast(nil)
	suite.deque.PutLast(ONE)
	suite.deque.PutLast(TWO)
	assert.Equal(suite.T(), 2, suite.deque.Size())
	assert.Equal(suite.T(), ONE, suite.deque.Pop())
}

func (suite *LinkedBlockDequeTestSuite) TestPollFirst() {
	assert.Nil(suite.T(), suite.deque.PollFirst())
	assert.True(suite.T(), suite.deque.OfferFirst(ONE))
	assert.True(suite.T(), suite.deque.OfferFirst(TWO))
	assert.Equal(suite.T(), TWO, suite.deque.PollFirst())
}

func (suite *LinkedBlockDequeTestSuite) TestPollLast() {
	assert.Nil(suite.T(), suite.deque.PollLast())
	assert.True(suite.T(), suite.deque.OfferFirst(ONE))
	assert.True(suite.T(), suite.deque.OfferFirst(TWO))
	assert.Equal(suite.T(), ONE, suite.deque.PollLast())
}

func (suite *LinkedBlockDequeTestSuite) TestTakeFirst() {
	assert.True(suite.T(), suite.deque.OfferFirst(ONE))
	assert.True(suite.T(), suite.deque.OfferFirst(TWO))
	assert.Equal(suite.T(), TWO, suite.deque.TakeFirst())
}

func (suite *LinkedBlockDequeTestSuite) TestTakeLast() {
	assert.True(suite.T(), suite.deque.OfferFirst(ONE))
	assert.True(suite.T(), suite.deque.OfferFirst(TWO))
	assert.Equal(suite.T(), ONE, suite.deque.TakeLast())
}

func (suite *LinkedBlockDequeTestSuite) TestRemoveLastOccurence()  {
	assert.False(suite.T(),suite.deque.removeLastOccurrence(nil))
	assert.False(suite.T(),suite.deque.removeLastOccurrence(ONE))
	suite.deque.Add(ONE)
	suite.deque.Add(ONE)
	fmt.Println(suite.deque.Size())
	assert.True(suite.T(), suite.deque.removeLastOccurrence(ONE))
	fmt.Println(suite.deque.Size())
	assert.True(suite.T(), suite.deque.Size() == 1)
}

func (suite *LinkedBlockDequeTestSuite) TestPollFirstWithTimeout() {
	assert.Nil(suite.T(), suite.deque.PollFirst())
	assert.Nil(suite.T(), suite.deque.PollFirstWithTimeout(50*time.Millisecond))
}

func (suite *LinkedBlockDequeTestSuite) TestPollLastWithTimeout() {
	assert.Nil(suite.T(), suite.deque.PollLast())
	assert.Nil(suite.T(), suite.deque.PollLastWithTimeout(50*time.Millisecond))
}

func (suite *LinkedBlockDequeTestSuite) TestInterrupt() {
	t := time.Tick(time.Duration(1000*time.Millisecond))
	go func() {
		for _ = range t{
			suite.deque.InterruptTakeWaiters()
		}
	}()
	assert.Nil(suite.T(), suite.deque.TakeFirst())
}

func (suite *LinkedBlockDequeTestSuite) TestIterator() {
	suite.deque.Add(ONE)
	suite.deque.Add(TWO)
	iterator := suite.deque.Iterator()
	var list []int
	for iterator.HasNext(){
		item := iterator.Next().(int)
		list = append(list,item)
	}
	//fmt.Println("list:",list)
	assert.True(suite.T(), reflect.DeepEqual(list,[]int{ONE,TWO}))
}

func (suite *LinkedBlockDequeTestSuite) TestDescendingIterator() {
	suite.deque.Add(ONE)
	suite.deque.Add(TWO)
	iterator := suite.deque.DescendingIterator()
	var list []int
	for iterator.HasNext(){
		item := iterator.Next().(int)
		list = append(list,item)
	}
	//fmt.Println("list:",list)
	assert.True(suite.T(), reflect.DeepEqual(list,[]int{TWO, ONE}))
}

func (suite *LinkedBlockDequeTestSuite) TestIteratorRemove() {
	count := 100;
	suite.deque,_ = NewDeque(count)

	for i:=0;i < count;i++{
		suite.deque.Add(i)
	}
	assert.Equal(suite.T(),count, suite.deque.Size())
	startWait := sync.WaitGroup{}
	startWait.Add(1)

	endWait := sync.WaitGroup{}
	endWait.Add(count +1)

	counts := make(map[int]int32, count)
	var hasErr int32 = 0
	for i :=0;i < count;i++{
		go func(idx int) {
			startWait.Wait()
			iterator := suite.deque.Iterator()
			for iterator.HasNext(){
				item := iterator.Next()
				if(item == nil){
					hasErr = atomic.AddInt32(&hasErr, int32(1))
				}else{
					c := counts[idx]
					counts[idx] = atomic.AddInt32(&c,int32(1))
				}
			}
			endWait.Done()
		}(i)
	}
	go func() {
		startWait.Wait()
		iterator := suite.deque.Iterator()
		c :=0
		for iterator.HasNext(){
			iterator.Next()
			if(c %2 == 1){
				iterator.Remove()
			}
			c = c+1
		}
		endWait.Done()
	}()
	startWait.Done()
	endWait.Wait()
	iterator := suite.deque.Iterator()
	var list []int
	for iterator.HasNext(){
		item := iterator.Next().(int)
		list = append(list,item)
	}
	//fmt.Println("list:",list)
	//fmt.Println("counts:", counts)
	assert.Equal(suite.T(),count/2, suite.deque.Size())
	assert.Equal(suite.T(),count/2, len(list))
	assert.Equal(suite.T(), int32(0), hasErr)
}

func TestChannelBlock(t *testing.T)  {
	ch := make(chan int)
	wait := new(sync.WaitGroup)
	wait.Add(1)
	go func() {
		<- ch
		//fmt.Println(v)
		wait.Done()
	}()
	time.Sleep(time.Duration(500*time.Millisecond))
	close(ch)
	wait.Wait()
}
