package workforce

import (
	"context"
	"sync"
)

type Pool struct {
	Size   int           //The number of workers and size of the ingress and egress channels
	Inbox  chan Job      //Workers take jobs from this channel
	Outbox chan Result   //Worker output results to this channel
	Done   chan struct{} //Messages to this channel will close the pool
}

// Helper method to create a new pool
func NewPool(size int) Pool {
	return Pool{
		Size:   size,
		Inbox:  make(chan Job, size),
		Outbox: make(chan Result, size),
		Done:   make(chan struct{}),
	}
}

// LoadInbox takes an slice of jobs and submits them to the pool inbox. It then closes the inbox
// channel to prevent new jobs from being submitted
func (p *Pool) LoodInbox(jobs []Job) {
	for _, v := range jobs {
		p.Inbox <- v
	}
	close(p.Inbox)
}

// Returns the channel with results from the worker in the pool.
func (p *Pool) Results() <-chan Result {
	return p.Outbox
}

func (p *Pool) Run(ctx context.Context) {

	//Create a wait grup to prevent early exit
	var wg = new(sync.WaitGroup)

	//Populate the pool with workers
	for i := 0; i < p.Size; i++ {
		wg.Add(1)
		go worker(ctx, wg, p.Inbox, p.Outbox)
	}

	//Wait until all jobs have been processed
	wg.Wait()

	//Close all open channels
	close(p.Done)
	close(p.Outbox)

}

func worker(ctx context.Context, wg *sync.WaitGroup, inbox <-chan Job, outbox chan<- Result) {

	//decrement workgroup when the worker is no longer needed
	defer wg.Done()

	for {
		select {
		//A message is recieved in the inbox
		case job, ok := <-inbox:
			//Terminate worker when inbox recieves a message that's not a valid job
			if !ok {
				return
			}
			//Otherwise, execute job and the result to the outbox
			outbox <- job.Execute(ctx)

		//The current context is closed
		case <-ctx.Done():
			//Poisin pill the outbox to terminate the pool
			outbox <- Result{
				Err: ctx.Err(),
			}

			return
		}
	}
}
