package dispatcher

import "sync"
import "time"

// Job - A job can be given to a dispatcher to be allocated
// to a worker
type Job interface {
	Do(worker Worker)
}

type delayedJob struct {
	job         Job
	delayPeriod time.Duration
}

type quitJob struct {
	wg *sync.WaitGroup
}

func (quitJob *quitJob) Do(worker Worker) {
	worker.recycle()
	// Tell the dispatcher that the worker has been recycled
	quitJob.wg.Done()
}
