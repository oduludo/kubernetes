package wait

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
	"google.golang.org/protobuf/encoding/prototext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"

	"encoding/json"

	"google.golang.org/protobuf/proto"
)

// TODO: Use proper (debug) logging and log only what's worth logging (being experimental CEL use).
const CELVerbose = false

func pr(text string) {
	if CELVerbose {
		fmt.Println(text)
	}
}

/*
	Current status:
	Got a very simple example working with a Pod. It can do things like reading the `pod.Name`. That's not YAML-like, as
	`pod.metadata.name` won't work.

	Next steps:
	- See if `cel: metadata` tags do anything on the native Go structs.
	- Import Pod type as a protobuf Message. This gives the best native CEL support and would be easiest to scale up to
	  all Kubernetes types.
*/

func runCelCommand(env *cel.Env, want *cel.Type, command string, vars map[string]interface{}) (ref.Val, error) {
	ast := compile(
		env,
		command,
		want,
	)
	program, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	out, _, err := eval(
		program,
		vars,
	)
	if err != nil {
		return nil, err
	}
	pr(fmt.Sprintf("------ type unwrap ------\n%v\n", out))
	pr("\n")
	return out, nil
}

// Convert unstructured.Unstructured back to a Pod (which only works for select use cases of course) and run it through
// a CEL program with the supplied command. The program loads a `pod` variable which is a native k8s.io/api/core/v1/Pod.
// Lookups are done Go structure style, e.g. `pod.Name`. This deviates from the structure people recognize from YAML
// manifests and JSON representation.
func runCelCommandOnNativeType(obj *unstructured.Unstructured, want *cel.Type, command string) (ref.Val, error) {
	env, err := cel.NewEnv(
		//cel.Types(&corev1.Pod{}),
		ext.NativeTypes(reflect.TypeOf(&corev1.Pod{})),
		cel.Variable("pod", cel.DynType),
	)

	if err != nil {
		return nil, err
	}

	var pod corev1.Pod
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &pod); err != nil {
		return nil, err
	}

	return runCelCommand(env, want, command, map[string]interface{}{"pod": pod})
}

// runCelCommandOnUnstructuredMapping takea the Object from unstructured.Unstructured and runs it through a CEL program
// with the supplied command. The program loads a variable with the name of the resource kind, with the value being a
// mapping type (string to any). Lookups follow the YAML structure, e.g. `pod.metadata.name`.
// runCelCommandOnUnstructuredMapping ever only handles a single resource at a time, never a list of resources.
func runCelCommandOnUnstructuredMapping(obj *unstructured.Unstructured, want *cel.Type, kind string, command string) (ref.Val, error) {
	opts := []cel.EnvOption{
		cel.Variable(kind, cel.MapType(cel.StringType, cel.DynType)),
	}
	opts = append(opts, celLibraryFunctions()...)

	env, err := cel.NewEnv(opts...)

	if err != nil {
		return nil, err
	}

	return runCelCommand(env, want, command, map[string]interface{}{kind: obj.Object})
}

// Use the protobuf type definition for Pod to run CEL queries on it.
// This setup doesn't work as corev1.Pod doesn't implement protobuf.Message or ref.Type interfaces. CEL and protobuf go
// hand in hand, but CEL can also work with mappings.
// The K8s docs state "Not all API resource types support Kubernetes' Protobuf encoding; specifically, Protobuf isn't
// available for resources that are defined as CustomResourceDefinitions or are served via the aggregation layer."
// (https://kubernetes.io/docs/reference/using-api/api-concepts/#protobuf-encoding-compatibility). So perhaps the
// protobuf approach isn't desired as it would mean some (native) resources are treated differently than other (CRD)
// resources while they do share the same interface on the CLI.
func runCelCommandOnProtobuf(obj *unstructured.Unstructured, want *cel.Type, command string) (ref.Val, error) {
	env, err := cel.NewEnv(
		cel.Types(&corev1.Pod{}),
		cel.Variable("pod", cel.ObjectType("api.core.v1.Pod")),
	)

	if err != nil {
		return nil, err
	}

	return runCelCommand(env, want, command, map[string]interface{}{"pod": obj})
}

// Functions to assist with CEL execution.

// compile will parse and check an expression `expr` against a given
// environment `env` and determine whether the resulting type of the expression
// matches the `exprType` provided as input.
func compile(env *cel.Env, expr string, celType *cel.Type) *cel.Ast {
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		klog.Exit(iss.Err())
	}
	if !reflect.DeepEqual(ast.OutputType(), celType) {
		klog.Exitf(
			"Got %v, wanted %v result type", ast.OutputType(), celType)
	}
	fmt.Printf("%s\n\n", strings.ReplaceAll(expr, "\t", " "))
	return ast
}

// eval will evaluate a given program `prg` against a set of variables `vars`
// and return the output, eval details (optional), or error that results from
// evaluation.
func eval(prg cel.Program,
	vars any) (out ref.Val, det *cel.EvalDetails, err error) {
	varMap, isMap := vars.(map[string]any)

	pr("------ input ------")
	if !isMap {
		pr(fmt.Sprintf("(%T)\n", vars))
	} else {
		for k, v := range varMap {
			switch val := v.(type) {
			case proto.Message:
				bytes, err := prototext.Marshal(val)
				if err != nil {
					klog.Exitf("failed to marshal proto to text: %v", val)
				}
				pr(fmt.Sprintf("%s = %s", k, string(bytes)))
			case map[string]any:
				b, _ := json.MarshalIndent(v, "", "  ")
				pr(fmt.Sprintf("%s = %v\n", k, string(b)))
			case uint64:
				pr(fmt.Sprintf("%s = %vu\n", k, v))
			default:
				pr(fmt.Sprintf("%s = %v\n", k, v))
			}
		}
	}
	pr("\n")
	out, det, err = prg.Eval(vars)
	if CELVerbose {
		report(out, det, err)
	}
	pr("\n")
	return
}

// report prints out the result of evaluation in human-friendly terms.
func report(result ref.Val, details *cel.EvalDetails, err error) {
	fmt.Println("------ result ------")
	if err != nil {
		fmt.Printf("error: %s\n", err)
	} else {
		fmt.Printf("value: %v (%T)\n", result, result)
	}
	if details != nil {
		fmt.Printf("\n------ eval states ------\n")
		state := details.State()
		stateIDs := state.IDs()
		ids := make([]int, len(stateIDs), len(stateIDs))
		for i, id := range stateIDs {
			ids[i] = int(id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			v, found := state.Value(int64(id))
			if !found {
				continue
			}
			fmt.Printf("%d: %v (%T)\n", id, v, v)
		}
	}
}

// CELWait provides wait conditions using the Common Expression Language (CEL).
// In the use case of `kubectl wait`, a CEL expression (or 'query') must always evaluate to a boolean result.
type CELWait struct {
	query  string
	errOut io.Writer
}

func (c CELWait) IsCELConditionMet(ctx context.Context, info *resource.Info, o *WaitOptions) (runtime.Object, bool, error) {
	return getObjAndCheckCondition(ctx, info, o, c.isConditionMet, c.checkCondition)
}

func (c CELWait) checkCondition(obj *unstructured.Unstructured) (bool, error) {
	kind := strings.ToLower(obj.GetKind())
	val, err := runCelCommandOnUnstructuredMapping(obj, cel.BoolType, kind, c.query)

	if err != nil {
		return false, err
	}

	return val.Value().(bool), nil
}

func (c CELWait) isConditionMet(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Error:
		// keep waiting in the event we see an error - we expect the watch to be closed by
		// the server
		err := apierrors.FromObject(event.Object)
		fmt.Fprintf(c.errOut, "error: An error occurred while waiting for the condition to be satisfied: %v", err)
		return false, nil
	case watch.Deleted:
		// this will chain back out, result in another get and a return false back up the chain
		return false, nil
	default:
		obj := event.Object.(*unstructured.Unstructured)
		return c.checkCondition(obj)
	}
}
