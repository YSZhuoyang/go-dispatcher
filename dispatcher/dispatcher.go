// Package dispatcher provides capabilities of limiting the total number of goroutines
// and easily managing the dependency of concurrent executions.
package dispatcher

import (
	"sync"
	"time"
)

// Dispatcher creates workers registered in a worker pool and dispatches
// jobs to them.
type Dispatcher interface {
	// Dispatch gives a job to a worker at a time, and blocks until at least one worker
	// becomes available. Each job dispatched is handled by a separate goroutine.
	Dispatch(job Job)
	// DispatchWithDelay behaves similarly to Dispatch, except it is delayed for a given
	// period of time (in nanoseconds) before the job is allocated to a worker.
	DispatchWithDelay(job Job, delayPeriod time.Duration) error
	// Await blocks until all jobs dispatched are finished and all workers are returned
	// to the global worker pool. Note that it can only be called once per dispatcher.
	Await()
	// GetNumWorkersAvail returns the number of workers that can be allocated to
	// a new dispatcher.
	GetNumWorkersAvail() int
	// GetTotalNumWorkers returns total number of workers created by the dispatcher.
	GetTotalNumWorkers() int
}

type _Dispatcher struct {
	sync.Mutex
	workerPool  chan *_Worker
	jobListener chan _DelayedJob
}

func (dispatcher *_Dispatcher) Dispatch(job Job) {
	dispatcher.Lock()
	defer dispatcher.Unlock()

	dispatcher.jobListener <- _DelayedJob{job: job}
}

func (dispatcher *_Dispatcher) DispatchWithDelay(job Job, delayPeriod time.Duration) error {
	if delayPeriod <= 0 {
		return newError("Invalid delay period")
	}

	dispatcher.Lock()
	defer dispatcher.Unlock()

	dispatcher.jobListener <- _DelayedJob{job: job, delayPeriod: delayPeriod}
	return nil
}

func (dispatcher *_Dispatcher) Await() {
	dispatcher.Lock()
	defer dispatcher.Unlock()

	// Wait for all jobs to be dispatched
	finishSignReceiver := make(chan _FinishSignal)
	dispatcher.jobListener <- _DelayedJob{job: &_EmptyJob{finishSignReceiver: finishSignReceiver}}
	<-finishSignReceiver
	// Wait for all workers to finish their jobs
	numWorkers := cap(dispatcher.workerPool)
	for i := 0; i < numWorkers; i++ {
		<-dispatcher.workerPool
	}
	for i := 0; i < numWorkers; i++ {
		dispatcher.workerPool <- &_Worker{}
	}
}

func (dispatcher *_Dispatcher) GetNumWorkersAvail() int {
	return len(dispatcher.workerPool)
}

func (dispatcher *_Dispatcher) GetTotalNumWorkers() int {
	return cap(dispatcher.workerPool)
}

// NewDispatcher returns a new job dispatcher with a worker pool
// initialized with given number of workers.
func NewDispatcher(numWorkers int) (Dispatcher, error) {
	if numWorkers <= 0 {
		return nil, newError("Invalid number of workers given to create a new dispatcher")
	}

	dispatcher := &_Dispatcher{}
	dispatcher.jobListener = make(chan _DelayedJob)
	dispatcher.workerPool = make(chan *_Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		// Register the worker in the dispatcher
		dispatcher.workerPool <- &_Worker{}
	}

	go func() {
		for delayedJob := range dispatcher.jobListener {
			time.Sleep(delayedJob.delayPeriod)
			worker := <-dispatcher.workerPool
			go func(job Job, worker *_Worker) {
				job.Do()
				// Return it back to the worker pool
				dispatcher.workerPool <- worker
			}(delayedJob.job, worker)
		}
	}()

	return dispatcher, nil
}
