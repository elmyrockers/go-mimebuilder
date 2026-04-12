package mimebuilder

import (
	"crypto/rand"
	"sync/atomic"
)

var (
	// 1KB Reservoir - Stays in CPU L1/L2 Cache
		entropy    [1024]byte
	// Atomic cursor for thread-safe access without Mutex
		entropyIdx atomic.Uint32
)

func init() {
	// Fill the reservoir once when the app starts on our server
		_, _ = rand.Read(entropy[:])
}

// setRandomBytes claims 16 bytes from the reservoir atomically
func setRandomBytes(b []byte) {
	size := uint32(len(b)) // 16
	
	// Atomically move the cursor
		newVal := entropyIdx.Add(size)

	// Bitwise mask to wrap the index safely within 0-1008
	// (1024 - 16 = 1008). This ensures idx + 16 never exceeds 1024.
		idx := (newVal - size) & (1024 - size)

	// If it wraps around, we are still safe because our setBoundaries()
	// XORs this with the current Nanosecond time and PID.
		copy(b, entropy[idx:idx+size])
}