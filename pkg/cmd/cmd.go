package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
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
)

type Options struct {
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags

	namespace     string
	allNamespaces bool
	chunkSize     int64

	dryRunStrategy cmdutil.DryRunStrategy
	quiet          bool

	printObj func(obj runtime.Object) error

	determiner *determiner
	result     *resource.Result

	genericclioptions.IOStreams
}

func NewOptions(ioStreams genericclioptions.IOStreams) *Options {
	return &Options{
		configFlags: genericclioptions.NewConfigFlags(true),
		printFlags:  genericclioptions.NewPrintFlags("deleted").WithTypeSetter(scheme.Scheme),
		chunkSize:   500,
		IOStreams:   ioStreams,
	}
}

func NewCmdPrune(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(streams)

	cmd := &cobra.Command{
		Use:     "kubectl prune TYPE",
		Example: pruneExample,
		Run: func(cmd *cobra.Command, args []string) {
			f := cmdutil.NewFactory(o.configFlags)

			cmdutil.CheckErr(o.Validate(args))
			cmdutil.CheckErr(o.Complete(f, args, cmd))
			cmdutil.CheckErr(o.Run(f))
		},
	}

	o.configFlags.AddFlags(cmd.Flags())
	o.printFlags.AddFlags(cmd)

	cmdutil.AddDryRunFlag(cmd)
	cmd.Flags().BoolVarP(&o.allNamespaces, "all-namespaces", "A", false, "If true, prune the targeted resources across all namespace except kube-system")
	cmd.Flags().BoolVarP(&o.quiet, "quiet", "q", false, "If true, no output is produced")

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

	o.printFlags = cmdutil.PrintFlagsWithDryRunStrategy(o.printFlags, o.dryRunStrategy)

	printer, err := o.printFlags.ToPrinter()
	if err != nil {
		return
	}
	o.printObj = func(obj runtime.Object) error {
		return printer.PrintObj(obj, o.Out)
	}

	clientset, err := f.KubernetesClientSet()
	if err != nil {
		return err
	}

	namespace := o.namespace
	if o.allNamespaces {
		namespace = metav1.NamespaceAll
	}

	o.result = f.
		NewBuilder().
		Unstructured().
		ContinueOnError().
		NamespaceParam(o.namespace).
		DefaultNamespace().
		AllNamespaces(o.allNamespaces).
		ResourceTypeOrNameArgs(false, args[0]).
		RequestChunksOf(o.chunkSize).
		SelectAllParam(true).
		Flatten().
		Do()

	if err = o.result.Err(); err != nil {
		return err
	}

	o.determiner, err = newDeterminer(clientset, o.result, namespace)
	if err != nil {
		return err
	}

	return
}

func (o *Options) Validate(args []string) error {
	if len(args) != 1 {
		return errors.New("arguments must be only resource type(s)")
	}

	return nil
}

func (o *Options) Run(f cmdutil.Factory) error {
	if err := o.result.Visit(func(info *resource.Info, err error) error {
		if info.Namespace == metav1.NamespaceSystem {
			return nil // ignore resources in kube-system namespace
		}

		prune, err := o.determiner.determinePrune(info)
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

		_, err = resource.
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
