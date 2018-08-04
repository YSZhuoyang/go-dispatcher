package dispatcher

var workerPoolGlobal chan *_Worker
var isGlobalWorkerPoolInitialized bool

// InitWorkerPoolGlobal initializes the global worker pool safely and
// creates a given number of workers
func InitWorkerPoolGlobal(numWorkersTotal int) error {
	mutex.Lock()
	defer mutex.Unlock()

	if !isGlobalWorkerPoolInitialized {
		workerPoolGlobal = make(chan *_Worker, numWorkersTotal)
		for i := 0; i < numWorkersTotal; i++ {
			workerPoolGlobal <- &_Worker{}
		}
		isGlobalWorkerPoolInitialized = true

		return nil
	}

	return newError("Global worker pool has been initialized before")
}

// DestroyWorkerPoolGlobal drains and closes the global worker pool safely,
// which blocks until all workers are popped out (all dispatchers are finished)
func DestroyWorkerPoolGlobal() error {
	mutex.Lock()
	defer mutex.Unlock()

	numWorkersTotal := GetNumWorkersTotal()
	if isGlobalWorkerPoolInitialized {
		for i := 0; i < numWorkersTotal; i++ {
			<-workerPoolGlobal
		}
		close(workerPoolGlobal)
		isGlobalWorkerPoolInitialized = false

		return nil
	}

	return newError("Global worker pool has been destroyed before")
}

// GetNumWorkersAvail returns the number of workers that can be allocated to
// a new dispatcher
func GetNumWorkersAvail() int {
	return len(workerPoolGlobal)
}

// GetNumWorkersTotal returns total number of workers created by the global worker pool
func GetNumWorkersTotal() int {
	return cap(workerPoolGlobal)
}
