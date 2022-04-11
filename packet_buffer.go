package main

import (
	"errors"
)

type RingBuffer struct {
	buf  []byte // actual data
	size int    // buffer size in bytes
	r, w int    // read/write cursors
	t    int    // total bytes written in buffer
}

var (
	ErrZeroSize       = errors.New("Can't create zero-size buffer")
	ErrZeroRead       = errors.New("Can't read from empty buffer")
	ErrEmptyBuffer    = errors.New("Buffer is empty")
	ErrBufferOverflow = errors.New("Can't read from empty buffer")
)

// creates new ring buffer
func NewRingBuffer(n int) (*RingBuffer, error) {
	if n < 1 {
		return nil, ErrZeroSize
	}

	b := &RingBuffer{
		size: n,
		buf:  make([]byte, n),
	}
	return b, nil
}

func (b *RingBuffer) Size() int {
	return b.size
}

// Write to buff. Override if full.
func (b *RingBuffer) Write(p []byte) (int, error) {

	l := len(p)
	// store total bytes written

	// potential data loss
	// comment if its ok for your purporses
	if b.w < b.r && b.r-b.w <= l {
		return -1, ErrBufferOverflow
	}
	if b.w > b.r && b.size-b.w+b.r <= l {
		return -1, ErrBufferOverflow
	}

	// if we try to write more bytes than
	// buf can store, we write only last SIZE bytes
	// this will never happen because of pervious overflow check
	if int(l) > b.size {
		p = p[int(l)-b.size:]
	}
	b.t += int(l)

	// write with no split
	offset := b.size - b.w
	copy(b.buf[b.w:], p)
	// if not fit write from beggining
	// if also not fit second time just drop
	if int(len(p)) > offset {
		copy(b.buf, p[offset:])
	}

	// Update w cursor
	b.w = (b.w + int(len(p))) % b.size
	return l, nil
}

func (b *RingBuffer) Read(p []byte) (n int, err error) {

	// can't read to zero
	if len(p) == 0 {
		return -2, ErrZeroRead
	}

	if b.r == b.w {
		return -1, ErrEmptyBuffer
	}

	// if read before write cursor -> single chunk
	if b.w > b.r {
		n = b.w - b.r
		if n > len(p) {
			n = len(p)
		}
		copy(p, b.buf[b.r:b.r+n])
		b.r = (b.r + n) % b.size
		return
	}

	// if write cursor before read but we still can fit
	n = b.size - b.r + b.w
	if n > len(p) {
		n = len(p)
	}
	if b.r+n <= b.size {
		copy(p, b.buf[b.r:b.r+n])
	} else { // need to read from second chunk at start
		c1 := b.size - b.r
		c2 := n - c1
		copy(p, b.buf[b.r:b.size])
		copy(p[c1:], b.buf[0:c2])
	}

	b.r = (b.r + n) % b.size

	return n, err
}

// amount of bytes can be read
func (b *RingBuffer) Buffered() int {
	if b.w > b.r {
		return b.w - b.r
	} else {
		return b.size - b.r + b.w
	}
}

// reset the buffer to empty state
func (b *RingBuffer) Reset() {
	b.w = 0
	b.t = 0
	b.r = 0
}
