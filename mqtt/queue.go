package mqtt

import (
	"fmt"
	"os"
	"sync"
)

func init() {
	os.Mkdir("session", 0755)
}

// PacketWriter is a writer that accepts Packets.
type PacketWriter interface {
	WritePacket(pkt Packet) (int, error)
}

// Queue (Packet-Queue) can hold packets to unblock a writer or to stack packets when
// there is no writer.
type Queue struct {
	id string // unique!

	// writer, session, size and sigFlushed are converd by the writeLock-mutex
	writer       PacketWriter
	session      *os.File
	size, offset int

	sigFlushed chan error

	writeLock sync.Mutex
}

// NewQueue crates a new Packet-Queue that must be named uniquely with the id.
func NewQueue(id string) *Queue {
	queue := &Queue{id: id}
	session, err := os.OpenFile("session/"+id+".tmp", os.O_RDONLY, 0755)
	if err == nil {
		stat, _ := session.Stat()
		queue.size = int(stat.Size())
		buffer := make([]byte, 128)
		offset := 0
		for {
			n, _ := session.Read(buffer)
			if n == 0 { // empty or only-zero file
				session.Close()
				os.Remove("session/" + id + ".tmp")
				queue.offset = 0
				queue.size = 0
				break
			}
			for i := 0; i < n; i++ {
				if buffer[i] != 0 {
					queue.offset = offset
					queue.size -= offset
					session.Close()
					// log.Println("queue size", queue.size, "at", queue.offset)
					return queue
				}
				offset++
			}
		}
	}
	return queue
}

// ServeWriter gives the queue a new writer.
// It will be feed with all packets that are waiting in the queue and will
// recieve all future packets until the writer fails.
// You must not call this function as long as it still has an active (non-failed)
// writer.
// The call blocks until the queue is empty.
func (q *Queue) ServeWriter(w PacketWriter) error {

	q.writeLock.Lock()

	if q.size != 0 {
		q.writeLock.Unlock()

		oldOffset := q.offset
		session, err := os.OpenFile("session/"+q.id+".tmp", os.O_RDONLY, 0755)
		if err != nil {
			panic(err)
		}
		session.Seek(int64(q.offset), 0)
		defer session.Close()

		for {
			pkt, n, err := Read(session)
			if n == 0 && err != nil {
				panic(err)
			}

			d, err := w.WritePacket(pkt)
			if err != nil {

				diff := q.offset - oldOffset
				if diff != 0 {
					zeros := make([]byte, 64)
					session.Close()
					session, err = os.OpenFile("session/"+q.id+".tmp", os.O_WRONLY, 0755)
					if err != nil {
						panic(err)
					}
					session.Seek(int64(oldOffset), 0)
					for oldOffset < q.offset {
						session.Write(zeros[0:min(q.offset-oldOffset, 64)])
						oldOffset += 64
					}
				}

				//session.Seek(int64(q.offset), 0)

				q.writeLock.Lock()
				if q.sigFlushed != nil {
					q.sigFlushed <- err
					q.sigFlushed = nil
				}
				q.writeLock.Unlock()
				return err
			}

			if d != n {
				panic("Queue read write mismatch")
			}

			q.writeLock.Lock()
			q.size -= d
			q.offset += d
			// log.Printf("< %d %#v\n", q.size, pkt)
			if q.size == 0 {
				q.writer = w

				if q.sigFlushed != nil {
					q.sigFlushed <- nil
					q.sigFlushed = nil
				}

				q.session.Close()
				q.session = nil
				os.Remove("session/" + q.id + ".tmp")

				q.writeLock.Unlock()
				return nil
			}
			if q.size < 0 {
				panic(fmt.Errorf("invalid queue size: %d", q.size))
			}
			q.writeLock.Unlock()
		}
	} else {

		q.writer = w
		q.writeLock.Unlock()
		return nil
	}
}

func (q *Queue) beginSession(pkt Packet) {

	var err error
	q.session, err = os.OpenFile("session/"+q.id+".tmp", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	q.session.Seek(int64(q.offset), 0)
	d, err := pkt.WriteTo(q.session)
	if err != nil {
		panic(err)
	}
	// log.Printf("> %d %#v\n", d, pkt)
	q.size = d
}

// WritePacket inserts a new packet to this queue.
// This will call the writer directly if the queue is empty ("fast forward").
func (q *Queue) WritePacket(pkt Packet) (n int, err error) {

	q.writeLock.Lock()

	if q.size != 0 {

		n, _ = pkt.WriteTo(q.session)
		q.size += n
		// log.Printf("> %d %#v\n", q.size, pkt)
	} else {

		if q.writer == nil {
			q.beginSession(pkt)

		} else {

			n, err = q.writer.WritePacket(pkt)
			if err != nil {
				q.beginSession(pkt)
			}
		}
	}

	q.writeLock.Unlock()
	return
}

func (q *Queue) Flush() (err error) {

	var sigFlushed chan error
	var oldSigFlushed chan error

	q.writeLock.Lock()
	// id := q.id
	// size := q.size
	// writer := q.writer
	if q.size != 0 && q.writer != nil {
		sigFlushed = make(chan error)
		oldSigFlushed = q.sigFlushed
		q.sigFlushed = sigFlushed
	}
	q.writeLock.Unlock()

	if sigFlushed != nil {
		err = <-sigFlushed
		if oldSigFlushed != nil {
			oldSigFlushed <- err
		}
	}

	// log.Println("flush?", id, size, writer)

	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
