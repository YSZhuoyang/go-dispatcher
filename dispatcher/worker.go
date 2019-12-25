package dispatcher

import "sync"

// Worker is registered in a worker pool waiting for jobs and execute them.
type _Worker struct {
	wg *sync.WaitGroup
}
