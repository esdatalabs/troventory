package workforce

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

// The 'work' being done is casting the input to an int
var castToInt ExecuteFunction = func(ctx context.Context, thingToCast interface{}) (interface{}, error) {

	maybeAnInt, ok := thingToCast.(string)

	// Return an error if casting fails for any reason
	if !ok {
		return nil, errors.New("The provided variable cannot be cast to an int")
	}

	return strconv.Atoi(maybeAnInt)
}

func Test_JobProcessesSuccessfully(t *testing.T) {

	ctx := context.TODO()

	job1 := NewJob("String to integer casting", "3", castToInt)

	result := job1.Execute(ctx)

	if result.Value != 3 {
		t.Errorf("Expected %s to be cast to int successfully: %s", "3", result.Err.Error())
	}
}

func Test_JobProcessingFails(t *testing.T) {

	ctx := context.TODO()

	job2 := NewJob("String to integer casting should fail", "thesed", castToInt)

	resp := job2.Execute(ctx)

	if resp.Err == nil {
		t.Errorf("Expected %s to failed being cast to int", "thesed")
	}
}
