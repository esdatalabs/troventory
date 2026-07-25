package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/esdatalabs/workforce/workforce"
)

const (
	numberOfJobs = 10
)

var castToInt workforce.ExecuteFunction = func(ctx context.Context, thingToCast interface{}) (interface{}, error) {

	maybeAnInt, ok := thingToCast.(string)

	if !ok {
		return nil, errors.New("the provided variable cannot be cast to an int")
	}

	return strconv.Atoi(maybeAnInt)
}

// Helper function to generate a batch of jobs
func batchJobs() []workforce.Job {

	//Make a new slice with size capped at number of jobs
	batch := make([]workforce.Job, numberOfJobs)

	for i := 0; i < numberOfJobs; i++ {

		batch[i] = workforce.NewJob(
			fmt.Sprintf("This is job %v, to be converted to int", i),
			fmt.Sprintf("%v", i),
			castToInt,
		)
	}

	return batch
}

func main() {

	ctx := context.Background()
	defer ctx.Done()

	//Create the pool
	pool := workforce.NewPool(numberOfJobs)

	//Initialized the workers
	go pool.Run(ctx)

	//Bulk load and close the inbox
	pool.LoodInbox(batchJobs())

	for {

		select {
		//Process messages sent to the outbox
		case resp, ok := <-pool.Results():

			//We only care about Results, ignore everying else
			if !ok {
				continue
			}

			//Log errors
			if resp.Err != nil {
				fmt.Println(resp.Err.Error())
			} else {
				// Print response values
				fmt.Println(resp.Value)
			}

		//Exit processing if any message is received on the done channel
		case <-pool.Done:
			return
		}

	}

}
