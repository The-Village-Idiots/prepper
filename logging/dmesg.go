package logging

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"sync"
)

// DmesgCapacity is the maximum DmesgCapacity ring buffer capacity, which
// stores the text written to it.
const DmesgCapacity = 8192

// DMesg implements a ring buffer for storing logging output. It has a default
// capacity of DmesgCapacity. Once the logging ring buffer is exhausted, it
// wraps back to zero and the oldest entries are discarded. Reads are
// guaranteed to occur in FIFO order and always succeed. Writes are guaranteed
// to write all data supplied and never fail. If data to write exceeds the ring
// buffer size it is overwritten.
type Dmesg struct {
	// Protects buf and head.
	mut sync.Mutex
	// Write head location. This wraps back to zero once at the capacity
	// and is set to *the next* location that will be written to.
	head int
	// Wrapped is set to true after the first wrap of the write buffer.
	// When true, a read will occur from the write pointer to the buffer
	// capacity, before wrapping back to zero and up to the byte before the
	// write pointer.
	wrapped bool
	// Fixed output buffer.
	buf [DmesgCapacity]byte
}

// NewDmesg allocates and returns a new dmesg buffer.
func NewDmesg() *Dmesg {
	return &Dmesg{mut: sync.Mutex{}, buf: [DmesgCapacity]byte{}}
}

// LogOutput returns an output which is suitable for use with the log package.
// It duplicates output between stderr and the Dmesg buffer.
func (d *Dmesg) LogOutput() io.Writer {
	return io.MultiWriter(d, os.Stderr)
}

// advance moves the write head forward by n bytes, carefully wrapping at n ==
// cap. advance assumes that d.mut is already held.
func (d *Dmesg) advance(n int) {
	inc := n
	if d.head == 0 && n < 0 {
		d.head = DmesgCapacity
	}
	if d.head+n >= len(d.buf) {
		inc = (d.head + n) - len(d.buf)
		d.head = 0
	}

	d.head += inc
}

// Write writes data to the ring buffer, possibly wrapping to zero and
// overwriting existing data. Writes always succeed and err is always nil. n
// will always be len(p).
func (d *Dmesg) Write(p []byte) (n int, err error) {
	d.mut.Lock()
	defer d.mut.Unlock()

	// Bytes remaining to write.
	remain := len(d.buf) - d.head

	// Partition up to the wrapping boundary.
	// If we will wrap, split at the maximum boundary and write the first
	// half to the end of the buffer then overwrite the beginning.
	var first, last []byte
	first = p[:]
	if len(p) > remain {
		first = p[:remain]
		last = p[remain:]
		d.wrapped = true
	}

	// Write halves to buffer.
	copy(d.buf[d.head:], first)
	copy(d.buf[:], last)

	// Advance write pointer
	d.advance(len(p))

	n = len(p)
	return
}

// Grab grabs exclusive control over Dmesg for the current execution thread.
// This is mostly useful for reads from the buffer to ensure no interleaving
// writes which invalidate the reader. The returned struct automatically
// releases the lock again when it is no longer reachable. It should never be
// retained!
func (d *Dmesg) Grab() *DmesgLock {
	d.mut.Lock()
	return newDmesgLock(d)
}

// Reader returns an io.Reader which allows reads from the buffer valid until
// the next call to Write.
func (d *Dmesg) Reader() io.Reader {
	// If we haven't wrapped yet, no need to do anything special
	if !d.wrapped {
		return bytes.NewReader(d.buf[:d.head])
	}

	return &DmesgReader{
		d.buf[d.head:], d.buf[:d.head],
		false, 0,
	}
}

// DmesgLock is returned from Grab and is a safety net against accidental
// failures to unlock. The lock is automatically released when DmesgLock goes
// out of scope. You should not retain this struct.
type DmesgLock struct {
	*sync.Mutex
	rel bool
}

// newDmesgLock returns a pointer to a dmesgLock instance with the finalizer
// set to the release method.
func newDmesgLock(d *Dmesg) *DmesgLock {
	l := &DmesgLock{&d.mut, false}
	runtime.SetFinalizer(l, l.finalizer)

	return l
}

func (d *DmesgLock) finalizer(_ *DmesgLock) {
	d.Release()
}

// Release manually releases the targetted dmesg buffer. You must use this
// method instead of the unlock method.
func (d *DmesgLock) Release() {
	d.Mutex.Unlock()
	runtime.SetFinalizer(d, nil)
}

// Unlock prevents invalid usage of the contained mutex.
func (d *DmesgLock) Unlock() {
	panic("invalid unlock of DmesgLock; use Release instead!")
}

// A DmesgReader contains a read-only reference to a live Dmesg buffer. It
// reads data from the buffer in the order in which it was written. Once the
// end of the data stream has been reached, the reader returns io.EOF.
// Subsequent reads will succeed but shall read back from zero again.
//
// IMPORTANT: Any and all calls to Write on the original buffer between calls
// to Read on a DmesgReader will invalidate the reader, necessitating a new
// call to Reader on the original buffer. DmesgReader assumes that the buffer
// is already exclusively locked.
type DmesgReader struct {
	upper, lower []byte
	inLower      bool
	head         int
}

// Read reads at most len(p) bytes from the Dmesg buffer, advancing the read
// head by len(p). The datastream is flattened such that data is read off in
// the order in which it was written. Once the end of the datastream is
// reached, io.EOF is returned. Read locks the Dmesg buffer while reading is
// taking place.
func (d *DmesgReader) Read(p []byte) (int, error) {
	if d.inLower {
		n := copy(p, d.lower[d.head:])
		d.head += n

		if d.head >= len(d.lower) {
			d.head, d.inLower = 0, false
			return n, io.EOF
		}

		return n, nil
	}

	n := copy(p, d.upper[d.head:])
	d.head += n

	if d.head >= len(d.upper) {
		d.head, d.inLower = 0, true
	}

	return n, nil
}
