package dispatcher_test

import (
	"testing"
	"time"

	"github.com/YSZhuoyang/go-dispatcher/dispatcher"
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
	// Verify initialization
	dispatcher.InitWorkerPoolGlobal(numWorkersTotal)
	numWorkersInitialized := dispatcher.GetNumWorkersAvail()
	assertion.Equal(
		numWorkersInitialized,
		numWorkersTotal,
		"Total number of workers initialized was not correct",
	)
	// Verify destroying
	dispatcher.DestroyWorkerPoolGlobal()
	numWorkersLeft := dispatcher.GetNumWorkersAvail()
	assertion.Equal(
		numWorkersLeft,
		0,
		"Total number of workers initialized was not correct",
	)
}

func TestStartingDispatchers(T *testing.T) {
	assertion := assert.New(T)
	dispatcher.InitWorkerPoolGlobal(numWorkersTotal)
	// Create one dispatcher
	disp, _ := dispatcher.NewDispatcher(nil)
	numWorkersTaken := 100
	disp.Start(numWorkersTaken)
	numWorkersLeft := dispatcher.GetNumWorkersAvail()
	numWorkersLeftExpected := numWorkersTotal - numWorkersTaken
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"1) Number of workers left were not correct",
	)
	// Create another dispatcher
	numWorkersTaken2 := numWorkersLeftExpected
	disp2, _ := dispatcher.NewDispatcher(nil)
	disp2.Start(numWorkersTaken2)
	numWorkersLeft = dispatcher.GetNumWorkersAvail()
	numWorkersLeftExpected -= numWorkersTaken2
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"2) Number of workers left were not correct",
	)
	// Finalize the first dispatcher
	disp.Finalize()
	numWorkersLeft = dispatcher.GetNumWorkersAvail()
	numWorkersLeftExpected += numWorkersTaken
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"3) Number of workers left were not correct",
	)
	// Finalize the second dispatcher
	disp2.Finalize()
	numWorkersLeft = dispatcher.GetNumWorkersAvail()
	numWorkersLeftExpected += numWorkersTaken2
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"4) Number of workers left were not correct",
	)

	dispatcher.DestroyWorkerPoolGlobal()
}

func TestDispatchingJobs(T *testing.T) {
	assertion := assert.New(T)
	dispatcher.InitWorkerPoolGlobal(numWorkersTotal)
	// Create one dispatcher
	numWorkersTaken := 100
	disp, _ := dispatcher.NewDispatcher(nil)
	disp.Start(numWorkersTaken)
	// Dispatch jobs
	sum := 0
	receiver := make(chan bool, numWorkersTaken)
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

	dispatcher.DestroyWorkerPoolGlobal()
}

func TestDispatchingJobsWithDelay(T *testing.T) {
	assertion := assert.New(T)
	dispatcher.InitWorkerPoolGlobal(numWorkersTotal)
	// Create one dispatcher
	numWorkersTaken := 100
	numJobs := 100
	disp, _ := dispatcher.NewDispatcher(nil)
	disp.Start(numWorkersTaken)

	receiver := make(chan bool, numJobs)
	start := time.Now()
	// Dispatch jobs with delay
	for i := 0; i < numJobs; i++ {
		disp.DispatchWithDelay(&testJob{resultSender: receiver}, 1000)
	}
	elapse := time.Since(start)
	assertion.True(elapse >= 100000, "Job dispatching was not delayed with the correct time period")

	disp.Finalize()
	dispatcher.DestroyWorkerPoolGlobal()
}

func TestMultiGoroutineDispatchingJobs(T *testing.T) {
	assertion := assert.New(T)
	dispatcher.InitWorkerPoolGlobal(numWorkersTotal)
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
		disp, _ := dispatcher.NewDispatcher(nil)
		disp.Start(numWorkersTaken1)
		// Dispatch jobs
		for i := 0; i < numJobs1; i++ {
			disp.Dispatch(&testJob{resultSender: receiver1})
		}
		disp.Finalize()
		close(receiver1)
	}()

	go func() {
		// Create one dispatcher
		disp, _ := dispatcher.NewDispatcher(nil)
		disp.Start(numWorkersTaken2)
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

	dispatcher.DestroyWorkerPoolGlobal()
}

func TestHandlingReachLimitWarning(T *testing.T) {
	dispatcher.InitWorkerPoolGlobal(numWorkersTotal)

	reachLimitChan := make(chan struct{})
	reachLimitHandler := func() {
		var sig struct{}
		reachLimitChan <- sig
	}

	jobFinishReceiver1 := make(chan struct{}, 1)
	jobFinishReceiver2 := make(chan struct{}, 1)

	disp1, _ := dispatcher.NewDispatcher(reachLimitHandler)
	disp2, _ := dispatcher.NewDispatcher(reachLimitHandler)

	go func() {
		disp1.Start(numWorkersTotal - 1)
		// Expect to be blocked until disp1 releases workers
		disp2.Start(numWorkersTotal - 1)

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
	dispatcher.DestroyWorkerPoolGlobal()
}
