package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// randSrc lets us have a non-global random number generator that's safe
// for concurrent use.  It's sad that this is basically a cheap ripoff of the
// math/rand lockedSource structure... which for some reason isn't exposed.
type randSrc struct {
	sync.Mutex
	s rand.Source64
}

func (r *randSrc) Int63() int64 {
	r.Lock()
	var n = r.s.Int63()
	r.Unlock()
	return n
}

func (r *randSrc) Uint64() uint64 {
	r.Lock()
	var n = r.s.Uint64()
	r.Unlock()
	return n
}

func (r *randSrc) Seed(seed int64) {
	r.Lock()
	r.s.Seed(seed)
	r.Unlock()
}

var rnd = rand.New(&randSrc{s: rand.NewSource(time.Now().UnixNano()).(rand.Source64)})

// genid returns a unique 128-bit random number, much like a uuid, but without
// the insane overhead UUIDs have (seriously, have you seen the absurd code in
// the various uuid packages?)
func genid() string {
	var r1 = rnd.Uint64()
	var r2 = rnd.Uint64()
	return fmt.Sprintf("%x%x", r1, r2)
}
