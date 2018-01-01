package dispatcher

import "sync"

// Job - A job can be given to a dispatcher to be allocated
// to a worker
type Job interface {
	Do(worker Worker)
}

type quitJob struct {
	wg *sync.WaitGroup
}

func (quitJob *quitJob) Do(worker Worker) {
	if worker.IsActive() {
		worker.recycle()
	}
	// Tell the dispatcher that the worker has been recycled
	quitJob.wg.Done()
}
