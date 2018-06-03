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
	disp := dispatcher.NewDispatcher()
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
	disp2 := dispatcher.NewDispatcher()
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
	disp := dispatcher.NewDispatcher()
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
	disp := dispatcher.NewDispatcher()
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
	numWorkersTakenTotal := numWorkersTotal + 900
	numWorkersTaken1 := 950
	numWorkersTaken2 := numWorkersTakenTotal - numWorkersTaken1
	numJobs1 := numWorkersTaken1 + 100
	numJobs2 := numWorkersTaken2 + 100
	// Count number of jobs done
	sum := 0
	receiver1 := make(chan bool, numJobs1)
	receiver2 := make(chan bool, numJobs2)

	go func() {
		// Create one dispatcher
		disp := dispatcher.NewDispatcher()
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
		disp := dispatcher.NewDispatcher()
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
