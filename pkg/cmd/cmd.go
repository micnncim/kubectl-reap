package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cliresource "k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/micnncim/kubectl-prune/pkg/determiner"
	"github.com/micnncim/kubectl-prune/pkg/version"
)

const (
	pruneExample = `
  # Delete ConfigMaps not mounted on any Pods and in the current namespace and context
  $ kubectl prune configmaps

  # Delete unused ConfigMaps and Secrets in the namespace/my-namespace and context/my-context
  $ kubectl prune cm,secret -n my-namespace --context my-context

  # Delete ConfigMaps not mounted on any Pods and across all namespace
  $ kubectl prune cm --all-namespaces

  # Delete Pods whose status is not Running as client-side dry-run
  $ kubectl prune po --dry-run=client`

	pruneShortDescription = `
Delete unused resources. Supported resources:

- ConfigMaps (not used in any Pods)
- Secrets (not used in any Pods or ServiceAccounts)
- Pods (whose status is not Running)
- PersistentVolumes (not satisfying any PersistentVolumeClaims)
- PersistentVolumeClaims (not used in any Pods)
- PodDisruptionBudgets (not targeting any Pods)
- HorizontalPodAutoscalers (not targeting any resources)
`

	// printedOperationTypePrune is used when printer outputs the result of operations.
	printedOperationTypePrune = "deleted"
)

type Options struct {
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags

	namespace     string
	allNamespaces bool
	chunkSize     int64

	dryRunStrategy cmdutil.DryRunStrategy
	quiet          bool

	showVersion bool

	printObj func(obj runtime.Object) error

	determiner *determiner.Determiner
	result     *cliresource.Result

	genericclioptions.IOStreams
}

func NewOptions(ioStreams genericclioptions.IOStreams) *Options {
	return &Options{
		configFlags: genericclioptions.NewConfigFlags(true),
		printFlags:  genericclioptions.NewPrintFlags(printedOperationTypePrune).WithTypeSetter(scheme.Scheme),
		chunkSize:   500,
		IOStreams:   ioStreams,
	}
}

func NewCmdPrune(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(streams)

	cmd := &cobra.Command{
		Use:     "kubectl prune RESOURCE_TYPE",
		Short:   pruneShortDescription,
		Example: pruneExample,
		Run: func(cmd *cobra.Command, args []string) {
			if o.showVersion {
				fmt.Fprintf(o.Out, "%s (%s)\n", version.Version, version.Revision)
				return
			}

			f := cmdutil.NewFactory(o.configFlags)

			cmdutil.CheckErr(o.Validate(args))
			cmdutil.CheckErr(o.Complete(f, args, cmd))
			cmdutil.CheckErr(o.Run(context.Background(), f))
		},
	}

	o.configFlags.AddFlags(cmd.Flags())
	o.printFlags.AddFlags(cmd)

	cmdutil.AddDryRunFlag(cmd)
	cmd.Flags().BoolVarP(&o.allNamespaces, "all-namespaces", "A", false, "If true, prune the targeted resources across all namespace except kube-system")
	cmd.Flags().BoolVarP(&o.quiet, "quiet", "q", false, "If true, no output is produced")
	cmd.Flags().BoolVarP(&o.showVersion, "version", "v", false, "If true, show the version of kubectl-prune")

	return cmd
}

func (o *Options) Complete(f cmdutil.Factory, args []string, cmd *cobra.Command) (err error) {
	o.namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return
	}

	o.dryRunStrategy, err = cmdutil.GetDryRunStrategy(cmd)
	if err != nil {
		return
	}

	if err = o.completePrintObj(); err != nil {
		return
	}

	if err = o.completeResources(f, args[0]); err != nil {
		return
	}

	clientset, err := f.KubernetesClientSet()
	if err != nil {
		return
	}
	dynamicClient, err := f.DynamicClient()
	if err != nil {
		return
	}

	namespace := o.namespace
	if o.allNamespaces {
		namespace = metav1.NamespaceAll
	}

	o.determiner, err = determiner.New(clientset, dynamicClient, o.result, namespace)
	if err != nil {
		return
	}

	return
}

func (o *Options) completePrintObj() error {
	o.printFlags = cmdutil.PrintFlagsWithDryRunStrategy(o.printFlags, o.dryRunStrategy)

	printer, err := o.printFlags.ToPrinter()
	if err != nil {
		return err
	}

	o.printObj = func(obj runtime.Object) error {
		return printer.PrintObj(obj, o.Out)
	}

	return nil
}

func (o *Options) completeResources(f cmdutil.Factory, resourceTypes string) error {
	o.result = f.
		NewBuilder().
		Unstructured().
		ContinueOnError().
		NamespaceParam(o.namespace).
		DefaultNamespace().
		AllNamespaces(o.allNamespaces).
		ResourceTypeOrNameArgs(false, resourceTypes).
		RequestChunksOf(o.chunkSize).
		SelectAllParam(true).
		Flatten().
		Do()

	return o.result.Err()
}

func (o *Options) Validate(args []string) error {
	if len(args) != 1 && !o.showVersion {
		return errors.New("arguments must be only resource type(s)")
	}

	return nil
}

func (o *Options) Run(ctx context.Context, f cmdutil.Factory) error {
	if err := o.result.Visit(func(info *cliresource.Info, err error) error {
		if info.Namespace == metav1.NamespaceSystem {
			return nil // ignore resources in kube-system namespace
		}

		prune, err := o.determiner.DeterminePrune(ctx, info)
		if err != nil {
			return err
		}
		if !prune {
			return nil // skip prune
		}

		if o.dryRunStrategy == cmdutil.DryRunClient && !o.quiet {
			o.printObj(info.Object)
			return nil // skip prune
		}

		_, err = cliresource.
			NewHelper(info.Client, info.Mapping).
			DryRun(o.dryRunStrategy == cmdutil.DryRunServer).
			DeleteWithOptions(info.Namespace, info.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		if !o.quiet {
			o.printObj(info.Object)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
