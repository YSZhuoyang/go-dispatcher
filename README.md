# go-dispatcher

[![Build Status](https://travis-ci.org/YSZhuoyang/go-dispatcher.svg?branch=master)](https://travis-ci.org/YSZhuoyang/go-dispatcher)
[![Coverage Status](https://coveralls.io/repos/github/YSZhuoyang/go-dispatcher/badge.svg?branch=master)](https://coveralls.io/github/YSZhuoyang/go-dispatcher?branch=master)
[![GoDoc](https://godoc.org/github.com/YSZhuoyang/go-dispatcher/dispatcher?status.svg)](https://godoc.org/github.com/YSZhuoyang/go-dispatcher/dispatcher)

A worker-pool job dispatcher inspired by the [Post: Handling 1 Million Requests per Minute with Go](http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/).

* Easily run batches of jobs in sequence, for algorithms that involve running a set of independent tasks concurrently for a while, then all wait on a barrier, and repeat again.
* Limit total number of goroutines to prevent it from draining out resources.

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

A worker pool is used to spawn and keep workers globally. One or more job dispatchers can be created by taking subsets of workers from it, and registering them in their local worker pools. Workers are recycled by global worker pool after dispatchers are finalized.

Note: a dispatcher is not supposed to be reused. Always create a new dispatcher for one *start -> dispatch -> finalize* cycle.

## How to use

1. Download and import package.

        go get -u github.com/YSZhuoyang/go-dispatcher/dispatcher

2. Initialize a global worker pool with a number of workers.

        dispatcher.InitWorkerPoolGlobal(100000)

3. Create a job dispatcher giving a subset of workers from the global worker pool, and start listening to new jobs.

        disp := dispatcher.NewDispatcher()
        disp.Start(1000, nil)

4. Dispatch jobs (dispatch() will block until at least one worker becomes available and takes the job).

        type myJob struct {
            // ...
        }

        func (job *myJob) Do() {
            // Do something ...
        }

        disp.Dispatch(&myJob{...})

5. Wait until all jobs are done and return workers back to the global worker pool.

        disp.Finalize()

6. Optional: wait until all job dispatchers are finalized and destroy the global worker pool.

        dispatcher.DestroyWorkerPoolGlobal()
