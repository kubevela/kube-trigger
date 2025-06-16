/*
Copyright 2023 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workqueue

import (
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter .
type RateLimiter interface {
	// When gets an item and gets to decide how long that item should wait
	When(item interface{}) time.Duration
	// Forget indicates that an item is finished being retried.  Doesn't matter whether it's for failing
	// or for success, we'll stop tracking it
	Forget(item interface{})
	// NumRequeues returns back how many failures the item has had
	NumRequeues(item interface{}) int
}

// DefaultControllerRateLimiter is a no-arg constructor for a default rate limiter for a workqueue.  It has
// both overall and per-item rate limiting.  The overall is a token bucket and the per-item is exponential
func DefaultControllerRateLimiter() RateLimiter {
	return NewMaxOfRateLimiter(
		NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)
}

// BucketRateLimiter adapts a standard bucket to the workqueue ratelimiter API
type BucketRateLimiter struct {
	*rate.Limiter
}

// RateLimiter .
var _ RateLimiter = &BucketRateLimiter{}

// When .
func (r *BucketRateLimiter) When(_ interface{}) time.Duration {
	return r.Limiter.Reserve().Delay()
}

// NumRequeues .
func (r *BucketRateLimiter) NumRequeues(_ interface{}) int {
	return 0
}

// Forget .
func (r *BucketRateLimiter) Forget(_ interface{}) {
}

// ItemExponentialFailureRateLimiter does a simple baseDelay*2^<num-failures> limit
// dealing with max failures and expiration are up to the caller
type ItemExponentialFailureRateLimiter struct {
	failuresLock sync.Mutex
	failures     map[interface{}]int

	baseDelay time.Duration
	maxDelay  time.Duration
}

var _ RateLimiter = &ItemExponentialFailureRateLimiter{}

// NewItemExponentialFailureRateLimiter creates a new ItemExponentialFailureRateLimiter
func NewItemExponentialFailureRateLimiter(baseDelay time.Duration, maxDelay time.Duration) RateLimiter {
	return &ItemExponentialFailureRateLimiter{
		failures:  map[interface{}]int{},
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

// DefaultItemBasedRateLimiter is a no-arg constructor for a default rate limiter for a workqueue
func DefaultItemBasedRateLimiter() RateLimiter {
	return NewItemExponentialFailureRateLimiter(time.Millisecond, 1000*time.Second)
}

// When .
func (r *ItemExponentialFailureRateLimiter) When(item interface{}) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	exp := r.failures[item]
	r.failures[item]++

	// The backoff is capped such that 'calculated' value never overflows.
	backoff := float64(r.baseDelay.Nanoseconds()) * math.Pow(2, float64(exp))
	if backoff > math.MaxInt64 {
		return r.maxDelay
	}

	calculated := time.Duration(backoff)
	if calculated > r.maxDelay {
		return r.maxDelay
	}

	return calculated
}

// NumRequeues .
func (r *ItemExponentialFailureRateLimiter) NumRequeues(item interface{}) int {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	return r.failures[item]
}

// Forget .
func (r *ItemExponentialFailureRateLimiter) Forget(item interface{}) {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	delete(r.failures, item)
}

// ItemFastSlowRateLimiter does a quick retry for a certain number of attempts, then a slow retry after that
type ItemFastSlowRateLimiter struct {
	failuresLock sync.Mutex
	failures     map[interface{}]int

	maxFastAttempts int
	fastDelay       time.Duration
	slowDelay       time.Duration
}

// RateLimiter .
var _ RateLimiter = &ItemFastSlowRateLimiter{}

// NewItemFastSlowRateLimiter creates a new ItemFastSlowRateLimiter
func NewItemFastSlowRateLimiter(fastDelay, slowDelay time.Duration, maxFastAttempts int) RateLimiter {
	return &ItemFastSlowRateLimiter{
		failures:        map[interface{}]int{},
		fastDelay:       fastDelay,
		slowDelay:       slowDelay,
		maxFastAttempts: maxFastAttempts,
	}
}

// When .
func (r *ItemFastSlowRateLimiter) When(item interface{}) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	r.failures[item]++

	if r.failures[item] <= r.maxFastAttempts {
		return r.fastDelay
	}

	return r.slowDelay
}

// NumRequeues .
func (r *ItemFastSlowRateLimiter) NumRequeues(item interface{}) int {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	return r.failures[item]
}

// Forget .
func (r *ItemFastSlowRateLimiter) Forget(item interface{}) {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	delete(r.failures, item)
}

// MaxOfRateLimiter calls every RateLimiter and returns the worst case response
// When used with a token bucket limiter, the burst could be apparently exceeded in cases where particular items
// were separately delayed a longer time.
type MaxOfRateLimiter struct {
	limiters []RateLimiter
}

// When .
func (r *MaxOfRateLimiter) When(item interface{}) time.Duration {
	ret := time.Duration(0)
	for _, limiter := range r.limiters {
		curr := limiter.When(item)
		if curr > ret {
			ret = curr
		}
	}

	return ret
}

// NewMaxOfRateLimiter creates a new MaxOfRateLimiter
func NewMaxOfRateLimiter(limiters ...RateLimiter) RateLimiter {
	return &MaxOfRateLimiter{limiters: limiters}
}

// NumRequeues .
func (r *MaxOfRateLimiter) NumRequeues(item interface{}) int {
	ret := 0
	for _, limiter := range r.limiters {
		curr := limiter.NumRequeues(item)
		if curr > ret {
			ret = curr
		}
	}

	return ret
}

// Forget .
func (r *MaxOfRateLimiter) Forget(item interface{}) {
	for _, limiter := range r.limiters {
		limiter.Forget(item)
	}
}

// WithMaxWaitRateLimiter have maxDelay which avoids waiting too long
type WithMaxWaitRateLimiter struct {
	limiter  RateLimiter
	maxDelay time.Duration
}

// NewWithMaxWaitRateLimiter creates a new WithMaxWaitRateLimiter
func NewWithMaxWaitRateLimiter(limiter RateLimiter, maxDelay time.Duration) RateLimiter {
	return &WithMaxWaitRateLimiter{limiter: limiter, maxDelay: maxDelay}
}

// When .
func (w WithMaxWaitRateLimiter) When(item interface{}) time.Duration {
	delay := w.limiter.When(item)
	if delay > w.maxDelay {
		return w.maxDelay
	}

	return delay
}

// Forget .
func (w WithMaxWaitRateLimiter) Forget(item interface{}) {
	w.limiter.Forget(item)
}

// NumRequeues .
func (w WithMaxWaitRateLimiter) NumRequeues(item interface{}) int {
	return w.limiter.NumRequeues(item)
}
