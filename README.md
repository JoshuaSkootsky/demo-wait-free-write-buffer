# Ring Buffer Demo

This project demonstrates usage of the [wait-free-write-buffer](https://github.com/JoshuaSkootsky/wait-free-write-buffer) Go library, a wait-free single-producer multi-consumer ring buffer.

## Library Overview

The library provides a lock-free ring buffer with these key properties:

- **Wait-free writes**: Single writer never blocks
- **Lock-free reads**: Multiple readers can read concurrently
- **Power-of-2 sizing**: Buffer size must be a power of 2
- **Cursor-based reading**: Readers track position via cursors

## API Usage

### Creating a Buffer

```go
buffer := rb.New[YourType](size)
```

The size must be a power of 2 (e.g., 1024, 65536).

### Writing Data

```go
buffer.Write(data)
```

Single-producer, never blocks.

### Reading Data

```go
var cursor uint64
result, ok := buffer.Read(&cursor)
```

- Returns `ok=false` when buffer is empty
- Cursor advances atomically
- Multiple readers each maintain their own cursor

### Reading with Gap Detection

```go
var cursor, gapStart, gapEnd uint64
result, ok := buffer.ReadWithGap(&cursor, &gapStart, &gapEnd)
```

Detects gaps in the sequence when data has been overwritten.

## Test Coverage

The `main_test.go` file demonstrates:

| Test | Purpose |
|------|---------|
| `TestBasicWriteRead` | Simple write/read cycle |
| `TestReadEmptyBuffer` | Empty buffer returns false |
| `TestMultipleWritesReads` | Sequential operations |
| `TestReadWithGap` | Gap detection API |
| `TestConcurrentWriteRead` | Thread safety with concurrent goroutines |
| `TestPowerOfTwoSize` | Validates power-of-2 requirement |
| `TestWrapAround` | Old data overwritten when full |
| `TestCursorPersistence` | Cursor advances correctly |

## Running Tests

```bash
go test -v ./...
```

## Production Example

See `main.go` for a high-frequency market data example with:
- Single writer publishing at ~60 FPS
- Multiple concurrent readers (network sender, analytics)
- Metrics tracking for performance monitoring
