package backend

import (
	"context"
	"fmt"
	"io"
	"sync"

	log "github.com/sirupsen/logrus"
)

// broadcastBuffer is an io.Writer which allows many simultaneous io.Readers
// to read the data written to it. This is needed to enable multiple clients
// to stream the output from a single job from the beginning.
type broadcastBuffer struct {
	// this lock must be held for any reading or writing of the internal
	// state of the broadcastBuffer.
	mtx sync.RWMutex

	// closed represents whether the buffer has been closed.
	closed bool

	// data contains all the data written to the buffer.
	data [][]byte

	// consumers are channels to which new write notifications are
	// propagated allowing readers to be alerted when new data is
	// available to be read.
	consumers []chan<- struct{}
}

// newBroadcastBuffer returns a correctly initialized BroadcastBuffer.
func newBroadcastBuffer() *broadcastBuffer {
	return &broadcastBuffer{
		data:      make([][]byte, 0),
		consumers: make([]chan<- struct{}, 0),
	}
}

// size returns the number of chunks written to the buffer.
func (b *broadcastBuffer) size() int {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return len(b.data)
}

// dataAt returns the slice of data at the given index.
func (b *broadcastBuffer) dataAt(index int) []byte {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	return b.data[index]
}

// notificationChannel returns a channel which will receive notifications
// when new data is written to the buffer.
func (b *broadcastBuffer) notificationChannel() <-chan struct{} {
	c := make(chan struct{}, 1)

	b.mtx.Lock()
	defer b.mtx.Unlock()

	// If the broadcastBuffer is closed we also want to close this
	// channel to prevent consumers blocking as they wait for
	// notifications of new data.
	if b.closed {
		close(c)
	}
	b.consumers = append(b.consumers, c)

	return c
}

// Write copies the given bytes to the internal buffer and notifies any
// consumers that new data is available.
func (b *broadcastBuffer) Write(p []byte) (int, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.closed {
		return 0, fmt.Errorf("attempt to write to closed broadcastBuffer")
	}

	if len(p) == 0 {
		return 0, nil
	}

	// Since implementations of io.Writer must not retain p we need to
	// make a copy.
	pCopy := make([]byte, len(p))
	copy(pCopy, p)

	b.data = append(b.data, pCopy)

	for _, c := range b.consumers {
		select {
		// Notify all registered consumers that new data is available.
		case c <- struct{}{}:

		// If the consumer's channel already has a notification that has not been
		// received we don't want to block.
		default:
		}
	}
	return len(p), nil
}

// Close marks the broadcastBuffer as closed. All existing channels are
// closed but new channels may still be created to consume the data.
func (b *broadcastBuffer) Close() error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.closed = true

	// Close all the channels
	for _, c := range b.consumers {
		close(c)
	}
	return nil
}

// NewReader returns a new io.Reader which reads from the broadcastBuffer.
func (b *broadcastBuffer) NewReader(ctx context.Context) io.Reader {
	return &consumer{
		buf:           b,
		notifications: b.notificationChannel(),
		ctx:           ctx,
	}
}

// consumer is an io.Reader which reads from the BroadcastBuffer.
type consumer struct {
	// buf a reference to the broadcastBuffer whose data is being read.
	buf *broadcastBuffer

	// notifications is channel that receives write notifications from
	// the broadcastBuffer.
	notifications <-chan struct{}

	// current is a reference to the most recent data read from the
	// broadcastBuffer.
	current []byte

	// nextIndex is the next index in the broadcastBuffer that the
	// reader will consume.
	nextIndex int

	ctx context.Context
}

// Read will consume the next available bytes from the broadcastBuffer. If no
// bytes are available this call will block until either data is available or
// the broadcastBuffer is closed. This prevents io.Readers returning an
// io.ErrNoProgress in the case where the buffer is only updated intermittently.
func (r *consumer) Read(p []byte) (int, error) {
	// If we don't have any current data get some from the buffer.
	if len(r.current) == 0 {

		// In this case we've read all the data available from the
		// buffer and we want to block by listening on the channel
		// until more data is available or the channel is closed.
		if r.nextIndex >= r.buf.size() {
			select {
			case <-r.notifications:
			case <-r.ctx.Done():
				log.Info("early out of reading")
				return 0, r.ctx.Err()
			}
		}

		// If the writer's index has advanced then we have more
		// data to read.
		if r.nextIndex < r.buf.size() {
			r.current = r.buf.dataAt(r.nextIndex)
			r.nextIndex++

			// drain the channel to prevent consuming stale
			// notifications and mistakenly returning EOF.
			for len(r.notifications) > 0 {
				<-r.notifications
			}
		}
	}
	// If we still don't have any data then we must have reached the end
	// of the data.
	if len(r.current) == 0 {
		return 0, io.EOF
	}

	n := copy(p, r.current)
	r.current = r.current[n:]
	return n, nil
}
