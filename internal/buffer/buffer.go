package buffer

import "sync/atomic"

type Frame struct {
	Data      []byte
	Timestamp int64
}

type RingBuffer struct {
	_      [64]byte
	head   atomic.Uint64
	_      [56]byte
	tail   atomic.Uint64
	_      [56]byte
	buffer []Frame
	mask   uint64
}

func New(size int) *RingBuffer {
	c := uint64(1)
	for c < uint64(size) {
		c <<= 1
	}
	return &RingBuffer{
		buffer: make([]Frame, c),
		mask:   c - 1,
	}
}

func (rb *RingBuffer) Put(f Frame) bool {
	h := rb.head.Load()
	if (h+1)&rb.mask == rb.tail.Load()&rb.mask {
		return false
	}
	for !rb.head.CompareAndSwap(h, h+1) {
		h = rb.head.Load()
		if (h+1)&rb.mask == rb.tail.Load()&rb.mask {
			return false
		}
	}
	rb.buffer[h&rb.mask] = f
	return true
}

func (rb *RingBuffer) Get() (Frame, bool) {
	t := rb.tail.Load()
	if t == rb.head.Load() {
		return Frame{}, false
	}
	f := rb.buffer[t&rb.mask]
	rb.tail.Add(1)
	return f, true
}

func (rb *RingBuffer) Len() int      { return int(rb.head.Load() - rb.tail.Load()) }
func (rb *RingBuffer) IsFull() bool  { return (rb.head.Load()+1)&rb.mask == rb.tail.Load()&rb.mask }
func (rb *RingBuffer) IsEmpty() bool { return rb.head.Load() == rb.tail.Load() }
