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
	// Finalize blocks until all jobs dispatched are finished and all workers are returned
	// to worker pool. Note a finalized dispatcher cannot be reused.
	Finalize()
	// GetNumWorkersAvail returns the number of workers availalbe for tasks at the time
	// it is called.
	GetNumWorkersAvail() int
	// GetTotalNumWorkers returns total number of workers created by the dispatcher.
	GetTotalNumWorkers() int
}

type _Dispatcher struct {
	sync.Mutex
	wg          *sync.WaitGroup
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

func (dispatcher *_Dispatcher) Finalize() {
	dispatcher.Lock()
	defer dispatcher.Unlock()

	// Wait for all jobs to be dispatched (to ensure wg.Add() happens before wg.Wait())
	finishSignReceiver := make(chan _FinishSignal)
	dispatcher.jobListener <- _DelayedJob{job: &_EmptyJob{finishSignReceiver: finishSignReceiver}}
	<-finishSignReceiver
	// Wait for all workers to finish their jobs
	dispatcher.wg.Wait()
	// Stop job dispatching loop
	close(dispatcher.jobListener)
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

	wg := &sync.WaitGroup{}
	dispatcher := &_Dispatcher{wg: wg}
	dispatcher.jobListener = make(chan _DelayedJob)
	dispatcher.workerPool = make(chan *_Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		// Register the worker in the dispatcher
		dispatcher.workerPool <- &_Worker{wg: wg}
	}

	go func() {
		for delayedJob := range dispatcher.jobListener {
			time.Sleep(delayedJob.delayPeriod)
			worker := <-dispatcher.workerPool
			worker.wg.Add(1)
			go func(job Job, worker *_Worker) {
				job.Do()
				// Return it back to the worker pool
				dispatcher.workerPool <- worker
				worker.wg.Done()
			}(delayedJob.job, worker)
		}
	}()

	return dispatcher, nil
}
