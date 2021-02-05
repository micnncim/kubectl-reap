package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	"github.com/micnncim/kubectl-reap/pkg/determiner"
	"github.com/micnncim/kubectl-reap/pkg/prompt"
	"github.com/micnncim/kubectl-reap/pkg/resource"
	"github.com/micnncim/kubectl-reap/pkg/version"
)

const (
	reapShortDescription = `
Delete unused resources. Supported resources:

- Pods (whose status is not Running)
- ConfigMaps (not used by any Pods)
- Secrets (not used by any Pods or ServiceAccounts)
- PersistentVolumes (not satisfying any PersistentVolumeClaims)
- PersistentVolumeClaims (not used by any Pods)
- Jobs (completed)
- PodDisruptionBudgets (not targeting any Pods)
- HorizontalPodAutoscalers (not targeting any resources)
`

	reapExample = `
  # Delete ConfigMaps not mounted on any Pods and in the current namespace and context
  $ kubectl reap configmaps

  # Delete unused ConfigMaps and Secrets in the namespace/my-namespace and context/my-context
  $ kubectl reap cm,secret -n my-namespace --context my-context

  # Delete ConfigMaps not mounted on any Pods and across all namespace
  $ kubectl reap cm --all-namespaces

  # Delete Pods whose status is not Running as client-side dry-run
  $ kubectl reap po --dry-run=client`

	// printedOperationTypeDeleted is used when printer outputs the result of operations.
	printedOperationTypeDeleted = "deleted"
)

var timeWeek = 168 * time.Hour

type runner struct {
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags

	namespace        string
	allNamespaces    bool
	chunkSize        int64
	labelSelector    string
	fieldSelector    string
	gracePeriod      int
	forceDeletion    bool
	needWaitDeletion bool
	timeout          time.Duration

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

func newRunner(ioStreams genericclioptions.IOStreams) *runner {
	return &runner{
		configFlags: genericclioptions.NewConfigFlags(true),
		printFlags:  genericclioptions.NewPrintFlags(printedOperationTypeDeleted).WithTypeSetter(scheme.Scheme),
		chunkSize:   500,
		IOStreams:   ioStreams,
	}
}

func NewCmdReap(streams genericclioptions.IOStreams) *cobra.Command {
	r := newRunner(streams)

	cmd := &cobra.Command{
		Use:     "kubectl reap RESOURCE_TYPE",
		Short:   reapShortDescription,
		Example: reapExample,
		Run: func(cmd *cobra.Command, args []string) {
			if r.showVersion {
				r.Infof("%s (%s)\n", version.Version, version.Revision)
				return
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-ch
				r.Infof("Canceling execution...\n")
				cancel()
			}()

			f := cmdutil.NewFactory(r.configFlags)

			cmdutil.CheckErr(r.Validate(args))
			cmdutil.CheckErr(r.Complete(f, args, cmd))
			cmdutil.CheckErr(r.Run(ctx, f))
		},
	}

	r.configFlags.AddFlags(cmd.Flags())
	r.printFlags.AddFlags(cmd)

	cmdutil.AddDryRunFlag(cmd)

	cmd.Flags().BoolVarP(&r.allNamespaces, "all-namespaces", "A", false, "If true, delete the targeted resources across all namespace except kube-system")
	cmd.Flags().StringVarP(&r.labelSelector, "selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().StringVar(&r.fieldSelector, "field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	cmd.Flags().IntVar(&r.gracePeriod, "grace-period", -1, "Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
	cmd.Flags().BoolVar(&r.forceDeletion, "force", false, "If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
	cmd.Flags().BoolVar(&r.needWaitDeletion, "wait", false, "If true, wait for resources to be gone before returning. This waits for finalizers.")
	cmd.Flags().DurationVar(&r.timeout, "timeout", 0, "The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
	cmd.Flags().BoolVarP(&r.quiet, "quiet", "q", false, "If true, no output is produced")
	cmd.Flags().BoolVarP(&r.interactive, "interactive", "i", false, "If true, a prompt asks whether resources can be deleted")
	cmd.Flags().BoolVar(&r.showVersion, "version", false, "If true, show the version of this plugin")

	return cmd
}

func (r *runner) Complete(f cmdutil.Factory, args []string, cmd *cobra.Command) (err error) {
	if !r.forceDeletion && r.gracePeriod == 0 {
		// To preserve backwards compatibility, but prevent accidental data loss, we convert --grace-period=0
		// into --grace-period=1. Users may provide --force to bypass this conversion.
		r.gracePeriod = 1
	}
	if r.forceDeletion && r.gracePeriod < 0 {
		r.gracePeriod = 0
	}
	r.deleteOpts = &metav1.DeleteOptions{}
	if r.gracePeriod >= 0 {
		r.deleteOpts = metav1.NewDeleteOptions(int64(r.gracePeriod))
	}

	r.namespace, _, err = r.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return
	}

	r.dryRunStrategy, err = cmdutil.GetDryRunStrategy(cmd)
	if err != nil {
		return
	}

	if err = r.completePrinter(); err != nil {
		return
	}

	if err = r.completeResources(f, args[0]); err != nil {
		return
	}

	clientset, err := f.KubernetesClientSet()
	if err != nil {
		return
	}
	r.dynamicClient, err = f.DynamicClient()
	if err != nil {
		return
	}
	resourceClient := resource.NewClient(clientset, r.dynamicClient)

	discoveryClient, err := f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	r.dryRunVerifier = cliresource.NewDryRunVerifier(r.dynamicClient, discoveryClient)

	namespace := r.namespace
	if r.allNamespaces {
		namespace = metav1.NamespaceAll
	}

	r.determiner, err = determiner.New(resourceClient, r.result, namespace)
	if err != nil {
		return
	}

	return
}

func (r *runner) completePrinter() (err error) {
	r.printFlags = cmdutil.PrintFlagsWithDryRunStrategy(r.printFlags, r.dryRunStrategy)

	r.printer, err = r.printFlags.ToPrinter()
	if err != nil {
		return err
	}

	return nil
}

func (r *runner) completeResources(f cmdutil.Factory, resourceTypes string) error {
	r.result = f.
		NewBuilder().
		Unstructured().
		ContinueOnError().
		NamespaceParam(r.namespace).
		DefaultNamespace().
		AllNamespaces(r.allNamespaces).
		LabelSelectorParam(r.labelSelector).
		FieldSelectorParam(r.fieldSelector).
		SelectAllParam(r.labelSelector == "" && r.fieldSelector == "").
		ResourceTypeOrNameArgs(false, resourceTypes).
		RequestChunksOf(r.chunkSize).
		Flatten().
		Do()

	return r.result.Err()
}

func (r *runner) Validate(args []string) error {
	if len(args) != 1 && !r.showVersion {
		return errors.New("arguments must be only resource type(s)")
	}

	switch {
	case r.forceDeletion && r.gracePeriod == 0:
		r.Errorf("warning: Immediate deletion does not wait for confirmation that the running resource has been terminated. The resource may continue to run on the cluster indefinitely.\n")
	case r.forceDeletion && r.gracePeriod > 0:
		return fmt.Errorf("--force and --grace-period greater than 0 cannot be specified together")
	}

	return nil
}

func (r *runner) Run(ctx context.Context, f cmdutil.Factory) error {
	deletedInfos := []*cliresource.Info{}
	uidMap := cmdwait.UIDMap{}

	if err := r.result.Visit(func(info *cliresource.Info, err error) error {
		if info.Namespace == metav1.NamespaceSystem {
			return nil // ignore resources in kube-system namespace
		}

		ok, err := r.determiner.DetermineDeletion(ctx, info)
		if err != nil {
			return err
		}
		if !ok {
			return nil // skip deletion
		}

		if r.interactive {
			kind := info.Object.GetObjectKind().GroupVersionKind().Kind
			if ok := prompt.Confirm(fmt.Sprintf("Are you sure to delete %s/%s?", strings.ToLower(kind), info.Name)); !ok {
				return nil // skip deletion
			}
		}

		deletedInfos = append(deletedInfos, info)

		if r.dryRunStrategy == cmdutil.DryRunClient && !r.quiet {
			r.printObj(info.Object)
			return nil // skip deletion
		}
		if r.dryRunStrategy == cmdutil.DryRunServer {
			if err := r.dryRunVerifier.HasSupport(info.Mapping.GroupVersionKind); err != nil {
				return err
			}
		}

		resp, err := cliresource.
			NewHelper(info.Client, info.Mapping).
			DryRun(r.dryRunStrategy == cmdutil.DryRunServer).
			DeleteWithOptions(info.Namespace, info.Name, r.deleteOpts)
		if err != nil {
			return err
		}

		if !r.quiet {
			r.printObj(info.Object)
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
			r.Infof("%v\n", err)
			return nil
		}
		uidMap[loc] = accessor.GetUID()

		return nil
	}); err != nil {
		return err
	}

	if !r.needWaitDeletion {
		return nil
	}

	r.waitDeletion(uidMap, deletedInfos)

	return nil
}

func (r *runner) waitDeletion(uidMap cmdwait.UIDMap, deletedInfos []*cliresource.Info) {
	timeout := r.timeout
	if timeout == 0 {
		timeout = timeWeek
	}

	waitOpts := cmdwait.WaitOptions{
		ResourceFinder: genericclioptions.ResourceFinderForResult(cliresource.InfoListVisitor(deletedInfos)),
		UIDMap:         uidMap,
		DynamicClient:  r.dynamicClient,
		Timeout:        timeout,
		Printer:        printers.NewDiscardingPrinter(),
		ConditionFn:    cmdwait.IsDeleted,
		IOStreams:      r.IOStreams,
	}
	err := waitOpts.RunWait()
	if apierrors.IsForbidden(err) || apierrors.IsMethodNotSupported(err) {
		// if we're forbidden from waiting, we shouldn't fail.
		// if the resource doesn't support a verb we need, we shouldn't fail.
		r.Errorf("%v\n", err)
	}
}

func (r *runner) Infof(format string, a ...interface{}) {
	fmt.Fprintf(r.Out, format, a...)
}

func (r *runner) Errorf(format string, a ...interface{}) {
	fmt.Fprintf(r.ErrOut, format, a...)
}

func (r *runner) printObj(obj runtime.Object) error {
	return r.printer.PrintObj(obj, r.Out)
}
