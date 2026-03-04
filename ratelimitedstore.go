package desyncutil

import (
"context"
"fmt"
"golang.org/x/time/rate"
)

const (
// maxChunkSize is the default maximum chunk size used by desync (256 KiB).
// The token bucket burst must be at least this large, otherwise WaitN will
// always return an error for chunks at or near the maximum size.
maxChunkSize = 256 * 1024
)

// RateLimitedStore wraps a desync.Store and limits the bandwidth consumed by
// GetChunk calls. HasChunk, Close, and String are passed through unchanged.
//
// The rate limit is shared across all concurrent GetChunk calls, making it
// suitable for use with desync’s parallel download flag (-n).
//
// Rate limiting is applied after the chunk is fetched (post-charge): the
// caller blocks for as long as required to stay within the configured
// bytes-per-second limit. A small burst is therefore possible on the first
// call, or after a period of inactivity.
type RateLimitedStore struct {
store   Store
limiter *rate.Limiter
}

// NewRateLimitedStore returns a RateLimitedStore that wraps s and limits
// GetChunk throughput to bytesPerSecond.
//
// The token-bucket burst is set to max(256 KiB, bytesPerSecond) so that a
// single maximum-sized chunk can always be admitted in one WaitN call.
func NewRateLimitedStore(s Store, bytesPerSecond float64) *RateLimitedStore {
burst := int(bytesPerSecond)
if burst < maxChunkSize {
burst = maxChunkSize
}
return &RateLimitedStore{
store:   s,
limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), burst),
}
}

// GetChunk retrieves the chunk from the inner store, then waits on the token
// bucket for a number of tokens equal to the uncompressed chunk size. This
// ensures that sustained throughput does not exceed the configured limit.
//
// Note: when the inner store returns compressed chunks, the token charge is
// based on the uncompressed size rather than the compressed wire size. Because
// zstd compression typically reduces chunk size by 2–4x, the limiter will fire
// more often than the raw bandwidth alone would require, making the effective
// limit more conservative than the configured value. The desync Chunk type
// does not expose the original compressed bytes through any public method, so
// correcting for this without forking desync is not practical. For the purpose
// of capping bandwidth the conservative error is acceptable.
//
// The wait uses context.Background() because the Store interface does not
// thread a context through GetChunk. The wait cannot therefore be cancelled.
func (r *RateLimitedStore) GetChunk(id ChunkID) (*Chunk, error) {
chunk, err := r.store.GetChunk(id)
if err != nil {
return nil, err
}

data, err := chunk.Data()
if err != nil {
	return nil, err
}

n := len(data)
if n > r.limiter.Burst() {
	// This should not happen with the burst sizing above, but guard
	// against it rather than letting WaitN return a permanent error.
	n = r.limiter.Burst()
}

// WaitN blocks until the limiter permits n tokens. We use a background
// context because GetChunk does not receive one from the Store interface.
if err := r.limiter.WaitN(context.Background(), n); err != nil {
	return nil, fmt.Errorf("rate limiter: %w", err)
}

return chunk, nil
}

// HasChunk passes the call through to the inner store unchanged.
func (r *RateLimitedStore) HasChunk(id ChunkID) (bool, error) {
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
