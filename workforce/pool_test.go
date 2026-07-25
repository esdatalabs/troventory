package workforce

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	numberOfJobs = 3
)

// Helper function to generate a slice of jobs
func jobs() []Job {

	//Make a new slice with size capped at number of jobs
	bulk := make([]Job, numberOfJobs)

	for i := 0; i < numberOfJobs; i++ {
		// Populate the slice with simple jobs
		bulk[i] = NewJob(
			fmt.Sprintf("This is job %v, to be converted to int", i),
			fmt.Sprintf("%v", i),
			castToInt,
		)
	}

	return bulk
}

func Test_Pool(t *testing.T) {

	//Create the pool
	pool := NewPool(numberOfJobs)

	//Initialized the workers
	go pool.Run(context.TODO())

	//Bulk load and close the inbox
	pool.LoodInbox(jobs())

	for {

		select {
		//Process messages sent to the outbox
		case resp, ok := <-pool.Results():

			//We only care about Results, ignore everying else
			if !ok {
				continue
			}

			//Valid jobs should not result in errors during processing
			if resp.Err != nil {
				t.Error(resp.Err.Error())
			}

		//Exit processing if any message is received on the done channel
		case <-pool.Done:
			return
		}

	}
}

func Test_PoolContextTimeout(t *testing.T) {

	//Create the pool
	pool := NewPool(numberOfJobs)

	//Cancel the context after some time
	ctx, cancel := context.WithTimeout(context.TODO(), time.Nanosecond*10)

	//Initialized the workers
	go pool.Run(ctx)

	cancel()

	for {

		select {
		//Process messages sent to the outbox. This should only be the cancel message from the ctx
		case resp, ok := <-pool.Results():

			//We only care about Results, ignore everying else
			if !ok {
				continue
			}

			if resp.Err != context.DeadlineExceeded {
				t.Errorf("Expected context deadline exceeded message but got %s", resp.Err.Error())
			}

		//Exit processing if any message is received on the done channel
		case <-pool.Done:
			return
		}

	}

}
