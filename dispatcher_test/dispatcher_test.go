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
	disp := dispatcher.NewDispatcher(0)
	numWorkersTaken := 100
	disp.Spawn(numWorkersTaken)
	numWorkersLeft := dispatcher.GetNumWorkersAvail()
	numWorkersLeftExpected := numWorkers - numWorkersTaken
	assertion.Equal(
		numWorkersLeftExpected,
		numWorkersLeft,
		"Number of workers left were not correct",
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
