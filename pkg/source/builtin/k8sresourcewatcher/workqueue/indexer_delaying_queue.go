package workqueue

import (
	"container/heap"
	"sync"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
)

const (
	queue_item_cap = 128
)

type indexerDelayingQueue struct {
	clock clock.Clock

	keyFunc func(obj interface{}) (string, error)

	// stopCh lets us signal a shutdown to the waiting loop
	stopCh chan struct{}
	// stopOnce guarantees we only signal shutdown a single time
	stopOnce sync.Once

	// heartbeat ensures we wait no more than maxWait before firing
	heartbeat clock.Ticker

	// waitingForAddCh is a buffered channel that feeds waitingForAdd
	waitingForAddCh chan *waitFor

	waitingForQueue *waitForPriorityQueue

	knownPrepareEntries cacheEntries

	queue []string

	// dirty defines all of the items that need to be processed.
	dirty set

	cond *sync.Cond

	addCond *sync.Cond

	shuttingDown bool
	drain        bool

	metrics queueMetrics

	unfinishedWorkUpdatePeriod time.Duration

	processingEntries cacheEntries
}

func NewIndexerDelayingQueue(name string, keyFunc func(obj interface{}) (string, error)) DelayingInterface {
	return newIndexerDelayingQueue(clock.RealClock{}, name, keyFunc)
}

func newIndexerDelayingQueue(clock clock.WithTicker, name string, keyFunc func(obj interface{}) (string, error)) *indexerDelayingQueue {
	mutex := &sync.Mutex{}
	ret := &indexerDelayingQueue{
		clock:               clock,
		keyFunc:             keyFunc,
		heartbeat:           clock.NewTicker(maxWait),
		stopCh:              make(chan struct{}),
		waitingForAddCh:     make(chan *waitFor, 1000),
		waitingForQueue:     &waitForPriorityQueue{},
		knownPrepareEntries: cacheEntries{},

		dirty:             set{},
		cond:              sync.NewCond(mutex),
		addCond:           sync.NewCond(mutex),
		metrics:           globalMetricsFactory.newQueueMetrics(name, clock),
		processingEntries: cacheEntries{},
	}

	go ret.waitingLoop()
	return ret
}

func (q *indexerDelayingQueue) Len() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return len(q.knownPrepareEntries) + len(q.processingEntries)
}

// waitEntry holds the data to push and the time it should be added
type waitEntry struct {
	data    t
	readyAt time.Time
}

// ShutDown stops the queue. After the queue drains, the returned shutdown bool
// on Get() will be true. This method may be invoked more than once.
func (q *indexerDelayingQueue) ShutDown() {
	q.stopOnce.Do(func() {
		q.shutdown()
		close(q.stopCh)
		q.heartbeat.Stop()
	})
}

func (q *indexerDelayingQueue) shutdown() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.shuttingDown = true
	q.cond.Broadcast()
	q.addCond.Broadcast()
}

// waitForProcessing waits for the worker goroutines to finish processing items
// and call Done on them.
func (q *indexerDelayingQueue) waitForProcessing() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// Ensure that we do not wait on a queue which is already empty, as that
	// could result in waiting for Done to be called on items in an empty queue
	// which has already been shut down, which will result in waiting
	// indefinitely.
	if q.processingEntries.len() == 0 {
		return
	}
	q.cond.Wait()
}

func (q *indexerDelayingQueue) setDrain(shouldDrain bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.drain = shouldDrain
}

func (q *indexerDelayingQueue) shouldDrain() bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.drain
}

// isProcessing indicates if there are still items on the work queue being
// processed. It's used to drain the work queue on an eventual shutdown.
func (q *indexerDelayingQueue) isProcessing() bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.processingEntries.len() != 0
}

func (q *indexerDelayingQueue) ShutDownWithDrain() {
	q.stopOnce.Do(func() {
		q.setDrain(true)
		q.shutdown()
		close(q.stopCh)
		q.heartbeat.Stop()
	})
	for q.isProcessing() && q.shouldDrain() {
		q.waitForProcessing()
	}
}

func (q *indexerDelayingQueue) ShuttingDown() bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	return q.shuttingDown
}

// waitingLoop runs until the workqueue is shutdown and keeps a check on the list of items to be added.
func (q *indexerDelayingQueue) waitingLoop() {
	defer utilruntime.HandleCrash()

	// Make a placeholder channel to use when there are no items in our list
	never := make(<-chan time.Time)

	// Make a timer that expires when the item at the head of the waiting queue is ready
	var nextReadyAtTimer clock.Timer

	waitingForQueue := q.waitingForQueue
	heap.Init(waitingForQueue)

	waitingEntryByData := map[t]*waitFor{}

	for {
		if q.ShuttingDown() {
			return
		}

		now := q.clock.Now()

		// Add ready entries
		for waitingForQueue.Len() > 0 {
			entry := waitingForQueue.Peek().(*waitFor)
			if entry.readyAt.After(now) {
				break
			}

			entry = heap.Pop(waitingForQueue).(*waitFor)
			q.push(entry.data.(string))
			delete(waitingEntryByData, entry.data)
		}

		// Set up a wait for the first item's readyAt (if one exists)
		nextReadyAt := never
		if waitingForQueue.Len() > 0 {
			if nextReadyAtTimer != nil {
				nextReadyAtTimer.Stop()
			}
			entry := waitingForQueue.Peek().(*waitFor)
			nextReadyAtTimer = q.clock.NewTimer(entry.readyAt.Sub(now))
			nextReadyAt = nextReadyAtTimer.C()
		}

		select {
		case <-q.stopCh:
			return

		case <-q.heartbeat.C():
			// continue the loop, which will push ready items

		case <-nextReadyAt:
			// continue the loop, which will push ready items

		case waitEntry := <-q.waitingForAddCh:
			if waitEntry.readyAt.After(q.clock.Now()) {
				insert(waitingForQueue, waitingEntryByData, waitEntry)
			} else {
				q.push(waitEntry.data.(string))
			}

			drained := false
			for !drained {
				select {
				case waitEntry := <-q.waitingForAddCh:
					if waitEntry.readyAt.After(q.clock.Now()) {
						insert(waitingForQueue, waitingEntryByData, waitEntry)
					} else {
						q.push(waitEntry.data.(string))
					}
				default:
					drained = true
				}
			}
		}
	}
}

func (q *indexerDelayingQueue) Add(item interface{}) {
	q.shouldAdd()
	q.AddAfter(item, 0)
}

func (q *indexerDelayingQueue) shouldAdd() {
	q.addCond.L.Lock()
	defer q.addCond.L.Unlock()
	for len(q.knownPrepareEntries)+len(q.processingEntries) >= queue_item_cap && !q.shuttingDown {
		q.addCond.Wait()
	}
}

func (q *indexerDelayingQueue) AddAfter(item interface{}, delay time.Duration) {
	key, err := q.keyFunc(item)
	if err != nil {
		klog.ErrorS(err, "indexerDelayingQueue generate key")
		return
	}
	readAt := q.clock.Now().Add(delay)

	needPush := func() (needPush bool) {
		q.cond.L.Lock()
		defer q.cond.L.Unlock()
		if q.shuttingDown {
			return
		}
		waitItem, exist := q.knownPrepareEntries.get(key)
		if exist {
			if o, ok := waitItem.data.(Compared); ok && o.LessOrEqual(item) {
				waitItem.data = item
			}
			if waitItem.readyAt.After(readAt) {
				waitItem.readyAt = readAt
				needPush = true
			}
			return
		}
		processItem, exist := q.processingEntries.get(key)
		if exist {
			if o, ok := processItem.data.(Compared); ok && !o.LessOrEqual(item) {
				return
			}
		}
		q.knownPrepareEntries.insert(key, &waitEntry{
			data:    item,
			readyAt: readAt,
		})
		q.metrics.add(key)
		needPush = true
		return
	}()

	if needPush {
		if readAt.Before(q.clock.Now()) {
			q.push(key)
			return
		}
		q.pushDelayQueue(key, readAt)
	}
}

func (q *indexerDelayingQueue) pushDelayQueue(key string, readyAt time.Time) {
	select {
	case <-q.stopCh:
		// unblock if ShutDown() is called
	case q.waitingForAddCh <- &waitFor{data: key, readyAt: readyAt}:
	}
}

// push marks item as needing processing.
func (q *indexerDelayingQueue) push(key string) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	if q.shuttingDown {
		return
	}

	// expired item,should be ignored.
	if item, exist := q.knownPrepareEntries.get(key); !exist {
		return
	} else if item.readyAt.After(q.clock.Now()) {
		return
	}

	if q.dirty.has(key) {
		return
	}

	q.metrics.add(key)

	q.dirty.insert(key)
	if _, exist := q.processingEntries.get(key); exist {
		return
	}

	q.queue = append(q.queue, key)
	q.cond.Signal()
}

// Get blocks until it can return an item to be processed. If shutdown = true,
// the caller should end their goroutine. You must call Done with item when you
// have finished processing it.
func (q *indexerDelayingQueue) Get() (item interface{}, shutdown bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	for len(q.queue) == 0 && !q.shuttingDown {
		q.cond.Wait()
	}
	if len(q.queue) == 0 {
		// We must be shutting down.
		return nil, true
	}

	key := q.queue[0]
	q.queue = q.queue[1:]

	entry, _ := q.knownPrepareEntries.get(key)
	q.processingEntries.insert(key, entry)
	q.dirty.delete(key)
	q.knownPrepareEntries.delete(key)
	return entry.data, false
}

// Done marks item as done processing, and if it has been marked as dirty again
// while it was being processed, it will be re-added to the queue for
// re-processing.
func (q *indexerDelayingQueue) Done(item interface{}) {
	key, err := q.keyFunc(item)
	if err != nil {
		klog.ErrorS(err, "indexerDelayingQueue generate key")
		return
	}

	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.metrics.done(key)

	q.processingEntries.delete(key)
	if q.dirty.has(key) {
		q.queue = append(q.queue, key)
		q.cond.Signal()
	} else if q.processingEntries.len() == 0 {
		q.cond.Signal()
	}
	q.addCond.Signal()
}

type cacheEntries map[string]*waitEntry

func (s cacheEntries) get(key string) (*waitEntry, bool) {
	val, exists := s[key]
	return val, exists
}

func (s cacheEntries) insert(key string, value *waitEntry) {
	s[key] = value
}

func (s cacheEntries) delete(key string) {
	delete(s, key)
}

func (s cacheEntries) len() int {
	return len(s)
}

type Compared interface {
	LessOrEqual(item interface{}) bool
}
