package dispatcher_test

import (
	"go-dispatcher/dispatcher"
	"testing"

	"github.com/stretchr/testify/assert"
)

const numWorkers int = 1000

type testJob struct {
	resultSender chan bool
}

func (job *testJob) Do(worker dispatcher.Worker) {
	job.resultSender <- true
}

func TestInitAndDestroyWorkerPool(T *testing.T) {
	assertion := assert.New(T)
	// Verify initialization
	dispatcher.InitWorkerPoolGlobal(numWorkers)
	numWorkersInitialized := dispatcher.GetNumWorkersAvail()
	assertion.Equal(
		numWorkersInitialized,
		numWorkers,
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

func TestSpawning(T *testing.T) {
	assertion := assert.New(T)
	dispatcher.InitWorkerPoolGlobal(numWorkers)
	// Create one dispatcher
	disp := dispatcher.NewDispatcher(0)
	numWorkersTaken := 100
	disp.Spawn(numWorkersTaken)
	disp.Start()
	numWorkersLeft := dispatcher.GetNumWorkersAvail()
	numWorkersLeftExpected := numWorkers - numWorkersTaken
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"1) Number of workers left were not correct",
	)
	// Create another dispatcher
	disp2 := dispatcher.NewDispatcher(0)
	disp2.Spawn(numWorkersTaken)
	disp2.Start()
	numWorkersLeft = dispatcher.GetNumWorkersAvail()
	numWorkersLeftExpected -= numWorkersTaken
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
	numWorkersLeftExpected += numWorkersTaken
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"4) Number of workers left were not correct",
	)
}

func TestDispatching(T *testing.T) {
	assertion := assert.New(T)
	dispatcher.InitWorkerPoolGlobal(numWorkers)
	disp := dispatcher.NewDispatcher(0)
	numWorkersTaken := 100
	disp.Spawn(numWorkersTaken)
	disp.Start()
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
}
