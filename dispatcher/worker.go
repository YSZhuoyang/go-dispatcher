package dispatcher

// Worker is registered in a worker pool to listen to jobs and execute them.
// It can only be created / destroyed by the global worker pool.
type Worker interface {
	register(workerPool chan *_Worker)
	recycle()
}

type _Worker struct {
	isActive    bool
	jobListener chan Job
}

// recycle is called when a quiteJob is executed,
// to set isActive to false to stop listening to
// new jobs
func (worker *_Worker) recycle() {
	close(worker.jobListener)
	worker.isActive = false
	// Push back to the global worker pool
	workerPoolGlobal <- worker
}

func (worker *_Worker) register(workerPool chan *_Worker) {
	worker.isActive = true
	worker.jobListener = make(chan Job)

	for worker.isActive {
		// Register itself in a dispatcher worker pool
		workerPool <- worker

		// Start listening to job queue
		job := <-worker.jobListener
		job.Do(worker)
	}
}
