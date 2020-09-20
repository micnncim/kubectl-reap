package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	cliresource "k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	cmdwait "k8s.io/kubectl/pkg/cmd/wait"

	"github.com/micnncim/kubectl-prune/pkg/determiner"
	"github.com/micnncim/kubectl-prune/pkg/prompt"
	"github.com/micnncim/kubectl-prune/pkg/resource"
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

	// printedOperationTypeDeleted is used when printer outputs the result of operations.
	printedOperationTypeDeleted = "deleted"
)

var timeWeek = 168 * time.Hour

type Options struct {
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags

	namespace       string
	allNamespaces   bool
	chunkSize       int64
	labelSelector   string
	fieldSelector   string
	gracePeriod     int
	forceDeletion   bool
	waitForDeletion bool
	timeout         time.Duration

	quiet       bool
	interactive bool

	showVersion bool

	dryRunStrategy cmdutil.DryRunStrategy
	dryRunVerifier *cliresource.DryRunVerifier

	deleteOpts *metav1.DeleteOptions

	determiner    determiner.Determiner
	dynamicClient dynamic.Interface
	printer       printers.ResourcePrinter
	result        *cliresource.Result

	genericclioptions.IOStreams
}

func NewOptions(ioStreams genericclioptions.IOStreams) *Options {
	return &Options{
		configFlags: genericclioptions.NewConfigFlags(true),
		printFlags:  genericclioptions.NewPrintFlags(printedOperationTypeDeleted).WithTypeSetter(scheme.Scheme),
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
				o.Infof("%s (%s)\n", version.Version, version.Revision)
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

	cmd.Flags().BoolVarP(&o.allNamespaces, "all-namespaces", "A", false, "If true, delete the targeted resources across all namespace except kube-system")
	cmd.Flags().StringVarP(&o.labelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().StringVar(&o.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	cmd.Flags().IntVar(&o.gracePeriod, "grace-period", -1, "Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
	cmd.Flags().BoolVar(&o.forceDeletion, "force", false, "If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
	cmd.Flags().BoolVar(&o.waitForDeletion, "wait", false, "If true, wait for resources to be gone before returning. This waits for finalizers.")
	cmd.Flags().DurationVar(&o.timeout, "timeout", 0, "The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
	cmd.Flags().BoolVarP(&o.quiet, "quiet", "q", false, "If true, no output is produced")
	cmd.Flags().BoolVarP(&o.interactive, "interactive", "i", false, "If true, a prompt asks whether resources can be deleted")
	cmd.Flags().BoolVarP(&o.showVersion, "version", "v", false, "If true, show the version of this plugin")

	return cmd
}

func (o *Options) Complete(f cmdutil.Factory, args []string, cmd *cobra.Command) (err error) {
	if !o.forceDeletion && o.gracePeriod == 0 {
		// To preserve backwards compatibility, but prevent accidental data loss, we convert --grace-period=0
		// into --grace-period=1. Users may provide --force to bypass this conversion.
		o.gracePeriod = 1
	}
	if o.forceDeletion && o.gracePeriod < 0 {
		o.gracePeriod = 0
	}
	o.deleteOpts = &metav1.DeleteOptions{}
	if o.gracePeriod >= 0 {
		o.deleteOpts = metav1.NewDeleteOptions(int64(o.gracePeriod))
	}

	o.namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return
	}

	o.dryRunStrategy, err = cmdutil.GetDryRunStrategy(cmd)
	if err != nil {
		return
	}

	if err = o.completePrinter(); err != nil {
		return
	}

	if err = o.completeResources(f, args[0]); err != nil {
		return
	}

	clientset, err := f.KubernetesClientSet()
	if err != nil {
		return
	}
	o.dynamicClient, err = f.DynamicClient()
	if err != nil {
		return
	}
	resourceClient := resource.NewClient(clientset, o.dynamicClient)

	discoveryClient, err := f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.dryRunVerifier = cliresource.NewDryRunVerifier(o.dynamicClient, discoveryClient)

	namespace := o.namespace
	if o.allNamespaces {
		namespace = metav1.NamespaceAll
	}

	o.determiner, err = determiner.New(resourceClient, o.result, namespace)
	if err != nil {
		return
	}

	return
}

func (o *Options) completePrinter() (err error) {
	o.printFlags = cmdutil.PrintFlagsWithDryRunStrategy(o.printFlags, o.dryRunStrategy)

	o.printer, err = o.printFlags.ToPrinter()
	if err != nil {
		return err
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
		LabelSelectorParam(o.labelSelector).
		FieldSelectorParam(o.fieldSelector).
		SelectAllParam(o.labelSelector == "" && o.fieldSelector == "").
		ResourceTypeOrNameArgs(false, resourceTypes).
		RequestChunksOf(o.chunkSize).
		Flatten().
		Do()

	return o.result.Err()
}

func (o *Options) Validate(args []string) error {
	if len(args) != 1 && !o.showVersion {
		return errors.New("arguments must be only resource type(s)")
	}

	switch {
	case o.forceDeletion && o.gracePeriod == 0:
		o.Errorf("warning: Immediate deletion does not wait for confirmation that the running resource has been terminated. The resource may continue to run on the cluster indefinitely.\n")
	case o.forceDeletion && o.gracePeriod > 0:
		return fmt.Errorf("--force and --grace-period greater than 0 cannot be specified together")
	}

	return nil
}

func (o *Options) Run(ctx context.Context, f cmdutil.Factory) error {
	deletedInfos := []*cliresource.Info{}
	uidMap := cmdwait.UIDMap{}

	if err := o.result.Visit(func(info *cliresource.Info, err error) error {
		if info.Namespace == metav1.NamespaceSystem {
			return nil // ignore resources in kube-system namespace
		}

		ok, err := o.determiner.DetermineDeletion(ctx, info)
		if err != nil {
			return err
		}
		if !ok {
			return nil // skip deletion
		}

		if o.interactive {
			kind := info.Object.GetObjectKind().GroupVersionKind().Kind
			if ok := prompt.Confirm(fmt.Sprintf("Are you sure to delete %s/%s?", strings.ToLower(kind), info.Name)); !ok {
				return nil // skip deletion
			}
		}

		deletedInfos = append(deletedInfos, info)

		if o.dryRunStrategy == cmdutil.DryRunClient && !o.quiet {
			o.printObj(info.Object)
			return nil // skip deletion
		}
		if o.dryRunStrategy == cmdutil.DryRunServer {
			if err := o.dryRunVerifier.HasSupport(info.Mapping.GroupVersionKind); err != nil {
				return err
			}
		}

		resp, err := cliresource.
			NewHelper(info.Client, info.Mapping).
			DryRun(o.dryRunStrategy == cmdutil.DryRunServer).
			DeleteWithOptions(info.Namespace, info.Name, o.deleteOpts)
		if err != nil {
			return err
		}

		if !o.quiet {
			o.printObj(info.Object)
		}

		loc := cmdwait.ResourceLocation{
			GroupResource: info.Mapping.Resource.GroupResource(),
			Namespace:     info.Namespace,
			Name:          info.Name,
		}
		if status, ok := resp.(*metav1.Status); ok && status.Details != nil {
			uidMap[loc] = status.Details.UID
			return nil
		}

		accessor, err := apimeta.Accessor(resp)
		if err != nil {
			// we don't have UID, but we didn't fail the delete, next best thing is just skipping the UID
			o.Infof("%v\n", err)
			return nil
		}
		uidMap[loc] = accessor.GetUID()

		return nil
	}); err != nil {
		return err
	}

	if !o.waitForDeletion {
		return nil
	}

	o.waitDeletion(uidMap, deletedInfos)

	return nil
}

func (o *Options) waitDeletion(uidMap cmdwait.UIDMap, deletedInfos []*cliresource.Info) {
	timeout := o.timeout
	if timeout == 0 {
		timeout = timeWeek
	}

	waitOpts := cmdwait.WaitOptions{
		ResourceFinder: genericclioptions.ResourceFinderForResult(cliresource.InfoListVisitor(deletedInfos)),
		UIDMap:         uidMap,
		DynamicClient:  o.dynamicClient,
		Timeout:        timeout,
		Printer:        printers.NewDiscardingPrinter(),
		ConditionFn:    cmdwait.IsDeleted,
		IOStreams:      o.IOStreams,
	}
	err := waitOpts.RunWait()
	if apierrors.IsForbidden(err) || apierrors.IsMethodNotSupported(err) {
		// if we're forbidden from waiting, we shouldn't fail.
		// if the resource doesn't support a verb we need, we shouldn't fail.
		o.Errorf("%v\n", err)
	}
}

func (o *Options) Infof(format string, a ...interface{}) {
	fmt.Fprintf(o.Out, format, a...)
}

func (o *Options) Errorf(format string, a ...interface{}) {
	fmt.Fprintf(o.ErrOut, format, a...)
}

func (o *Options) printObj(obj runtime.Object) error {
	return o.printer.PrintObj(obj, o.Out)
}
