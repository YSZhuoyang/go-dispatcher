package dispatcher

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
