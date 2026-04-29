package cel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Pods is a container to query Pod objects on.
//
// TODO: Should there be an internal `pods` type to consider the native type?
type Pods struct {
	restClientGetter genericclioptions.RESTClientGetter
}

var PodsType = cel.OpaqueType("k8s.Pods")

//func (ps Pods) byName(name string) (Pod, error) {
//
//}

func (ps Pods) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if reflect.TypeOf(ps).AssignableTo(typeDesc) {
		return ps, nil
	}
	return nil, fmt.Errorf("type conversion error from 'Pods' to '%v'", typeDesc)
}

func (ps Pods) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case PodsType:
		return ps
	case types.TypeType:
		return PodsType
	}
	return types.NewErr("type conversion error from '%s' to '%s'", PodsType, typeVal)
}

// Equal checks if two Pods are equal.
//
// As Pods is a container of methods more than an actual type. In that sense two Pods instances cannot be unequal.
func (ps Pods) Equal(other ref.Val) ref.Val {
	_, ok := other.(Pods)
	if !ok {
		return types.ValOrErr(other, "no such overload")
	}

	return types.True
}

func (ps Pods) Type() ref.Type {
	return PodsType
}

func (ps Pods) Value() any {
	return ps
}

// Pod wraps api.core.v1.Pod to provide a CEL representation of a Kubernetes Pod.
type Pod struct {
	corev1.Pod
}

var PodType = cel.OpaqueType("k8s.Pod")

func (p Pod) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if reflect.TypeOf(p.Pod).AssignableTo(typeDesc) {
		return p.Pod, nil
	}
	if reflect.TypeOf("").AssignableTo(typeDesc) {
		return p.Pod.String(), nil
	}
	return nil, fmt.Errorf("type conversion error from 'Pod' to '%v'", typeDesc)
}

func (p Pod) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case PodType:
		return p
	case types.TypeType:
		return PodType
	case types.StringType:
		return types.String(p.Pod.String())
	}
	return types.NewErr("type conversion error from '%s' to '%s'", PodType, typeVal)
}

func (p Pod) Equal(other ref.Val) ref.Val {
	otherPod, ok := other.(Pod)
	if !ok {
		return types.ValOrErr(other, "no such overload")
	}

	// TODO: add more checks?
	return types.Bool(p.Pod.Name == otherPod.Pod.Name)
}

func (p Pod) Type() ref.Type {
	return PodType
}

func (p Pod) Value() any {
	return p.Pod
}
