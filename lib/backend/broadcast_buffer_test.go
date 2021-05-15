package backend

import (
	"context"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

const (
	itemOne   = "first line"
	itemTwo   = "second line"
	itemThree = "third line"
)

// TestRead verifies that a new reader will correctly read a single write.
func TestRead(t *testing.T) {
	b := newBroadcastBuffer()

	_, err := b.Write([]byte(itemOne))
	require.Nil(t, err)

	r := b.NewReader(context.Background())
	p := make([]byte, 15)

	n, err := r.Read(p)
	require.Nil(t, err)
	require.Equal(t, 10, n)

	err = b.Close()
	require.Nil(t, err)
}

// TestEOFDetection verifies that the buffer correctly identifies the EOF
// condition. Previously, this particular sequence of reads and writes
// would cause the reader to incorrectly return io.EOF even though the
// buffer was not closed.
func TestEOFDetection(t *testing.T) {
	timeout := time.After(1 * time.Second)
	done := make(chan bool)

	go func() {
		b := newBroadcastBuffer()

		_, _ = b.Write([]byte(itemOne))
		_, _ = b.Write([]byte(itemTwo))

		r := b.NewReader(context.Background())
		p := make([]byte, 15)

		_, _ = r.Read(p)
		_, _ = b.Write([]byte(itemThree))
		_, _ = r.Read(p)
		_, _ = r.Read(p)

		// Fourth read. This should timeout waiting for a notification
		// on the channel.
		_, _ = r.Read(p)

		done <- true
	}()

	select {
	case <-timeout:
		// We expect this test case to timeout as the reader is
		// correctly waiting for new data.
	case <-done:
		t.Fail()
	}
}

// TestLateReadersGetAllData verifies that readers created after the
// BroadcastBuffer has been closed will still be able to read all the
// previous data.
func TestLateReadersGetAllData(t *testing.T) {
	b := newBroadcastBuffer()

	_, err := b.Write([]byte(itemOne))
	require.Nil(t, err)
	_, err = b.Write([]byte(itemTwo))
	require.Nil(t, err)
	err = b.Close()
	require.Nil(t, err)

	r := b.NewReader(context.Background())
	data, err := ioutil.ReadAll(r)
	require.Nil(t, err)
	require.Equal(t, itemOne+itemTwo, string(data))
}

func TestContext(t *testing.T) {
	b := newBroadcastBuffer()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	r := b.NewReader(ctx)

	_, err := ioutil.ReadAll(r)
	require.EqualError(t, err, context.DeadlineExceeded.Error())
}

// TestConcurrentRead verifies that readers can read correctly from a
// slow writer.
func TestConcurrentRead(t *testing.T) {
	b := newBroadcastBuffer()

	var wg sync.WaitGroup
	for range []int{0, 1, 2} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := b.NewReader(context.Background())
			out, err := ioutil.ReadAll(r)
			require.Nil(t, err)
			require.Equal(t, itemOne+itemTwo, string(out))
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		_, err := b.Write([]byte(itemOne))
		require.Nil(t, err)

		time.Sleep(1 * time.Second)
		_, err = b.Write([]byte(itemTwo))
		require.Nil(t, err)

		time.Sleep(1 * time.Second)
		err = b.Close()
		require.Nil(t, err)
	}()
	wg.Wait()
}
