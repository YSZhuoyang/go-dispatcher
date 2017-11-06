# go-dispatcher
A goroutine job dispatcher inspired by the [Post: Handling 1 Million Requests per Minute with Go](http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/).

* A worker-pool pattern is used.
* Easy control of dependencies of batches of job executions.
* Limit the total number of goroutines to prevent it from draining out of resources.

## How it works



## How to use

1. Initialize a global worker pool with a number of workers.

        dispatcher.InitWorkerPoolGlobal(100000)

2. Create a job dispatcher giving a subset of workers from the global worker pool.

        disp := dispatcher.NewDispatcher(0)
        disp.Spawn(1000)

3. Start listening to new jobs

        disp.Start()

4. Dispatch jobs (dispatch() will block until at least one worker becomes available, use go dispatch() to dispatch jobs async)

        type myJob struct {
            // ...
        }

        func (job *testJob) Do(worker dispatcher.Worker) {
            // Do something ...
        }

        disp.Dispatch(&myJob{...})

5. Finalize job dispatcher and wait until all jobs are done

        disp.Finalize()

6. Optional: Wait until all job dispatchers are finalized and destroy the global worker pool.

        dispatcher.DestroyWorkerPoolGlobal()
