// Package queue provides a lock-free queue and two-Lock concurrent queue which use the algorithm proposed by Michael and Scott.
// https://doi.org/10.1145/248052.248106.
//
// see pseudocode at https://www.cs.rochester.edu/research/synchronization/pseudocode/queues.html
// It will be refactored after go generic is released.
package queue

import (
	"sync/atomic"
	"unsafe"
)

// LKQueue is a lock-free unbounded queue.
type LKQueue struct {
	head unsafe.Pointer
	tail unsafe.Pointer
}

type node struct {
	value interface{}
	next  unsafe.Pointer
}

// NewLKQueue returns an empty queue.
func NewLKQueue() *LKQueue {
	n := unsafe.Pointer(&node{})
	return &LKQueue{head: n, tail: n}
}

// Enqueue puts the given value v at the tail of the queue.
func (q *LKQueue) Enqueue(v interface{}) {
	n := &node{value: v}
	for {
		tail := load(&q.tail)
		next := load(&tail.next)
		if tail == load(&q.tail) { // are tail and next consistent?
			if next == nil {
				if cas(&tail.next, next, n) {
					cas(&q.tail, tail, n) // Enqueue is done.  try to swing tail to the inserted node
					return
				}
			} else { // tail was not pointing to the last node
				// try to swing Tail to the next node
				cas(&q.tail, tail, next)
			}
		}
	}
}

// Dequeue removes and returns the value at the head of the queue.
// It returns nil if the queue is empty.
func (q *LKQueue) Dequeue() interface{} {
	for {
		head := load(&q.head)
		tail := load(&q.tail)
		next := load(&head.next)
		if head == load(&q.head) { // are head, tail, and next consistent?
			if head == tail { // is queue empty or tail falling behind?
				if next == nil { // is queue empty?
					return nil
				}
				// tail is falling behind.  try to advance it
				cas(&q.tail, tail, next)
			} else {
				// read value before CAS otherwise another dequeue might free the next node
				v := next.value
				if cas(&q.head, head, next) {
					return v // Dequeue is done.  return
				}
			}
		}
	}
}

func (q *LKQueue) Top() interface{} {
	head := load(&q.head)
	next := load(&head.next)
	v := next.value
	return v
}

func load(p *unsafe.Pointer) (n *node) {
	return (*node)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *node) (ok bool) {
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}
