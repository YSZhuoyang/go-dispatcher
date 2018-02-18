// Package dispatcher provides capabilities of limiting the total number of goroutines
// and easily managing the dependency of concurrent executions.
package dispatcher

import (
	"fmt"
	"sync"
	"time"
)

const (
	isInitialized int = 0
	isReady       int = 1
	isListening   int = 2
	isFinalized   int = 3
)

var workerPoolGlobal chan *_Worker
var isGlobalWorkerPoolInitialized bool
var mutex sync.Mutex

// Dispatcher takes workers from the global worker pool and dispatches
// jobs to workers. Multiple dispatchers can be created but only a limited
// number of workers can be used.
type Dispatcher interface {
	// Spawn takes a given number of workers from the global worker pool, and registers
	// them in the dispatcher worker pool. Note that it blocks other dispatchers from
	// calling it, and can only be called once per dispatcher.
	Spawn(numWorkers int)
	// Start a for loop receving new jobs dispatched and giving them to any workers
	// available. Note that it can only be called once per dispatcher.
	Start()
	// Dispatch gives a job to a worker at a time, and blocks until at least one worker
	// becomes available. Each job dispatched is handled by a separate goroutine. Note
	// that it can only be called once per dispatcher.
	Dispatch(job Job)
	// DispatchWithDelay behaves similarly to Dispatch, except it is delayed for a given
	// period of time before the job is allocated to a worker.
	DispatchWithDelay(job Job, delayPeriod time.Duration)
	// Finalize blocks until all jobs dispatched are finished and all workers are returned
	// to the global worker pool. Note that it can only be called once per dispatcher.
	Finalize()
}

type _Dispatcher struct {
	workerPool chan *_Worker
	jobChan    chan delayedJob
	wg         sync.WaitGroup
	mutex      sync.Mutex
	numWorkers int
	state      int
}

func (dispatcher *_Dispatcher) Spawn(numWorkers int) {
	mutex.Lock()
	defer mutex.Unlock()

	if dispatcher.state != isInitialized {
		panic(`Dispatcher is not in initialized state, Spawn() can only be called once
			after creating a new dispatcher`)
	}

	numWorkersTotal := cap(workerPoolGlobal)
	if numWorkers > numWorkersTotal {
		panic(`Cannot spawn more workers than the number of created
			workers in the global worker pool`)
	}

	dispatcher.jobChan = make(chan delayedJob)
	dispatcher.workerPool = make(chan *_Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		// Take a worker from the global worker pool
		worker := <-workerPoolGlobal

		// Register the worker into its local worker pool
		go worker.register(dispatcher.workerPool)
		dispatcher.numWorkers++
	}

	dispatcher.state = isReady
}

func (dispatcher *_Dispatcher) Start() {
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if dispatcher.state != isReady {
		panic(`Dispatcher is not ready. Spawn() must be called to obtain
			workers before starting it`)
	}

	go func() {
		for delayedJob := range dispatcher.jobChan {
			time.Sleep(delayedJob.delayPeriod)
			worker := <-dispatcher.workerPool
			worker.jobListener <- delayedJob.job
		}

		// Stop recycling workers
		close(dispatcher.workerPool)
	}()

	dispatcher.state = isListening
}

func (dispatcher *_Dispatcher) Dispatch(job Job) {
	if dispatcher.state != isListening {
		panic(`Dispatcher is not in listening state. Start() must be called
			to enable dispatcher to dispatch jobs`)
	}

	dispatcher.jobChan <- delayedJob{job: job}
}

func (dispatcher *_Dispatcher) DispatchWithDelay(job Job, delayPeriod time.Duration) {
	if dispatcher.state != isListening {
		panic(`Dispatcher is not in listening state. Start() must be called
			to enable dispatcher to dispatch jobs`)
	}

	dispatcher.jobChan <- delayedJob{job: job, delayPeriod: delayPeriod}
}

func (dispatcher *_Dispatcher) Finalize() {
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if dispatcher.state != isListening {
		panic(`Dispatcher is not in listening state. Start() must be called to start
			listening before finalizing it`)
	}

	dispatcher.wg.Add(dispatcher.numWorkers)
	// Send a quit job to each worker
	for i := 0; i < dispatcher.numWorkers; i++ {
		dispatcher.Dispatch(&quitJob{wg: &dispatcher.wg})
	}
	// Block until all workers are recycled
	dispatcher.wg.Wait()
	// Stop receiving more jobs
	close(dispatcher.jobChan)
	// Stop listening
	dispatcher.state = isFinalized
}

// NewDispatcher returns a new job dispatcher.
func NewDispatcher() Dispatcher {
	if !isGlobalWorkerPoolInitialized {
		panic(`Please call InitWorkerPoolGlobal() before creating
			new dispatchers`)
	}

	return &_Dispatcher{wg: sync.WaitGroup{}}
}

// InitWorkerPoolGlobal initializes the global worker pool safely and
// creates a given number of workers
func InitWorkerPoolGlobal(numWorkersTotal int) {
	mutex.Lock()
	defer mutex.Unlock()

	if !isGlobalWorkerPoolInitialized {
		workerPoolGlobal = make(chan *_Worker, numWorkersTotal)
		for i := 0; i < numWorkersTotal; i++ {
			workerPoolGlobal <- &_Worker{}
		}
		isGlobalWorkerPoolInitialized = true
	} else {
		fmt.Println("Global worker pool has been initialized before")
	}
}

// DestroyWorkerPoolGlobal drains and closes the global worker pool safely,
// which blocks until all workers are popped out (all dispatchers are finished)
func DestroyWorkerPoolGlobal() {
	mutex.Lock()
	defer mutex.Unlock()

	numWorkersTotal := cap(workerPoolGlobal)
	if isGlobalWorkerPoolInitialized {
		for i := 0; i < numWorkersTotal; i++ {
			<-workerPoolGlobal
		}
		close(workerPoolGlobal)
		isGlobalWorkerPoolInitialized = false
	} else {
		fmt.Println("Global worker pool has been destroyed before")
	}
}

// GetNumWorkersAvail returns the number of workers that can be allocated to
// a new dispatcher
func GetNumWorkersAvail() int {
	return len(workerPoolGlobal)
}
