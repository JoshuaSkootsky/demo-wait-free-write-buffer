package main

import (
	"sync"
	"testing"
	"time"

	rb "github.com/JoshuaSkootsky/wait-free-write-buffer"
)

type TestData struct {
	ID        int
	Value     string
	Timestamp int64
}

func TestBasicWriteRead(t *testing.T) {
	buffer := rb.New[TestData](1024)

	data := TestData{ID: 1, Value: "test", Timestamp: time.Now().UnixNano()}
	buffer.Write(data)

	var cursor uint64
	result, ok := buffer.Read(&cursor)
	if !ok {
		t.Fatal("expected read to succeed")
	}
	if result.ID != 1 {
		t.Errorf("expected ID 1, got %d", result.ID)
	}
}

func TestReadEmptyBuffer(t *testing.T) {
	buffer := rb.New[TestData](1024)

	var cursor uint64
	_, ok := buffer.Read(&cursor)
	if ok {
		t.Error("expected read from empty buffer to fail")
	}
}

func TestMultipleWritesReads(t *testing.T) {
	buffer := rb.New[TestData](1024)

	for i := 0; i < 10; i++ {
		buffer.Write(TestData{ID: i, Value: "data"})
	}

	var cursor uint64
	for i := 0; i < 10; i++ {
		result, ok := buffer.Read(&cursor)
		if !ok {
			t.Fatalf("read %d failed", i)
		}
		if result.ID != i {
			t.Errorf("expected ID %d, got %d", i, result.ID)
		}
	}
}

func TestReadWithGap(t *testing.T) {
	buffer := rb.New[TestData](1024)

	buffer.Write(TestData{ID: 1})
	buffer.Write(TestData{ID: 2})

	var cursor, gapStart, gapEnd uint64
	_, ok := buffer.ReadWithGap(&cursor, &gapStart, &gapEnd)
	if !ok {
		t.Fatal("expected read to succeed")
	}

	_, ok = buffer.ReadWithGap(&cursor, &gapStart, &gapEnd)
	if !ok {
		t.Fatal("expected second read to succeed")
	}

	_, ok = buffer.ReadWithGap(&cursor, &gapStart, &gapEnd)
	if ok {
		t.Error("expected third read to fail (empty buffer)")
	}
}

func TestConcurrentWriteRead(t *testing.T) {
	buffer := rb.New[TestData](1024)

	var wg sync.WaitGroup
	writeDone := make(chan bool)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			buffer.Write(TestData{ID: i})
			time.Sleep(time.Microsecond)
		}
		close(writeDone)
	}()

	reads := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		var cursor uint64
		for {
			_, ok := buffer.Read(&cursor)
			if ok {
				reads++
			} else {
				select {
				case <-writeDone:
					return
				default:
					time.Sleep(time.Microsecond)
				}
			}
		}
	}()

	wg.Wait()
	if reads != 100 {
		t.Errorf("expected 100 reads, got %d", reads)
	}
}

func TestPowerOfTwoSize(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("non-power-of-2 size causes panic (expected): %v", r)
		}
	}()

	_ = rb.New[TestData](uint64(1000))
	t.Error("expected panic for non-power-of-2 size")
}

func TestWrapAround(t *testing.T) {
	size := 16
	buffer := rb.New[TestData](uint64(size))

	for i := 0; i < size+5; i++ {
		buffer.Write(TestData{ID: i})
	}

	var cursor uint64
	maxReads := size + 5
	for i := 0; i < maxReads; i++ {
		result, ok := buffer.Read(&cursor)
		if !ok {
			break
		}
		if i >= size && result.ID != i-size+5 {
			t.Errorf("expected ID %d, got %d", i-size+5, result.ID)
		}
	}
}

func TestCursorPersistence(t *testing.T) {
	buffer := rb.New[TestData](1024)

	buffer.Write(TestData{ID: 1})
	buffer.Write(TestData{ID: 2})

	var cursor uint64
	_, _ = buffer.Read(&cursor)
	_, _ = buffer.Read(&cursor)

	buffer.Write(TestData{ID: 3})

	_, ok := buffer.Read(&cursor)
	if !ok {
		t.Fatal("expected read to succeed after write")
	}
	if cursor != 3 {
		t.Errorf("expected cursor 3, got %d", cursor)
	}
}
