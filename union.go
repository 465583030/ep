package ep

import (
	"context"
	"fmt"
)

var _ = registerGob(union([]Runner{}))

// Union returns a new composite Runner that dispatches its inputs to all of
// its internal runners and collects their output into a single unified stream
// of datasets. It is required the all of the individual runners returns the
// same data types
func Union(runners ...Runner) (Runner, error) {
	if len(runners) == 0 {
		err := fmt.Errorf("at least 1 runner is required for union")
		return nil, err
	} else if len(runners) == 1 {
		return runners[0], nil
	}

	u := union(runners)
	_, err := u.ReturnsErr()
	if err != nil {
		return nil, err
	}

	return u, nil
}

type union []Runner

// see Runner. Assumes all runners has the same return types.
func (rs union) Returns() []Type {
	types, err := rs.ReturnsErr()
	if err != nil {
		panic("Union() should've prevented this error from panicking")
	}

	return types
}

// determine the return types - skipping NULLS as they don't expose any
// information about the actual data types.
func (rs union) ReturnsErr() ([]Type, error) {
	types := rs[0].Returns()

	// ensure that the return types are compatible
	for _, r := range rs {
		have := r.Returns()
		if len(have) != len(types) {
			return nil, fmt.Errorf("mismatch number of columns: %v and %v", types, have)
		}

		for i, t := range have {

			// choose the first column type that isn't a null
			if Null.Is(types[i]) {
				types[i] = have[i]
			} else if !Null.Is(t) && t.Name() != types[i].Name() {
				return nil, fmt.Errorf("type mismatch %s and %s", types, have)
			}
		}
	}

	return types, nil
}

func (rs union) Run(ctx context.Context, inp, out chan Dataset) (err error) {

	// start all inner runners
	inputs := make([]chan Dataset, len(rs))
	outputs := make([]chan Dataset, len(rs))
	for i := range rs {
		inputs[i] = make(chan Dataset)
		outputs[i] = make(chan Dataset)

		go func(i int) {
			defer close(outputs[i])
			err1 := rs[i].Run(ctx, inputs[i], outputs[i])
			if err1 != nil && err == nil {
				err = err1
			}
		}(i)
	}

	// fork the input to all inner runners
	go func() {
		for data := range inp {
			for _, s := range inputs {
				s <- data
			}
		}

		// close all inner runners
		for _, s := range inputs {
			close(s)
		}
	}()

	// collect and union all of the stream into a single output
	for _, s := range outputs {
		for data := range s {
			out <- data
		}
	}

	return err
}
