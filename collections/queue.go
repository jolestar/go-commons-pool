package collections

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

//const (
//
//)
//
//type Error struct {
//	msg string
//
//}
//
//func (this Error) Error() string {
//	return this.msg
//}

const (
	debug_queue = true
)

type Node struct {

	/**
	 * The item, or nil if this node has been removed.
	 */
	item interface{}

	/**
	 * One of:
	 * - the real predecessor Node
	 * - this Node, meaning the predecessor is tail
	 * - nil, meaning there is no predecessor
	 */
	prev *Node

	/**
	 * One of:
	 * - the real successor Node
	 * - this Node, meaning the successor is head
	 * - null, meaning there is no successor
	 */
	next *Node
}

func NewNode(item interface{}, prev *Node, next *Node) *Node {
	return &Node{item: item, prev: prev, next: next}
}

type LinkedBlockDeque struct {
	/**
	 * Pointer to first node.
	 * Invariant: (first == nil && last == nil) ||
	 *            (first.prev == nil && first.item != nil)
	 */
	first *Node

	/**
	 * Pointer to last node.
	 * Invariant: (first == null && last == null) ||
	 *            (last.next == null && last.item != null)
	 */
	last *Node

	/** Number of items in the deque */
	count int

	/** Maximum number of items in the deque */
	capacity int

	/** Main lock guarding all access */
	lock *sync.Mutex

	/** Condition for waiting takes */
	notEmpty *TimeoutCond

	/** Condition for waiting puts */
	notFull *TimeoutCond
}

func NewDeque(capacity int) *LinkedBlockDeque {
	if capacity < 0 {
		panic(errors.New("capacity must > 0"))
	}
	lock := new(sync.Mutex)
	return &LinkedBlockDeque{capacity: capacity, lock: lock, notEmpty: NewTimeoutCond(lock), notFull: NewTimeoutCond(lock)}
}

/**
* Links provided element as first element, or returns false if full.
*
* @param e The element to link as the first element.
*
* @return {@code true} if successful, otherwise {@code false}
 */
func (this *LinkedBlockDeque) linkFirst(e interface{}) bool {
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

/**
* Links provided element as last element, or returns false if full.
*
* @param e The element to link as the last element.
*
* @return {@code true} if successful, otherwise {@code false}
 */
func (this *LinkedBlockDeque) linkLast(e interface{}) bool {
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

/**
* Removes and returns the first element, or nil if empty.
*
 */
func (this *LinkedBlockDeque) unlinkFirst() interface{} {
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

/**
 * Removes and returns the last element, or nil if empty.
 *
 */
func (this *LinkedBlockDeque) unlinkLast() interface{} {
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

/**
 * Unlinks the provided node.
 *
 * @param x The node to unlink
 */
func (this *LinkedBlockDeque) unlink(x *Node) {
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

func (this *LinkedBlockDeque) Add(e interface{}) error {
	return this.AddLast(e)
}

func (this *LinkedBlockDeque) AddFirst(e interface{}) error {
	if e == nil {
		return errors.New("e is nil")
	}
	if !this.OfferFirst(e) {
		return errors.New("Deque full")
	}
	return nil
}

func (this *LinkedBlockDeque) AddLast(e interface{}) error {
	if e == nil {
		return errors.New("e is nil")
	}
	if !this.OfferLast(e) {
		return errors.New("Deque full")
	}
	return nil
}

func (this *LinkedBlockDeque) OfferFirst(e interface{}) bool {
	if e == nil {
		return false
	}
	this.lock.Lock()
	result := this.linkFirst(e)
	this.lock.Unlock()
	return result
}

func (this *LinkedBlockDeque) OfferLast(e interface{}) bool {
	if e == nil {
		return false
	}
	this.lock.Lock()
	result := this.linkLast(e)
	this.lock.Unlock()
	return result
}

/**
 * Links the provided element as the first in the queue, waiting until there
 * is space to do so if the queue is full.
 *
 * @param e element to link
 *
 * @throws NullPointerException
 * @throws InterruptedException
 */
func (this *LinkedBlockDeque) PutFirst(e interface{}) {
	if e == nil {
		return
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	for !this.linkFirst(e) {
		this.notFull.Wait()
	}
}

/**
 * Links the provided element as the last in the queue, waiting until there
 * is space to do so if the queue is full.
 */
func (this *LinkedBlockDeque) PutLast(e interface{}) {
	if e == nil {
		return
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	for !this.linkLast(e) {
		this.notFull.Wait()
	}
}

func (this *LinkedBlockDeque) PollFirst() (e interface{}) {
	this.lock.Lock()
	result := this.unlinkFirst()
	this.lock.Unlock()
	return result
}

func (this *LinkedBlockDeque) PollFirstWithTimeout(timeout time.Duration) (e interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkFirst(); x == nil;x = this.unlinkFirst() {
		if timeout <= 0 || interrupt {
			break
		}
		timeout, interrupt = this.notEmpty.WaitWithTimeout(timeout)
	}
	return x
}

func (this *LinkedBlockDeque) PollLast() interface{} {
	this.lock.Lock()
	result := this.unlinkLast()
	this.lock.Unlock()
	return result
}

func (this *LinkedBlockDeque) PollLastWithTimeout(timeout time.Duration) (e interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkLast(); x == nil; x = this.unlinkLast() {
		if timeout <= 0 || interrupt {
			break
		}
		timeout, interrupt = this.notEmpty.WaitWithTimeout(timeout)
	}
	return x
}

func (this *LinkedBlockDeque) Poll() (e interface{}) {
	return this.PollFirst()
}

//func (this *LinkedBlockDeque)  RemoveFirst()(interface{},error) {
//}

//func (this *LinkedBlockDeque)  RemoveLast()(interface{},error) {
//}

/**
 * Unlinks the first element in the queue, waiting until there is an element
 * to unlink if the queue is empty.
 *
 * @return the unlinked element
 */
func (this *LinkedBlockDeque) TakeFirst() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkFirst(); x == nil; x = this.unlinkFirst() {
		if interrupt {
			break
		}
		interrupt = this.notEmpty.Wait()
	}
	return x
}

/**
 * Unlinks the last element in the queue, waiting until there is an element
 * to unlink if the queue is empty.
 *
 * @return the unlinked element
 * @throws InterruptedException if the current thread is interrupted
 */
func (this *LinkedBlockDeque) TakeLast() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	var x interface{}
	interrupt := false
	for x = this.unlinkLast(); x == nil; x = this.unlinkLast() {
		if interrupt {
			break
		}
		interrupt = this.notEmpty.Wait()
	}
	return x
}

//func (this *LinkedBlockDeque) GetFirst() (interface{}) {
//E x = peekFirst();
//if (x == null) {
//throw new NoSuchElementException();
//}
//return x;
//}

//func (this *LinkedBlockDeque) GetLast() (interface{}) {
//E x = peekLast();
//if (x == null) {
//throw new NoSuchElementException();
//}
//return x;
//}

func (this *LinkedBlockDeque) PeekFirst() interface{} {
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

func (this *LinkedBlockDeque) PeekLast() interface{} {
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

func (this *LinkedBlockDeque) Push(e interface{}) {
	if e == nil {
		return
	}
	this.AddFirst(e)
}

func (this *LinkedBlockDeque) Pop() interface{} {
	return this.PollFirst()
}

func (this *LinkedBlockDeque) removeFirstOccurrence(item interface{}) bool {
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

func (this *LinkedBlockDeque) removeLastOccurrence(item interface{}) bool {
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

func (this *LinkedBlockDeque) Remove(item interface{}) bool {
	return this.removeFirstOccurrence(item)
}

func (this *LinkedBlockDeque) InterruptTakeWaiters() {
	if debug_queue {
		fmt.Println("InterruptTakeWaiters")
	}
	this.notEmpty.Interrupt()
}

func (this *LinkedBlockDeque) Size() int {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.size()
}

func (this *LinkedBlockDeque) size() int {
	return this.count
}

func (this *LinkedBlockDeque) Iterator() *LinkedBlockDequeIterator {
	return NewIterator(this, false)
}

func (this *LinkedBlockDeque) DescendingIterator() *LinkedBlockDequeIterator {
	return NewIterator(this, true)
}

func NewIterator(deque *LinkedBlockDeque, desc bool) *LinkedBlockDequeIterator {
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
	deque    *LinkedBlockDeque
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
