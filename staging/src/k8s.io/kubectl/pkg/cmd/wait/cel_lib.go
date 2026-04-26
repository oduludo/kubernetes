package wait

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

func celLibraryFunctions() []cel.EnvOption {
	typeParamA := cel.TypeParamType("A")
	typeParamB := cel.TypeParamType("B")
	mapAB := cel.MapType(typeParamA, typeParamB)

	type function struct {
		name       string
		overload   string
		inputTypes []*cel.Type
		outputType *cel.Type
		impl       cel.OverloadOpt
	}

	functions := []function{
		{
			"hasFinished",
			"job_map_hasFinished",
			[]*cel.Type{mapAB},
			cel.BoolType,
			cel.FunctionBinding(jobHasFinished),
		},
		{
			"hasSucceeded",
			"job_map_hasSucceeded",
			[]*cel.Type{mapAB},
			cel.BoolType,
			cel.FunctionBinding(jobHasSucceeded),
		},
		{
			"hasFailed",
			"job_map_hasFailed",
			[]*cel.Type{mapAB},
			cel.BoolType,
			cel.FunctionBinding(jobHasFailed),
		},
	}

	registeredFunctions := make([]cel.EnvOption, len(functions))

	for i, f := range functions {
		registeredFunctions[i] = cel.Function(
			f.name,
			cel.MemberOverload(
				f.overload,
				f.inputTypes,
				f.outputType,
				f.impl,
			),
		)
	}

	return registeredFunctions
}

// --------------------------
//
// # CONVERTERS
//
// --------------------------
func convertMapperToJob(v ref.Val) (*batchv1.Job, error) {
	m, ok := v.(traits.Mapper)

	if !ok {
		return nil, fmt.Errorf("expected traits.Mapper, got %T", v)
	}

	raw, err := m.ConvertToNative(reflect.TypeOf(map[string]interface{}{}))
	if err != nil {
		return nil, fmt.Errorf("failed to convert traits.Mapper to native type: %w", err)
	}

	obj, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.New("failed to cast raw interfacte to map[string]interface{}")
	}

	var job batchv1.Job
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj, &job); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured object to batch.v1.Job: %w", err)
	}

	return &job, nil
}

// --------------------------
//
// # LIBRARY IMPLEMENTATIONS
//
// Libraries have their focus object (likely always a runtime.Object implementation) on which they run functionality.
// Each such object has a cel<Kind> type, e.g. celJob.
//
// What happens where?
// - CEL types are the input for CEL methods/functions and are passed to the constructor (e.g. newCelJob).
// - The type (e.g. celJob) has methods implemented to run logic. All types received and returned by these methods are native Kubernetes and Go types.
// - It's the CEL method/function's task to take the return value from e.g. celJob.hasFinished() and convert it to a CEL type (in this case types.Bool).
//
// --------------------------

// --------------------------
// ## JOBS
// --------------------------
type celJob struct {
	job *batchv1.Job
}

func newCelJob(v ref.Val) (*celJob, error) {
	job, err := convertMapperToJob(v)

	if err != nil {
		return nil, err
	}

	return &celJob{job: job}, nil
}

func (c celJob) hasFinished() bool {
	return c.hasFinishedInSuccessState() || c.hasFinishedInFailedState()
}

func (c celJob) hasFinishedInSuccessState() bool {
	for _, c := range c.job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

func (c celJob) hasFinishedInFailedState() bool {
	for _, c := range c.job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// jobHasFinished implements the custom method:
//
//	`job.hasFinished() bool`
//
// This method checks if a Job has finished (in any outcome). It uses Job.Status.Conditions to determine the outcome.
func jobHasFinished(args ...ref.Val) ref.Val {
	job, err := newCelJob(args[0])

	if err != nil {
		klog.Errorf("failed to convert traits.Mapper: %v", err)
		return types.False
	}

	return types.Bool(job.hasFinished())
}

// jobHasSucceeded implements the custom method:
//
//	`job.hasSucceeded() bool`
//
// This method checks if a Job has finished successfully, using Job.Status.Conditions.
func jobHasSucceeded(args ...ref.Val) ref.Val {
	job, err := newCelJob(args[0])

	if err != nil {
		klog.Errorf("failed to convert traits.Mapper: %v", err)
		return types.False
	}

	return types.Bool(job.hasFinishedInSuccessState())
}

// jobHasFailed implements the custom method:
//
//	`job.hasFailed() bool`
//
// This method checks if a Job has finished in failure, using Job.Status.Conditions.
func jobHasFailed(args ...ref.Val) ref.Val {
	job, err := newCelJob(args[0])

	if err != nil {
		klog.Errorf("failed to convert traits.Mapper: %v", err)
		return types.False
	}

	return types.Bool(job.hasFinishedInFailedState())
}
