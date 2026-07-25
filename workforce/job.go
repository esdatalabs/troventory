package workforce

import "context"

// The ExecuteFunction is the single unit of work to be performed
type ExecuteFunction func(ctx context.Context, args interface{}) (interface{}, error)

// A job has the fields required to submit a unit of work to the worker pool
type Job struct {
	Description string
	Arguments   interface{}
	executable  ExecuteFunction
}

// A result has the fields required to describe the output of a job being executed
type Result struct {
	Description string
	Value       interface{}
	Err         error
}

// Helper method to construct a new Job
func NewJob(desc string, args interface{}, exec ExecuteFunction) Job {
	return Job{
		Description: desc,
		Arguments:   args,
		executable:  exec,
	}
}

// The Execute function is triggered by a worker in worker pool
func (j *Job) Execute(ctx context.Context) Result {

	// Invoke the underlying unit of work
	value, err := j.executable(ctx, j.Arguments)

	res := Result{Description: j.Description, Value: value}

	if err != nil {
		res.Err = err
	}

	return res

}
