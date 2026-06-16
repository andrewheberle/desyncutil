// This package provides utilities that complement the [desync] library.
package desyncutil

import (
	"context"
	"fmt"

	"github.com/folbricht/desync"
	"golang.org/x/time/rate"
)

var _ desync.Store = (*RateLimitedStore)(nil)

// RateLimitedStore wraps a [desync.Store] and limits the bandwidth consumed by
// GetChunk calls. HasChunk, Close, and String are passed through unchanged.
//
// The rate limit is shared across all concurrent GetChunk calls, making it
// suitable for use with desync's parallel download flag (-n).
//
// Rate limiting is applied after the chunk is fetched (post-charge): the
// caller blocks for as long as required to stay within the configured
// bytes-per-second limit. A small burst is therefore possible on the first
// call, or after a period of inactivity.
type RateLimitedStore struct {
	store   desync.Store
	limiter *rate.Limiter
}

// NewRateLimitedStore returns a [RateLimitedStore] that wraps s and limits
// [RateLimitedStore.GetChunk] throughput to bytesPerSecond.
//
// The token-bucket burst is set to bytesPerSecond, rounded up to at least 1.
// Chunks larger than the burst are charged across multiple WaitN calls (see
// GetChunk), so any bytesPerSecond value, however small, is honoured
// accurately rather than being silently floored to a minimum chunk size.
func NewRateLimitedStore(s desync.Store, bytesPerSecond float64) *RateLimitedStore {
	burst := max(int(bytesPerSecond), 1)
	return &RateLimitedStore{
		store:   s,
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burst),
	}
}

// GetChunk retrieves the chunk from the inner store, then waits on the token
// bucket for a number of tokens equal to the uncompressed chunk size. This
// ensures that sustained throughput does not exceed the configured limit.
//
// Because the limiter's burst may be smaller than the chunk (this is expected
// when a low bytesPerSecond value is configured), the wait is split across as
// many WaitN calls as required, each requesting at most the burst size. This
// keeps the byte accounting accurate at any configured rate, rather than
// under-charging large chunks against a small burst.
//
// Note: when the inner store returns compressed chunks, the token charge is
// based on the uncompressed size rather than the compressed wire size. Because
// zstd compression typically reduces chunk size by 2-4x, the limiter will fire
// more often than the raw bandwidth alone would require, making the effective
// limit more conservative than the configured value. The desync Chunk type
// does not expose the original compressed bytes through any public method, so
// correcting for this without forking desync is not practical. For the purpose
// of capping bandwidth the conservative error is acceptable.
//
// The wait uses context.Background() because the Store interface does not
// thread a context through GetChunk. The wait cannot therefore be cancelled.
func (r *RateLimitedStore) GetChunk(id desync.ChunkID) (*desync.Chunk, error) {
	chunk, err := r.store.GetChunk(id)
	if err != nil {
		return nil, err
	}

	data, err := chunk.Data()
	if err != nil {
		return nil, err
	}

	burst := r.limiter.Burst()
	for remaining := len(data); remaining > 0; {
		n := min(remaining, burst)
		if err := r.limiter.WaitN(context.Background(), n); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}
		remaining -= n
	}

	return chunk, nil
}

// HasChunk passes the call through to the inner store unchanged.
func (r *RateLimitedStore) HasChunk(id desync.ChunkID) (bool, error) {
	return r.store.HasChunk(id)
}

// Close passes the call through to the inner store unchanged.
func (r *RateLimitedStore) Close() error {
	return r.store.Close()
}

// String returns a human-readable description of the store.
func (r *RateLimitedStore) String() string {
	return fmt.Sprintf("rate-limited(%s)", r.store)
}
