// Package dispatcher provides capabilities of limiting the total number of goroutines
// and easily managing the dependency of concurrent executions.
package dispatcher

import (
	"log"
	"sync"
	"time"
)

const (
	isInitialized int = 0
	isReady       int = 1
	isFinalized   int = 2
)

var mutex sync.Mutex

// Dispatcher takes workers from the global worker pool and dispatches
// jobs to workers. Multiple dispatchers can be created but only a limited
// number of workers can be used.
type Dispatcher interface {
	// Start takes a given number of workers from the global worker pool, and registers
	// them in the dispatcher worker pool. It starts a for loop waiting for new jobs
	// dispatched and giving them to any workers available. A receiver channel will be
	// returned which will return an empty struct signal indicating the given number
	// of workers are all obtained from global worker pool. Note that it blocks other
	// dispatchers from calling it, and can only be called once per dispatcher.
	Start(numWorkers int, reachLimitHandler func()) (chan struct{}, error)
	// Dispatch gives a job to a worker at a time, and blocks until at least one worker
	// becomes available. Each job dispatched is handled by a separate goroutine.
	Dispatch(job Job) error
	// DispatchWithDelay behaves similarly to Dispatch, except it is delayed for a given
	// period of time before the job is allocated to a worker.
	DispatchWithDelay(job Job, delayPeriod time.Duration) error
	// Finalize blocks until all jobs dispatched are finished and all workers are returned
	// to the global worker pool. Note that it can only be called once per dispatcher.
	Finalize() error
}

type _Dispatcher struct {
	workerPool  chan *_Worker
	jobListener chan _DelayedJob
	mutex       sync.Mutex
	numWorkers  int
	state       int
}

func (dispatcher *_Dispatcher) Start(numWorkers int, reachLimitHandler func()) (chan struct{}, error) {
	// Block other dispatchers from getting workers
	mutex.Lock()
	// Block this dispatcher from dispatching jobs
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if dispatcher.state != isInitialized {
		// Unblock other dispatchers from getting workers
		mutex.Unlock()
		return nil, newError("Dispatcher is not in initialized state, " +
			"Start() can only be called once after a new dispatcher is created")
	}

	if numWorkers > GetNumWorkersTotal() {
		// Unblock other dispatchers from getting workers
		mutex.Unlock()
		return nil, newError("Cannot obtain more workers than the number of created " +
			"workers in the global worker pool")
	}

	if numWorkers > GetNumWorkersAvail() {
		if reachLimitHandler != nil {
			reachLimitHandler()
		} else {
			log.Print("Not enough workers available at this moment")
		}
	}

	dispatcher.jobListener = make(chan _DelayedJob)
	dispatcher.workerPool = make(chan *_Worker, numWorkers)
	workersReceivedSigChan := make(chan struct{}, 1)
	go func() {
		for i := 0; i < numWorkers; i++ {
			// Take a worker from the global worker pool
			worker := <-workerPoolGlobal

			// Register the worker in the dispatcher
			worker.isActive = true
			dispatcher.workerPool <- worker
			dispatcher.numWorkers++
		}

		var sig struct{}
		workersReceivedSigChan <- sig
		// Unblock other dispatchers from getting workers
		mutex.Unlock()
	}()

	go func() {
		for delayedJob := range dispatcher.jobListener {
			time.Sleep(delayedJob.delayPeriod)
			worker := <-dispatcher.workerPool
			go func(job Job, worker *_Worker) {
				job.Do()
				// Return it back to the dispatcher worker pool
				dispatcher.workerPool <- worker
			}(delayedJob.job, worker)
		}

		// Stop recycling workers
		close(dispatcher.workerPool)
	}()

	dispatcher.state = isReady

	return workersReceivedSigChan, nil
}

func (dispatcher *_Dispatcher) Dispatch(job Job) error {
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if dispatcher.state != isReady {
		return newError("Dispatcher is not in ready state. " +
			"Start() must be called to enable dispatcher to dispatch jobs")
	}

	dispatcher.jobListener <- _DelayedJob{job: job}
	return nil
}

func (dispatcher *_Dispatcher) DispatchWithDelay(job Job, delayPeriod time.Duration) error {
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if dispatcher.state != isReady {
		return newError("Dispatcher is not in ready state. " +
			"Start() must be called to enable dispatcher to dispatch jobs")
	}

	dispatcher.jobListener <- _DelayedJob{job: job, delayPeriod: delayPeriod}
	return nil
}

func (dispatcher *_Dispatcher) Finalize() error {
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if dispatcher.state != isReady {
		return newError("Dispatcher is not in ready state. " +
			"Start() must be called to start listening before finalizing it")
	}

	quitSignChan := make(chan _QuitSignal)
	// Send a quit quit signal after all tasks are dispatched
	dispatcher.jobListener <- _DelayedJob{job: &_QuitJob{quitSignChan: quitSignChan}}
	<-quitSignChan
	numWorkers := cap(dispatcher.workerPool)
	// Start recycling workers
	for i := 0; i < numWorkers; i++ {
		worker := <-dispatcher.workerPool
		worker.recycle()
	}
	// Stop receiving more jobs
	close(dispatcher.jobListener)
	// Stop listening
	dispatcher.state = isFinalized
	return nil
}

// NewDispatcher returns a new job dispatcher.
func NewDispatcher() (Dispatcher, error) {
	if !isGlobalWorkerPoolInitialized {
		return nil, newError("Global worker pool was not initialized")
	}

	return &_Dispatcher{}, nil
}
