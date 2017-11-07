package dispatcher

import (
	"fmt"
	"sync"
	"time"
)

const (
	initialized int = 0
	ready       int = 1
	listening   int = 2
)

var workerPoolGlobal chan *_Worker
var isInitialized bool
var mutex sync.Mutex

// Job - A job can be given to a dispatcher to be allocated
// to a worker
type Job interface {
	Do(worker Worker)
}

type _QuitJob struct {
	quitSender chan bool
}

func (quitJob *_QuitJob) Do(worker Worker) {
	if worker.IsActive() {
		worker.recycle()
	}
	// Tell the dispatcher that the worker has been recycled
	quitJob.quitSender <- true
}

// Worker - A worker can only be created by the global worker pool
// to execute jobs
type Worker interface {
	IsActive() bool
	register(workerPool chan *_Worker)
	recycle()
}

type _Worker struct {
	stopped     bool
	jobListener chan Job
}

func (worker *_Worker) IsActive() bool {
	return !worker.stopped
}

func (worker *_Worker) recycle() {
	close(worker.jobListener)
	worker.stopped = true
	// Push back to the global worker pool
	workerPoolGlobal <- worker
}

func (worker *_Worker) register(workerPool chan *_Worker) {
	worker.stopped = false
	worker.jobListener = make(chan Job)

	for !worker.stopped {
		// Register itself in a dispatcher worker pool
		workerPool <- worker

		// Start listening to job queue
		job := <-worker.jobListener
		job.Do(worker)
	}
}

// Dispatcher - A dispatcher takes workers from the global worker pool
// and dispatch jobs. Multiple dispatchers can be created but only a
// limited number of workers can be used.
type Dispatcher interface {
	Spawn(numWorkers int)
	Start()
	Dispatch(job Job)
	Finalize()
}

type _Dispatcher struct {
	workerPool   chan *_Worker
	jobChan      chan Job
	quitReceiver chan bool
	numWorkers   int
	closed       bool
	timeInterval time.Duration
	state        int
}

func (dispatcher *_Dispatcher) Dispatch(job Job) {
	dispatcher.jobChan <- job
}

// Spawn - Block until the given number of workers are obtained from the
// global worker pool
func (dispatcher *_Dispatcher) Spawn(numWorkers int) {
	numWorkersTotal := cap(workerPoolGlobal)
	if numWorkers > numWorkersTotal {
		panic(`Cannot spawn more workers than the number of created
			workers in the global worker pool`)
	}

	dispatcher.workerPool = make(chan *_Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		// Take a worker from the global worker pool
		worker := <-workerPoolGlobal

		// Register the worker into its local worker pool
		go worker.register(dispatcher.workerPool)
		dispatcher.numWorkers++
	}

	dispatcher.state = ready
}

// Start - Start another goroutine listening to jobs
func (dispatcher *_Dispatcher) Start() {
	if dispatcher.state != ready {
		fmt.Println(`dispatcher.Spawn() must be called to obtain
			workers before starting it`)
		return
	}

	go func() {
		for !dispatcher.closed {
			worker := <-dispatcher.workerPool
			job := <-dispatcher.jobChan
			worker.jobListener <- job
			if dispatcher.timeInterval > 0 {
				// Pause worker for a period of time as specified
				time.Sleep(dispatcher.timeInterval * time.Millisecond)
			}
		}

		// Stop receiving more jobs
		close(dispatcher.jobChan)
		// Stop recycle workers
		close(dispatcher.workerPool)
	}()

	dispatcher.state = listening
}

// Finalize - Block until all worker threads are popped out
// of the pool and signaled with a quit command.
func (dispatcher *_Dispatcher) Finalize() {
	if dispatcher.state != listening {
		fmt.Println(`dispatcher.Start() must be called to start
			listening before finalizing it`)
		return
	}

	// Send a quit job to each worker
	for i := 0; i < dispatcher.numWorkers; i++ {
		dispatcher.Dispatch(&_QuitJob{quitSender: dispatcher.quitReceiver})
		// Block until the worker is recycled
		<-dispatcher.quitReceiver
	}
	// Stop listening
	dispatcher.closed = true
	dispatcher.state = initialized
}

// NewDispatcher - Return a new job dispatcher
func NewDispatcher(timeInterval time.Duration) Dispatcher {
	if !isInitialized {
		panic(`Please call InitWorkerPoolGlobal() before creating
			new dispatchers`)
	}

	return &_Dispatcher{
		jobChan:      make(chan Job),
		quitReceiver: make(chan bool),
		timeInterval: timeInterval,
	}
}

// InitWorkerPoolGlobal - Initialize the global worker pool safely and
// spawn a given number of workers
func InitWorkerPoolGlobal(numWorkersTotal int) {
	mutex.Lock()
	defer mutex.Unlock()

	if !isInitialized {
		workerPoolGlobal = make(chan *_Worker, numWorkersTotal)
		for i := 0; i < numWorkersTotal; i++ {
			// Create a new worker
			worker := _Worker{stopped: true}
			workerPoolGlobal <- &worker
		}
		isInitialized = true
	}

	fmt.Println("Global worker pool has been initialized")
}

// DestroyWorkerPoolGlobal - Drain and close the global worker pool safely,
// which will block until all workers are popped out (all dispatchers are finished)
func DestroyWorkerPoolGlobal() {
	mutex.Lock()
	defer mutex.Unlock()

	numWorkersTotal := cap(workerPoolGlobal)
	if isInitialized {
		for i := 0; i < numWorkersTotal; i++ {
			<-workerPoolGlobal
		}
		close(workerPoolGlobal)
		isInitialized = false
	}

	fmt.Println("Global worker pool has been destroyed")
}

// GetNumWorkersAvail - Get number of workers can be used by
// a new dispatcher to spawn
func GetNumWorkersAvail() int {
	return len(workerPoolGlobal)
}
