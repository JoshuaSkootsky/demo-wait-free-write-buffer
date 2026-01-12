package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	rb "github.com/JoshuaSkootsky/wait-free-write-buffer"
)

// MarketData represents a typical high-frequency data event
type MarketData struct {
	Symbol    string
	Price     float64
	Volume    int
	Timestamp int64
}

// Metrics tracks buffer performance
type Metrics struct {
	writes, reads, gaps uint64
	mu                  sync.Mutex
}

func (m *Metrics) RecordWrite() { m.mu.Lock(); defer m.mu.Unlock(); m.writes++ }
func (m *Metrics) RecordRead()  { m.mu.Lock(); defer m.mu.Unlock(); m.reads++ }
func (m *Metrics) RecordGap()   { m.mu.Lock(); defer m.mu.Unlock(); m.gaps++ }
func (m *Metrics) Print() {
	m.mu.Lock()
	defer m.mu.Unlock()
	fmt.Printf("\n=== Metrics: Writes=%d Reads=%d Gaps=%d ===\n",
		m.writes, m.reads, m.gaps)
}

func main() {
	// ✅ FIXED: Use power-of-2 size (required)
	const bufferSize = 65536

	// ✅ FIXED: Correct API is New[Type](size)
	buffer := rb.New[MarketData](bufferSize)

	var wg sync.WaitGroup
	metrics := &Metrics{}
	stop := make(chan struct{})

	// Writer: Single goroutine, never blocks
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
		defer ticker.Stop()

		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			case <-ticker.C:
				buffer.Write(MarketData{
					Symbol:    "AAPL",
					Price:     150.0 + float64(i%100),
					Volume:    1000 + i,
					Timestamp: time.Now().UnixNano(),
				})
				metrics.RecordWrite()
			}
		}
	}()

	// Reader 1: Low-latency network sender
	wg.Add(1)
	go func() {
		defer wg.Done()
		var cursor, gapStart, gapEnd uint64
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		for {
			select {
			case <-stop:
				return
			default:

				_, ok := buffer.ReadWithGap(&cursor, &gapStart, &gapEnd)
				switch {
				case ok:
					metrics.RecordRead()
				case gapStart > 0:
					metrics.RecordGap()
					log.Printf("Network gap [%d, %d], skipping ahead", gapStart, gapEnd)
					cursor = gapEnd
					gapStart, gapEnd = 0, 0
				default:
					runtime.Gosched()
				}
			}
		}
	}()

	// Reader 2: Analytics processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		var cursor uint64

		for {
			select {
			case <-stop:
				return
			default:
				// ✅ FIXED: Use underscore for unused data
				_, ok := buffer.Read(&cursor)
				if ok {
					metrics.RecordRead()
					time.Sleep(50 * time.Microsecond)
				} else {
					time.Sleep(100 * time.Microsecond)
				}
			}
		}
	}()

	// Monitor
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				metrics.Print()
			}
		}
	}()

	// Run for 5 seconds
	time.Sleep(5 * time.Second)
	close(stop)
	wg.Wait()
	metrics.Print()
}
