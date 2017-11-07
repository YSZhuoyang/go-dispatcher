package main

import (
	"fmt"
	"go-dispatcher/dispatcher"
	"net/http"
	"time"
)

const (
	numIter    int = 5000
	numWorkers int = 10000
)

type serverJob struct {
	numIter int
}

func (job *serverJob) Do(worker dispatcher.Worker) {
	doSomething(job.numIter)
}

func doSomething(numIter int) {
	// x := 0
	// y := 1
	// a := 0
	// // Generate some workload
	// for i := 0; i < numIter; i++ {
	// 	for j := 0; j < numIter; j++ {
	// 		a += (x + y) / 2
	// 	}
	// }
	time.Sleep(10 * time.Second)
}

var disp dispatcher.Dispatcher

func DisLoadTestRequestHandler(
	w http.ResponseWriter,
	r *http.Request) {
	disp.Dispatch(&serverJob{numIter: numIter})
	fmt.Fprintf(w, "Request received, start dispatching job ...\n")
}

func NaiLoadTestRequestHandler(
	w http.ResponseWriter,
	r *http.Request) {
	go doSomething(numIter)
	fmt.Fprintf(w, "Request received, start a goroutine for the job ...\n")
}

func main() {
	// Init the global worker pool
	dispatcher.InitWorkerPoolGlobal(numWorkers)
	defer dispatcher.DestroyWorkerPoolGlobal()

	disp = dispatcher.NewDispatcher(0)
	disp.Spawn(numWorkers)
	disp.Start()
	defer disp.Finalize()

	// Setup request handlers
	http.HandleFunc("/dispatcherloadtest", DisLoadTestRequestHandler)
	http.HandleFunc("/naiveloadtest", NaiLoadTestRequestHandler)

	// Start server listener
	fmt.Println("Http server is now listening at 8080")
	http.ListenAndServe(":8080", nil)
}
