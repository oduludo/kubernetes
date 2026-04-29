package cel

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

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
	return out, nil
}

/*
Next steps:
- Load PodType into the CEL environment.
- Add overload methods for PodType, e.g. `pod.byName("pod-name")`
- Differentiate between global value `pod`, which is a container/namespace for utilities to query Pods and the _actual_ Pod(s) that would be returned by e.g. `pod.byName("pod-name")`
- Implement ORM-like lookup such as `<pod>._services` to get all Services a Pod is associated with or `<pod>._deployment`/`<pod>._replicaset` to get the Deployment/ReplicaSet (assuming there's never more than one) the Pod is associated with.
	- The association is probably done using label matching. Can differ per resource type or even per user configuration. It's pretty much what Rancher would show under 'Related services' (or smth).
*/

func exec(restClientGetter genericclioptions.RESTClientGetter, expression string, want *cel.Type) (ref.Val, error) {
	typeParamPods := cel.TypeParamType("Pods")
	typeParamA := cel.TypeParamType("A")

	opts := []cel.EnvOption{
		cel.Types(PodsType),
		cel.Variable("pods", PodsType),
		cel.Function(
			"byName",
			cel.MemberOverload("byName", []*cel.Type{typeParamPods, typeParamA}, PodType, cel.FunctionBinding(podsByName)),
		),
	}
	//opts = append(opts, celLibraryFunctions()...)

	env, err := cel.NewEnv(opts...)

	if err != nil {
		return nil, err
	}

	return runCelCommand(
		env,
		want,
		expression,
		map[string]interface{}{
			"pods": Pods{restClientGetter: restClientGetter},
		},
	)
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
func eval(prg cel.Program, vars any) (out ref.Val, det *cel.EvalDetails, err error) {
	out, det, err = prg.Eval(vars)
	return
}

// Function bindings

func podsByName(args ...ref.Val) ref.Val {
	klog.Info("podsByName called with args: ", args)
	//klog.Info("podsByName called with pods: ", pods)
	pods, ok := args[0].Value().(Pods)
	if !ok {
		klog.Exitf("expected Pods argument for pods, got %T", args[0].Value())
	}
	klog.Info("go find pods through: ", pods)

	name, ok := args[1].Value().(string)
	if !ok {
		klog.Exitf("expected string argument for name, got %T", args[1].Value())
	}
	klog.Info("go find pods by name: ", name)

	return Pod{}
}
