# go-dispatcher

[![Build Status](https://travis-ci.org/YSZhuoyang/go-dispatcher.svg?branch=master)](https://travis-ci.org/YSZhuoyang/go-dispatcher)
[![GoDoc](https://godoc.org/github.com/YSZhuoyang/go-dispatcher/dispatcher?status.svg)](https://godoc.org/github.com/YSZhuoyang/go-dispatcher/dispatcher)

A worker-pool job dispatcher inspired by the [Post: Handling 1 Million Requests per Minute with Go](http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/).

* Easily manage dependencies of concurrent job executions.
* Limit the total number of goroutines to prevent it from draining out of resources.

## How it works

                    -------------------------
                    |  global worker pool   |
                    |                       |
                    |                       |
                    -------------------------
                     | /|\              | /|\
               get   |  | return        |  |
             workers |  | workers       |  |
                    \|/ |              \|/ |
              ----------------     ----------------
      jobs -> |  dispatcher  |     |  dispatcher  | <- jobs
              |              |     |              |
              ----------------     ----------------

A worker pool is used to spawn and keep workers globally. One or more job dispatchers can be created by taking subsets of workers from it, and registering them in their local worker pools. Workers are recycled by global worker pool after dispatchers are finalized. Batches of jobs executed by multiple dispatchers can be either synchronous or asynchronous.

Note: a dispatcher is not supposed to be reused. Always create a new dispatcher for one start - finalize execution cycle.

## How to use

1. Download and import package.

        go get github.com/YSZhuoyang/go-dispatcher/dispatcher

1. Initialize a global worker pool with a number of workers.

        dispatcher.InitWorkerPoolGlobal(100000)

2. Create a job dispatcher giving a subset of workers from the global worker pool, and start listening to new jobs.

        disp := dispatcher.NewDispatcher()
        disp.Start(1000)

3. Dispatch jobs (dispatch() will block until at least one worker becomes available and takes the job).

        type myJob struct {
            // ...
        }

        func (job *myJob) Do() {
            // Do something ...
        }

        disp.Dispatch(&myJob{...})

4. Wait until all jobs are done and return workers back to the global worker pool.

        disp.Finalize()

5. Optional: wait until all job dispatchers are finalized and destroy the global worker pool.

        dispatcher.DestroyWorkerPoolGlobal()
