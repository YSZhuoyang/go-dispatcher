package dispatcher

import "time"

// Job defines a task, which is given to a dispatcher to be executed
// by a worker with a separate goroutine
type Job interface {
	Do()
}

type _DelayedJob struct {
	job         Job
	delayPeriod time.Duration
}

type _QuitJob struct {
	quitSignChan chan bool
}

func (quitJob *_QuitJob) Do() {
	// Tell the dispatcher that all jobs have been dispatched
	quitJob.quitSignChan <- true
}
