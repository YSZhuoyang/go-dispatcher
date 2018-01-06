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

// Dispatcher - A dispatcher takes workers from the global worker pool
// and dispatch jobs. Multiple dispatchers can be created but only a
// limited number of workers can be used.
type Dispatcher interface {
	Spawn(numWorkers int)
	Start()
	Dispatch(job Job)
	DispatchWithDelay(job Job, delayPeriod time.Duration)
	Finalize()
}

type _Dispatcher struct {
	workerPool chan *_Worker
	jobChan    chan Job
	wg         sync.WaitGroup
	numWorkers int
	state      int
}

// Spawn - Block until the given number of workers are obtained from the
// global worker pool. Note it can only be called by one goroutine at a time.
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

	dispatcher.jobChan = make(chan Job)
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

// Start - Start another goroutine listening to jobs
func (dispatcher *_Dispatcher) Start() {
	if dispatcher.state != isReady {
		panic(`Dispatcher is not ready. Spawn() must be called to obtain
			workers before starting it`)
	}

	go func() {
		for job := range dispatcher.jobChan {
			worker := <-dispatcher.workerPool
			worker.jobListener <- job
		}

		// Stop recycling workers
		close(dispatcher.workerPool)
	}()

	dispatcher.state = isListening
}

// Dispatch - Dispatching a job to its job listener. The job being dispatched
// will be executed asynchronously
func (dispatcher *_Dispatcher) Dispatch(job Job) {
	if dispatcher.state != isListening {
		panic(`Dispatcher is not in listening state. Start() must be called
			to enable dispatcher to dispatch jobs`)
	}

	dispatcher.jobChan <- job
}

// DispatchWithDelay - Similar to Dispatch() except job dispatching is delayed for
// a specified period of time
func (dispatcher *_Dispatcher) DispatchWithDelay(job Job, delayPeriod time.Duration) {
	if dispatcher.state != isListening {
		panic(`Dispatcher is not in listening state. Start() must be called
			to enable dispatcher to dispatch jobs`)
	}

	time.Sleep(delayPeriod)
	dispatcher.jobChan <- job
}

// Finalize - Block until all worker threads are popped out
// of the pool and signaled with a quit command.
func (dispatcher *_Dispatcher) Finalize() {
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

// NewDispatcher - Return a new job dispatcher
func NewDispatcher() Dispatcher {
	if !isGlobalWorkerPoolInitialized {
		panic(`Please call InitWorkerPoolGlobal() before creating
			new dispatchers`)
	}

	return &_Dispatcher{wg: sync.WaitGroup{}}
}

// InitWorkerPoolGlobal - Initialize the global worker pool safely and
// spawn a given number of workers
func InitWorkerPoolGlobal(numWorkersTotal int) {
	mutex.Lock()
	defer mutex.Unlock()

	if !isGlobalWorkerPoolInitialized {
		workerPoolGlobal = make(chan *_Worker, numWorkersTotal)
		for i := 0; i < numWorkersTotal; i++ {
			// Create a new worker
			worker := _Worker{stopped: true}
			workerPoolGlobal <- &worker
		}
		isGlobalWorkerPoolInitialized = true
	} else {
		fmt.Println("Global worker pool has been initialized before")
	}
}

// DestroyWorkerPoolGlobal - Drain and close the global worker pool safely,
// which will block until all workers are popped out (all dispatchers are finished)
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

// GetNumWorkersAvail - Get number of workers can be used by
// a new dispatcher to spawn
func GetNumWorkersAvail() int {
	return len(workerPoolGlobal)
}
