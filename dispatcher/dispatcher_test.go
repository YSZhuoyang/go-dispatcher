package dispatcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testJob struct {
	resultSender chan bool
}

func (job *testJob) Do() {
	job.resultSender <- true
}

type testBigJob struct {
	mutex       *sync.Mutex
	accumulator *int
}

func (job *testBigJob) Do() {
	job.mutex.Lock()
	defer job.mutex.Unlock()

	time.Sleep(50000)
	*job.accumulator++
}

func TestInitWorkerPool(T *testing.T) {
	assertion := assert.New(T)
	disp, _ := NewDispatcher(1000)

	// Expect correct number of workers should be returned
	numWorkersAvail := disp.GetNumWorkersAvail()
	assertion.Equal(numWorkersAvail, 1000)

	totalNumWorkers := disp.GetTotalNumWorkers()
	assertion.Equal(totalNumWorkers, 1000)

	// Expect an error to be returned by getting number of available workers
	// given invalid param
	disp, err := NewDispatcher(0)
	assertion.Nil(disp)
	assertion.EqualError(
		err,
		"Invalid number of workers given to create a new dispatcher",
		"Unexpected error returned by creating new dispatcher given invalid number of workers",
	)

	disp, err = NewDispatcher(-1)
	assertion.Nil(disp)
	assertion.EqualError(
		err,
		"Invalid number of workers given to create a new dispatcher",
		"Unexpected error returned by creating new dispatcher given invalid number of workers",
	)
}

func TestFinializingJobs(T *testing.T) {
	assertion := assert.New(T)
	numWorkers := 100
	disp, _ := NewDispatcher(numWorkers)
	var mutex sync.Mutex
	accumulator := 0
	for i := 0; i < numWorkers; i++ {
		// Each big job takes 50000 ns to finish
		disp.Dispatch(&testBigJob{accumulator: &accumulator, mutex: &mutex})
	}

	disp.Finialize()
	assertion.Equal(accumulator, numWorkers, "Dispatcher did not wait for all jobs to complete")
}

func TestFinializingDelayedJobs(T *testing.T) {
	assertion := assert.New(T)
	numWorkers := 100
	disp, _ := NewDispatcher(numWorkers)
	var mutex sync.Mutex
	accumulator := 0
	for i := 0; i < numWorkers; i++ {
		// Each big job takes 50000 ns to finish
		disp.DispatchWithDelay(&testBigJob{accumulator: &accumulator, mutex: &mutex}, 50000)
	}

	disp.Finialize()
	assertion.Equal(accumulator, numWorkers, "Dispatcher did not wait for all jobs to complete")
}

func TestFinializingManyDelayedJobs(T *testing.T) {
	assertion := assert.New(T)
	numWorkers := 100
	numJobs := numWorkers * 10
	disp, _ := NewDispatcher(numWorkers)
	var mutex sync.Mutex
	accumulator := 0
	for i := 0; i < numJobs; i++ {
		// Each big job takes 50000 ns to finish
		disp.DispatchWithDelay(&testBigJob{accumulator: &accumulator, mutex: &mutex}, 50000)
	}

	disp.Finialize()
	assertion.Equal(accumulator, numJobs, "Dispatcher did not wait for all jobs to complete")
}

func TestDispatchingJobs(T *testing.T) {
	assertion := assert.New(T)
	numWorkers := 100
	receiver := make(chan bool, numWorkers)
	disp, _ := NewDispatcher(numWorkers)

	// Dispatch jobs
	sum := 0
	for i := 0; i < numWorkers; i++ {
		disp.Dispatch(&testJob{resultSender: receiver})
	}
	disp.Finialize()
	close(receiver)
	// Verify the number of jobs being done
	for range receiver {
		sum++
	}
	assertion.Equal(numWorkers, sum, "Incorrect number of job being dispatched")
}

func TestDispatchingJobsWithDelay(T *testing.T) {
	assertion := assert.New(T)
	numWorkers := 100
	receiver := make(chan bool, numWorkers)
	disp, _ := NewDispatcher(numWorkers)

	start := time.Now()
	// Dispatch jobs with delay
	for i := 0; i < numWorkers; i++ {
		disp.DispatchWithDelay(&testJob{resultSender: receiver}, 10000)
	}
	disp.Finialize()
	elapse := time.Since(start)
	assertion.True(elapse >= 1000000, "Job dispatching was not delayed with the correct time period")
}

func TestDispatchingJobsWithDelayError(T *testing.T) {
	assertion := assert.New(T)
	numWorkers := 100
	receiver := make(chan bool, numWorkers)
	disp, _ := NewDispatcher(numWorkers)

	err := disp.DispatchWithDelay(&testJob{resultSender: receiver}, 0)
	assertion.EqualError(
		err,
		"Invalid delay period",
		"Unexpected error returned by dispatch given invalid delay period",
	)
}

func TestDispatchingManyJobs(T *testing.T) {
	assertion := assert.New(T)
	numWorkers := 100
	numJobs := numWorkers * 100
	receiver := make(chan bool, numWorkers)
	disp, _ := NewDispatcher(numWorkers)

	go func(numJobs int) {
		for i := 0; i < numJobs; i++ {
			disp.Dispatch(&testJob{resultSender: receiver})
		}
		disp.Finialize()
		close(receiver)
	}(numJobs)

	sum := 0
	// Verify the number of jobs being done
	for range receiver {
		sum++
	}
	assertion.Equal(numJobs, sum, "Incorrect number of job executed")
}
