// Copyright 2020 Authors of Hubble
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ring

import (
	"context"

	pb "github.com/moolen/juno/proto"
)

// RingReader is a reader for a Ring container.
type RingReader struct {
	ring *Ring
	idx  uint64
	ctx  context.Context
	c    <-chan *pb.Trace
}

// NewRingReader creates a new RingReader that starts reading the ring at the
// position given by start.
func NewRingReader(ring *Ring, start uint64) *RingReader {
	return &RingReader{
		ring: ring,
		idx:  start,
		ctx:  nil,
	}
}

// Previous reads the event at the current position and decrement the read
// position. When no more event can be read, Previous returns nil.
func (r *RingReader) Previous() *pb.Trace {
	var e *pb.Trace
	// when the ring is not full, ring.read() may return <nil>, true
	// in such a case, one should continue reading
	for ok := true; e == nil && ok; r.idx-- {
		e, ok = r.ring.read(r.idx)
	}
	return e
}

// Next reads the event at the current position and increment the read position.
// When no more event can be read, Next returns nil.
func (r *RingReader) Next() *pb.Trace {
	var e *pb.Trace
	// when the ring is not full, ring.read() may return <nil>, true
	// in such a case, one should continue reading
	for ok := true; e == nil && ok; r.idx++ {
		e, ok = r.ring.read(r.idx)
	}
	return e
}

// NextFollow reads the event at the current position and increment the read
// position by one. If there are no more event to read, NextFollow blocks
// until the next event is added to the ring or the context is canceled.
func (r *RingReader) NextFollow(ctx context.Context) *pb.Trace {
	// if the context changed between invocations, we also have to restart
	// readFrom, as the old readFrom instance will be using the old context.
	if r.c == nil || r.ctx != ctx {
		r.c = r.ring.ReadFrom(ctx, r.idx)
		r.ctx = ctx
	}

	select {
	case e, ok := <-r.c:
		if !ok {
			// channel can only be closed by readFrom if ctx is canceled
			r.c = nil
			r.ctx = nil
			return nil
		}
		// increment idx so that future calls to the ring reader will
		// continue reading from were we stopped.
		r.idx++
		return e
	case <-ctx.Done():
		r.c = nil
		r.ctx = nil
		return nil
	}
}
