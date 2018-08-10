package dispatcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const numWorkersTotal int = 1000

type testJob struct {
	resultSender chan bool
}

func (job *testJob) Do() {
	job.resultSender <- true
}

func TestInitAndDestroyWorkerPool(T *testing.T) {
	assertion := assert.New(T)
	// Expect an error to be returned by getting number of available workers
	// without initializing global worker pool first
	numWorkersAvail, err := GetNumWorkersAvail()
	assertion.Equal(numWorkersAvail, 0, "Wrong number of available workers "+
		"returned give global worker pool was not initialized")
	assertion.EqualError(
		err,
		"Global worker pool was not initialized",
		"Wrong error was returned by getting number of available workers"+
			"without initializing global worker pool first",
	)
	// Expect an error to be returned by getting number of total workers
	// without initializing global worker pool first
	numWorkersInitializedTotal, err := GetNumWorkersTotal()
	assertion.Equal(numWorkersInitializedTotal, 0, "Wrong total number of workers "+
		"returned give global worker pool was not initialized")
	assertion.EqualError(
		err,
		"Global worker pool was not initialized",
		"Wrong error was returned by getting total number of workers"+
			"without initializing global worker pool first",
	)

	// Verify initialization
	InitWorkerPoolGlobal(numWorkersTotal)
	numWorkersInitialized, err := GetNumWorkersAvail()
	assertion.Nil(err, "A non empty error was returned by getting number of available workers")
	assertion.Equal(
		numWorkersInitialized,
		numWorkersTotal,
		"Total number of workers initialized was not correct",
	)
	// Expect error to be returned when initialization is called twice
	err = InitWorkerPoolGlobal(numWorkersTotal)
	assertion.EqualError(
		err,
		"Global worker pool has been initialized",
		"Wrong error returned by initializing global worker pool twice",
	)

	// Verify destroying
	DestroyWorkerPoolGlobal()
	numWorkersLeft, _ := GetNumWorkersAvail()
	assertion.Equal(
		numWorkersLeft,
		0,
		"Total number of workers initialized was not correct",
	)
	// Expect an error to be returned when destroy is called twice
	err = DestroyWorkerPoolGlobal()
	assertion.EqualError(
		err,
		"Global worker pool has been destroyed",
		"Wrong error returned by destroying global worker pool twice",
	)
}

func TestStartingDispatchers(T *testing.T) {
	assertion := assert.New(T)
	// Expect an error to be returned by creating a dispatcher without
	// initializing global worker pool
	disp, err := NewDispatcher()
	assertion.EqualError(
		err,
		"Global worker pool was not initialized",
		"Wrong error returned by creating a new dispatcher when global worker pool was not initialized",
	)
	assertion.Nil(disp, "Dispatcher was created given global worker pool was not initialized")
	InitWorkerPoolGlobal(numWorkersTotal)
	// Create one dispatcher
	disp1, err := NewDispatcher()
	assertion.Nil(err, "A non empty error was returned by creating a new dispatcher")
	numWorkersTaken := 100
	complete1, err := disp1.Start(numWorkersTaken, nil)
	assertion.Nil(err, "A non empty error was returned by starting a dispatcher")
	<-complete1
	// Expect an error to be returned by calling Start() twice
	completeErr, err := disp1.Start(numWorkersTaken, nil)
	assertion.Nil(
		completeErr,
		"A non empty channel was returned by starting a dispatcher for the second time",
	)
	assertion.EqualError(
		err,
		"Dispatcher is not in initialized state, "+
			"Start() can only be called once after a new dispatcher is created",
		"Wrong error was returned by calling Start() twice",
	)

	numWorkersLeft, _ := GetNumWorkersAvail()
	numWorkersLeftExpected := numWorkersTotal - numWorkersTaken
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"1) Number of workers left were not correct",
	)
	// Create another dispatcher
	numWorkersTaken2 := numWorkersLeftExpected
	disp2, err := NewDispatcher()
	assertion.Nil(err, "A non empty error was returned by creating a new dispatcher")
	complete2, err := disp2.Start(numWorkersTaken2, nil)
	assertion.Nil(err, "A non empty error was returned by starting a dispatcher")
	<-complete2

	numWorkersLeft, _ = GetNumWorkersAvail()
	numWorkersLeftExpected -= numWorkersTaken2
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"2) Number of workers left were not correct",
	)

	// Finalize the first dispatcher
	err = disp1.Finalize()
	assertion.Nil(err, "A non empty error was returned by finalizing a dispatcher")
	numWorkersLeft, _ = GetNumWorkersAvail()
	numWorkersLeftExpected += numWorkersTaken
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"3) Number of workers left were not correct",
	)
	// Expect an error to be returned by calling finalizing twice
	err = disp1.Finalize()
	assertion.EqualError(
		err,
		"Dispatcher is not in ready state. "+
			"Start() must be called to start listening before finalizing it",
		"Wrong error was returned by calling finalizing twice",
	)

	// Finalize the second dispatcher
	err = disp2.Finalize()
	assertion.Nil(err, "A non empty error was returned by finalizing a dispatcher")
	numWorkersLeft, _ = GetNumWorkersAvail()
	numWorkersLeftExpected += numWorkersTaken2
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"4) Number of workers left were not correct",
	)

	DestroyWorkerPoolGlobal()
}

func TestDispatchingJobs(T *testing.T) {
	assertion := assert.New(T)
	InitWorkerPoolGlobal(numWorkersTotal)
	numWorkersTaken := 100
	receiver := make(chan bool, numWorkersTaken)

	// Create one dispatcher
	disp, _ := NewDispatcher()

	// Expect an error to be returned by calling dispatch without
	// calling starting the dispatcher first
	err := disp.Dispatch(&testJob{resultSender: receiver})
	assertion.EqualError(
		err,
		"Dispatcher is not in ready state. "+
			"Start() must be called to enable dispatcher to dispatch jobs",
		"Wrong error was returned by dispatching jobs without "+
			"starting the dispatcher first",
	)

	disp.Start(numWorkersTaken, nil)

	// Dispatch jobs
	sum := 0
	for i := 0; i < numWorkersTaken; i++ {
		disp.Dispatch(&testJob{resultSender: receiver})
	}
	disp.Finalize()
	close(receiver)
	// Verify the number of jobs being done
	for range receiver {
		sum++
	}
	assertion.Equal(numWorkersTaken, sum, "Incorrect number of job being dispatched")

	DestroyWorkerPoolGlobal()
}

func TestDispatchingJobsWithDelay(T *testing.T) {
	assertion := assert.New(T)
	InitWorkerPoolGlobal(numWorkersTotal)
	numWorkersTaken := 100
	numJobs := 100
	receiver := make(chan bool, numJobs)

	// Create one dispatcher
	disp, _ := NewDispatcher()

	// Expect an error to be returned by calling dispatch with delay without
	// calling starting the dispatcher first
	err := disp.Dispatch(&testJob{resultSender: receiver})
	assertion.EqualError(
		err,
		"Dispatcher is not in ready state. "+
			"Start() must be called to enable dispatcher to dispatch jobs",
		"Wrong error was returned by dispatching jobs with delay without "+
			"starting the dispatcher first",
	)

	disp.Start(numWorkersTaken, nil)

	start := time.Now()
	// Dispatch jobs with delay
	for i := 0; i < numJobs; i++ {
		disp.DispatchWithDelay(&testJob{resultSender: receiver}, 1000)
	}
	elapse := time.Since(start)
	assertion.True(elapse >= 100000, "Job dispatching was not delayed with the correct time period")

	disp.Finalize()
	DestroyWorkerPoolGlobal()
}

func TestMultiGoroutineDispatchingJobs(T *testing.T) {
	assertion := assert.New(T)
	InitWorkerPoolGlobal(numWorkersTotal)
	numWorkersTaken1 := numWorkersTotal - 1
	numWorkersTaken2 := numWorkersTotal - 1
	numJobs1 := numWorkersTaken1 + 100
	numJobs2 := numWorkersTaken2 + 100
	// Count number of jobs done
	sum := 0
	receiver1 := make(chan bool, numJobs1)
	receiver2 := make(chan bool, numJobs2)

	go func() {
		// Create one dispatcher
		disp, _ := NewDispatcher()
		disp.Start(numWorkersTaken1, nil)
		// Dispatch jobs
		for i := 0; i < numJobs1; i++ {
			disp.Dispatch(&testJob{resultSender: receiver1})
		}
		disp.Finalize()
		close(receiver1)
	}()

	go func() {
		// Create one dispatcher
		disp, _ := NewDispatcher()
		disp.Start(numWorkersTaken2, nil)
		// Dispatch jobs
		for i := 0; i < numJobs2; i++ {
			disp.Dispatch(&testJob{resultSender: receiver2})
		}
		disp.Finalize()
		close(receiver2)
	}()

	// Verify the number of jobs being done
	for range receiver1 {
		sum++
	}
	for range receiver2 {
		sum++
	}
	assertion.Equal(numJobs1+numJobs2, sum, "Incorrect number of job being dispatched")

	DestroyWorkerPoolGlobal()
}

func TestHandlingReachLimitWarning(T *testing.T) {
	InitWorkerPoolGlobal(numWorkersTotal)

	reachLimitChan := make(chan struct{})
	reachLimitHandler := func() {
		var sig struct{}
		reachLimitChan <- sig
	}

	jobFinishReceiver1 := make(chan struct{}, 1)
	jobFinishReceiver2 := make(chan struct{}, 1)

	disp1, _ := NewDispatcher()
	disp2, _ := NewDispatcher()

	go func() {
		complete1, _ := disp1.Start(numWorkersTotal-1, nil)
		<-complete1
		// Expect to be blocked until disp1 releases workers
		complete2, _ := disp2.Start(numWorkersTotal-1, reachLimitHandler)
		<-complete2

		disp2.Finalize()
		var sig struct{}
		jobFinishReceiver2 <- sig
	}()

	select {
	case <-reachLimitChan:
		// Reach limit handler is called as expected, disp1 releases workers
		// to unblock disp2
		close(reachLimitChan)
		disp1.Finalize()
		var sig struct{}
		jobFinishReceiver1 <- sig
	case <-time.After(2 * time.Second):
		panic("Test handling reach limit warning failed, handler was not called")
	}

	<-jobFinishReceiver1
	<-jobFinishReceiver2
	close(jobFinishReceiver1)
	close(jobFinishReceiver2)
	DestroyWorkerPoolGlobal()
}
