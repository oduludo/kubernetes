package cel

import (
	"context"

	"github.com/google/cel-go/cel"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	celLong = templates.LongDesc(i18n.T(`
		Run a CEL expression against Kubernetes API objects.

		CEL provides a powerful and easy to use expression language. CEL commands can be used to 
		fetch Kubernetes objects from the cluster. An example use case is to get the service(s) attached 
		to a Pod, by running kubectl cel 'pod.byName("my-pod")._services'. Any field beginning with an
		underscore is a CEL macro/function that tries to resolve certain related objects. In this case 
		that's Service objects which are retrieved using the label matching used by Kubernetes.
	`))
	celExample = templates.Examples(i18n.T(`
		# Get all Pods
		kubectl cel pod

		# Get a Pod by name
		kubectl cel 'pod.byName("my-pod")'

		# Get all Services attached to a Pod
		kubectl cel 'pod.byName("my-pod")._services'
	`))
)

func NewCelCommand(restClientGetter genericclioptions.RESTClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cel <expression>",
		Short:   i18n.T("Query Kubernetes objects using CEL expressions."),
		Long:    celLong,
		Example: celExample,

		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd.Context(), args[0])
		},
	}

	return cmd
}

func Run(ctx context.Context, expression string) error {
	klog.Infof("run CEL command: %s\n", expression)

	env, err := cel.NewEnv()
	if err != nil {
		return err
	}

	ast := compile(env, expression, cel.DynType)
	program, err := env.Program(ast)
	if err != nil {
		return err
	}

	out, _, err := eval(
		program,
		[]string{},
	)

	if err != nil {
		return err
	}

	klog.Info(out)
	return nil
}
