package dispatcher

// Worker is registered in a worker pool to listen to jobs and execute them.
// It can only be created / destroyed by the global worker pool.
type _Worker struct {
	isActive bool
}

func (worker *_Worker) do(task func()) {
	task()
}

// recycle is called when a quiteJob is executed,
// to set isActive to false to stop listening to
// new jobs
func (worker *_Worker) recycle() {
	worker.isActive = false
	// Return it back to the global worker pool
	workerPoolGlobal <- worker
}
