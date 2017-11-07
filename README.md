# go-dispatcher
A worker-pool job dispatcher inspired by the [Post: Handling 1 Million Requests per Minute with Go](http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/).

* Easily manage dependencies of batches of job executions.
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
          |  dispatcher  |     |  dispatcher  |
          |              |     |              |
          ----------------     ----------------

A worker pool is used to spawn and keep workers globally. One or more job dispatchers can be created by taking subsets of workers from it, and registering them in their local worker pools. Workers are recycled by global worker pool after dispatchers are finalized. Batches of jobs executed with different dispatchers can be either synchronous or asynchronous.

## How to use

1. Initialize a global worker pool with a number of workers.

        dispatcher.InitWorkerPoolGlobal(100000)

2. Create a job dispatcher giving a subset of workers from the global worker pool.

        disp := dispatcher.NewDispatcher(0)
        disp.Spawn(1000)

3. Start listening to new jobs

        disp.Start()

4. Dispatch jobs (dispatch() will block until at least one worker becomes available)

        type myJob struct {
            // ...
        }

        func (job *testJob) Do(worker dispatcher.Worker) {
            // Do something ...
        }

        disp.Dispatch(&myJob{...})

5. Wait until all jobs are done and return workers back to the global worker pool

        disp.Finalize()

6. Optional: wait until all job dispatchers are finalized and destroy the global worker pool.

        dispatcher.DestroyWorkerPoolGlobal()
