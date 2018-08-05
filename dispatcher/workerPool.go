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

	return newError("Global worker pool has been initialized")
}

// DestroyWorkerPoolGlobal drains and closes the global worker pool safely,
// which blocks until all workers are popped out (all dispatchers are finished)
func DestroyWorkerPoolGlobal() error {
	mutex.Lock()
	defer mutex.Unlock()

	if isGlobalWorkerPoolInitialized {
		numWorkersTotal, _ := GetNumWorkersTotal()
		for i := 0; i < numWorkersTotal; i++ {
			<-workerPoolGlobal
		}
		close(workerPoolGlobal)
		isGlobalWorkerPoolInitialized = false

		return nil
	}

	return newError("Global worker pool has been destroyed")
}

// GetNumWorkersAvail returns the number of workers that can be allocated to
// a new dispatcher
func GetNumWorkersAvail() (int, error) {
	if !isGlobalWorkerPoolInitialized {
		return 0, newError("Global worker pool was not initialized")
	}

	return len(workerPoolGlobal), nil
}

// GetNumWorkersTotal returns total number of workers created by the global worker pool
func GetNumWorkersTotal() (int, error) {
	if !isGlobalWorkerPoolInitialized {
		return 0, newError("Global worker pool was not initialized")
	}

	return cap(workerPoolGlobal), nil
}
