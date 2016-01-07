package collections

import (
	"errors"
	"fmt"
	"github.com/jolestar/go-commons-pool/concurrent"
	"sync"
	"time"
)

const (
	debug_queue = false
)

type InterruptedErr struct {
}

func NewInterruptedErr() *InterruptedErr {
	return &InterruptedErr{}
}

func (this *InterruptedErr) Error() string {
	return "Interrupted"
}

type Node struct {

	//The item, or nil if this node has been removed.
	item interface{}

	//One of:
	//- the real predecessor Node
	//- this Node, meaning the predecessor is tail
	//- nil, meaning there is no predecessor
	prev *Node

	//One of:
	//- the real successor Node
	//- this Node, meaning the successor is head
	//- null, meaning there is no successor
	next *Node
}

func NewNode(item interface{}, prev *Node, next *Node) *Node {
	return &Node{item: item, prev: prev, next: next}
}

type LinkedBlockingDeque struct {

	//Pointer to first node.
	//Invariant: (first == nil && last == nil) ||
	//            (first.prev == nil && first.item != nil)
	first *Node

	// Pointer to last node.
	// Invariant: (first == null && last == null) ||
	//            (last.next == null && last.item != null)
	last *Node

	// Number of items in the deque
	count int

	// Maximum number of items in the deque
	capacity int

	// Main lock guarding all access
	lock *sync.Mutex

	// Condition for waiting takes
	notEmpty *concurrent.TimeoutCond

	//Condition for waiting puts
	notFull *concurrent.TimeoutCond
}

func NewDeque(capacity int) *LinkedBlockingDeque {
	if capacity < 0 {
		panic(errors.New("capacity must > 0"))
	}
	lock := new(sync.Mutex)
	return &LinkedBlockingDeque{capacity: capacity, lock: lock, notEmpty: concurrent.NewTimeoutCond(lock), notFull: concurrent.NewTimeoutCond(lock)}
}

//Links provided element as first element, or returns false if full.
//return true if successful, otherwise false
func (this *LinkedBlockingDeque) linkFirst(e interface{}) bool {
	if this.count >= this.capacity {
		return false
	}
	f := this.first
	x := NewNode(e, nil, f)
	this.first = x
	if this.last == nil {
		this.last = x
	} else {
		f.prev = x
	}
	this.count = this.count + 1
	this.notEmpty.Signal()
	return true
}

//Links provided element as last element, or returns false if full.
//return true} if successful, otherwise false
func (this *LinkedBlockingDeque) linkLast(e interface{}) bool {
	// assert lock.isHeldByCurrentThread();
	if this.count >= this.capacity {
		return false
	}
	l := this.last
	x := NewNode(e, l, nil)
	this.last = x
	if this.first == nil {
		this.first = x
	} else {
		l.next = x
	}
	this.count = this.count + 1
	this.notEmpty.Signal()
	return true
}

//Removes and returns the first element, or nil if empty.
func (this *LinkedBlockingDeque) unlinkFirst() interface{} {
	// assert lock.isHeldByCurrentThread();
	f := this.first
	if f == nil {
		if debug_queue {
			fmt.Println("unlinkFirst first is nil")
		}
		return nil
	}
	n := f.next
	item := f.item
	f.item = nil
	f.next = f //help GC
	this.first = n
	if n == nil {
		this.last = nil
	} else {
		n.prev = nil
	}
	this.count = this.count - 1
	this.notFull.Signal()
	if debug_queue {
		fmt.Println("unlinkFirst", item)
	}
	return item
}

//Removes and returns the last element, or nil if empty.
func (this *LinkedBlockingDeque) unlinkLast() interface{} {
	l := this.last
	if l == nil {
		return nil
	}
	p := l.prev
	item := l.item
	l.item = nil
	l.prev = l // help GC
	this.last = p
	if p == nil {
		this.first = nil
	} else {
		p.next = nil
	}
	this.count = this.count - 1
	this.notFull.Signal()
	return item
}

//Unlinks the provided node.
func (this *LinkedBlockingDeque) unlink(x *Node) {
	// assert lock.isHeldByCurrentThread();
	if debug_queue {
		fmt.Println("unlink")
	}
	p := x.prev
	n := x.next
	if p == nil {
		this.unlinkFirst()
	} else if n == nil {
		this.unlinkLast()
	} else {
		p.next = n
		n.prev = p
		x.item = nil
		// Don't mess with x's links.  They may still be in use by
		// an iterator.
		this.count = this.count - 1
		this.notFull.Signal()
	}
}

//Inserts the specified element at the front of this deque if it is
//possible to do so immediately without violating capacity restrictions,
//return error if no space is currently available.
func (this *LinkedBlockingDeque) AddFirst(e interface{}) error {
	if e == nil {
		return errors.New("e is nil")
	}
	if !this.OfferFirst(e) {
		return errors.New("Deque full")
	}
	return nil
}

//Inserts the specified element at the end of this deque if it is
//possible to do so immediately without violating capacity restrictions,
//return error if no space is currently available.
func (this *LinkedBlockingDeque) AddLast(e interface{}) error {
	if e == nil {
		return errors.New("e is nil")
	}
	if !this.OfferLast(e) {
		return errors.New("Deque full")
	}
	return nil
}

//Inserts the specified element at the front of this deque unless it would
//violate capacity restrictions.
//return if the element was added to this deque
func (this *LinkedBlockingDeque) OfferFirst(e interface{}) bool {
	if e == nil {
		return false
	}
	this.lock.Lock()
	result := this.linkFirst(e)
	this.lock.Unlock()
	return result
}

//Inserts the specified element at the end of this deque unless it would
//violate capacity restrictions.
//return if the element was added to this deque
func (this *LinkedBlockingDeque) OfferLast(e interface{}) bool {
	if e == nil {
		return false
	}
	this.lock.Lock()
	result := this.linkLast(e)
	this.lock.Unlock()
	return result
}

//Links the provided element as the first in the queue, waiting until there
//is space to do so if the queue is full.
func (this *LinkedBlockingDeque) PutFirst(e interface{}) {
	if e == nil {
		return
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	for !this.linkFirst(e) {
		this.notFull.Wait()
	}
}

// Links the provided element as the last in the queue, waiting until there
// is space to do so if the queue is full.
func (this *LinkedBlockingDeque) PutLast(e interface{}) {
	if e == nil {
		return
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	for !this.linkLast(e) {
		this.notFull.Wait()
	}
}

//Retrieves and removes the first element of this deque,
//or returns nil if this deque is empty.
func (this *LinkedBlockingDeque) PollFirst() (e interface{}) {
	this.lock.Lock()
	result := this.unlinkFirst()
	this.lock.Unlock()
	return result
}

//Retrieves and removes the first element of this deque, waiting
//up to the specified wait time if necessary for an element to become available.
//return NewInterruptedErr when waiting bean interrupted
func (this *LinkedBlockingDeque) PollFirstWithTimeout(timeout time.Duration) (interface{}, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkFirst(); x == nil; x = this.unlinkFirst() {
		if timeout <= 0 {
			break
		}
		if interrupt {
			return nil, NewInterruptedErr()
		}
		timeout, interrupt = this.notEmpty.WaitWithTimeout(timeout)
	}
	return x, nil
}

//Retrieves and removes the last element of this deque,
//or returns nil if this deque is empty.
func (this *LinkedBlockingDeque) PollLast() interface{} {
	this.lock.Lock()
	result := this.unlinkLast()
	this.lock.Unlock()
	return result
}

//Retrieves and removes the last element of this deque, waiting
//up to the specified wait time if necessary for an element to become available.
//return NewInterruptedErr when waiting bean interrupted
func (this *LinkedBlockingDeque) PollLastWithTimeout(timeout time.Duration) (interface{}, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkLast(); x == nil; x = this.unlinkLast() {
		if timeout <= 0 {
			break
		}
		if interrupt {
			return nil, NewInterruptedErr()
		}
		timeout, interrupt = this.notEmpty.WaitWithTimeout(timeout)
	}
	return x, nil
}

//Unlinks the first element in the queue, waiting until there is an element
//to unlink if the queue is empty.
//return NewInterruptedErr if wait condition is interrupted
func (this *LinkedBlockingDeque) TakeFirst() (interface{}, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkFirst(); x == nil; x = this.unlinkFirst() {
		if interrupt {
			return nil, NewInterruptedErr()
		}
		interrupt = this.notEmpty.Wait()
	}
	return x, nil
}

//Unlinks the last element in the queue, waiting until there is an element
//to unlink if the queue is empty.
//return NewInterruptedErr if wait condition is interrupted
func (this *LinkedBlockingDeque) TakeLast() (interface{}, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkLast(); x == nil; x = this.unlinkLast() {
		if interrupt {
			return nil, NewInterruptedErr()
		}
		interrupt = this.notEmpty.Wait()
	}
	return x, nil
}

//Retrieves, but does not remove, the first element of this deque,
//or returns nil if this deque is empty.
func (this *LinkedBlockingDeque) PeekFirst() interface{} {
	var result interface{}
	this.lock.Lock()
	if this.first == nil {
		result = nil
	} else {
		result = this.first.item
	}
	this.lock.Unlock()
	return result
}

//Retrieves, but does not remove, the last element of this deque,
//or returns nil if this deque is empty.
func (this *LinkedBlockingDeque) PeekLast() interface{} {
	var result interface{}
	this.lock.Lock()
	if this.last == nil {
		result = nil
	} else {
		result = this.last.item
	}
	this.lock.Unlock()
	return result
}

//Removes the first occurrence of the specified element from this deque.
//If the deque does not contain the element, it is unchanged.
//More formally, removes the first element item such that
//		o == item
//(if such an element exists).
//Returns true if this deque contained the specified element
//(or equivalently, if this deque changed as a result of the call).
func (this *LinkedBlockingDeque) RemoveFirstOccurrence(item interface{}) bool {
	if item == nil {
		return false
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	for p := this.first; p != nil; p = p.next {
		if item == p.item {
			this.unlink(p)
			return true
		}
	}
	return false
}

//Removes the last occurrence of the specified element from this deque.
//If the deque does not contain the element, it is unchanged.
//More formally, removes the last element item such that
//		o == item
//(if such an element exists).
//Returns true if this deque contained the specified element
//(or equivalently, if this deque changed as a result of the call).
func (this *LinkedBlockingDeque) RemoveLastOccurrence(item interface{}) bool {
	if item == nil {
		return false
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	for p := this.last; p != nil; p = p.prev {
		if item == p.item {
			this.unlink(p)
			return true
		}
	}
	return false
}

//Interrupts the goroutine currently waiting to take an object from the pool.
func (this *LinkedBlockingDeque) InterruptTakeWaiters() {
	if debug_queue {
		fmt.Println("InterruptTakeWaiters")
	}
	this.notEmpty.Interrupt()
}

//Returns true if there are threads waiting to take instances from this deque.
//See disclaimer on accuracy in  TimeoutCond.HasWaiters()
func (this *LinkedBlockingDeque) HasTakeWaiters() bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.notEmpty.HasWaiters()
}

//Returns an slice containing all of the elements in this deque, in
//proper sequence (from first to last element).
func (this *LinkedBlockingDeque) ToSlice() []interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	a := make([]interface{}, this.count)
	for p, k := this.first, 0; p != nil; p, k = p.next, k+1 {
		a[k] = p.item
	}
	return a
}

func (this *LinkedBlockingDeque) Size() int {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.size()
}

func (this *LinkedBlockingDeque) size() int {
	return this.count
}

func (this *LinkedBlockingDeque) Iterator() *LinkedBlockDequeIterator {
	return NewIterator(this, false)
}

func (this *LinkedBlockingDeque) DescendingIterator() *LinkedBlockDequeIterator {
	return NewIterator(this, true)
}

func NewIterator(deque *LinkedBlockingDeque, desc bool) *LinkedBlockDequeIterator {
	deque.lock.Lock()
	defer deque.lock.Unlock()
	iterator := LinkedBlockDequeIterator{deque: deque, desc: desc}
	iterator.next = iterator.firstNode()
	if iterator.next == nil {
		iterator.nextItem = nil
	} else {
		iterator.nextItem = iterator.next.item
	}
	return &iterator
}

type LinkedBlockDequeIterator struct {
	deque    *LinkedBlockingDeque
	next     *Node
	nextItem interface{}
	lastRet  *Node
	desc     bool
}

func (this *LinkedBlockDequeIterator) firstNode() *Node {
	if !this.desc {
		return this.deque.first
	} else {
		return this.deque.last
	}
}

func (this *LinkedBlockDequeIterator) nextNode(node *Node) *Node {
	if !this.desc {
		return node.next
	} else {
		return node.prev
	}
}

func (this *LinkedBlockDequeIterator) HasNext() bool {
	return this.next != nil
}

func (this *LinkedBlockDequeIterator) Next() interface{} {
	if this.next == nil {
		//TODO error or nil ?
		//panic(errors.New("NoSuchElement"))
		return nil
	}
	this.lastRet = this.next
	x := this.nextItem
	this.advance()
	return x
}

func (this *LinkedBlockDequeIterator) advance() {
	lock := this.deque.lock
	lock.Lock()
	defer lock.Unlock()
	this.next = this.succ(this.next)
	if this.next == nil {
		this.nextItem = nil
	} else {
		this.nextItem = this.next.item
	}
}

func (this *LinkedBlockDequeIterator) succ(n *Node) *Node {
	for {
		s := this.nextNode(n)
		if s == nil {
			return nil
		} else if s.item != nil {
			return s
		} else if s == n {
			return this.firstNode()
		} else {
			n = s
		}
	}
}

func (this *LinkedBlockDequeIterator) Remove() {
	n := this.lastRet
	if n == nil {
		panic(errors.New("IllegalStateException"))
	}
	this.lastRet = nil
	lock := this.deque.lock
	lock.Lock()
	if n.item != nil {
		this.deque.unlink(n)
	}
	lock.Unlock()
}
