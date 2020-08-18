package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

  # Delete Secrets not mounted on any Pods and in the namespace/my-namespace and context/my-context
  $ kubectl prune secret -n my-namespace --context my-context

  # Delete ConfigMaps not mounted on any Pods and across all namespace
  $ kubectl prune cm --all-namespaces

  # Delete Pods not managed by any ReplicaSets and ReplicaSets not managed by any Deployments
  $ kubectl prune po,rs`
)

const (
	kindConfigMap  = "ConfigMap"
	kindSecret     = "Secret"
	kindPod        = "Pod"
	kindReplicaSet = "ReplicaSet"
)

type Options struct {
	configFlags *genericclioptions.ConfigFlags
	printFlags  *genericclioptions.PrintFlags

	namespace     string
	context       string
	allNamespaces bool
	chunkSize     int64

	dryRunStrategy cmdutil.DryRunStrategy
	quiet          bool

	printObj func(obj runtime.Object) error

	genericclioptions.IOStreams
}

func NewOptions(ioStreams genericclioptions.IOStreams) *Options {
	configFlags := genericclioptions.NewConfigFlags(true)

	return &Options{
		configFlags: configFlags,
		printFlags:  genericclioptions.NewPrintFlags("deleted").WithTypeSetter(scheme.Scheme),
		chunkSize:   500,
		IOStreams:   ioStreams,
	}
}

func NewCmdPrune(ioStreams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(ioStreams)

	cmd := &cobra.Command{
		Use:     "kubectl prune TYPE",
		Example: pruneExample,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				fmt.Fprintf(o.ErrOut, "arguments must be only resource type(s)\n")
				return
			}

			cmdutil.CheckErr(o.Complete(cmd))
			cmdutil.CheckErr(o.Run(cmdutil.NewFactory(o.configFlags), args[0]))
		},
	}

	o.configFlags.AddFlags(cmd.Flags())
	o.printFlags.AddFlags(cmd)

	cmdutil.AddDryRunFlag(cmd)
	cmd.Flags().BoolVarP(&o.allNamespaces, "all-namespaces", "A", false, "If true, prune the targeted resources accross all namespace")
	cmd.Flags().BoolVarP(&o.quiet, "quiet", "q", false, "If true, no output is produced")

	return cmd
}

func (o *Options) Complete(cmd *cobra.Command) (err error) {
	clientConfig := o.configFlags.ToRawKubeConfigLoader()

	o.namespace, _, err = clientConfig.Namespace()
	if err != nil {
		return
	}

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return
	}

	o.context = rawConfig.CurrentContext

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

	return
}

func (o *Options) Run(f cmdutil.Factory, resourceTypes string) error {
	r := resource.
		NewBuilder(o.configFlags).
		Unstructured().
		ContinueOnError().
		NamespaceParam(o.namespace).
		DefaultNamespace().
		AllNamespaces(o.allNamespaces).
		ResourceTypes(resourceTypes).
		RequestChunksOf(o.chunkSize).
		SelectAllParam(true).
		Flatten().
		Do()

	err := r.Err()
	if err != nil {
		return err
	}

	var (
		pruneCms     bool
		pruneSecrets bool
		prunePods    bool
		pruneRss     bool
	)

	if err := r.Visit(func(info *resource.Info, err error) error {
		switch info.Object.GetObjectKind().GroupVersionKind().Kind {
		case kindConfigMap:
			pruneCms = true
		case kindSecret:
			pruneSecrets = true
		case kindPod:
			prunePods = true
		case kindReplicaSet:
			pruneRss = true
		}
		return nil
	}); err != nil {
		return err
	}

	clientset, err := f.KubernetesClientSet()
	if err != nil {
		return err
	}

	var (
		// key=ConfigMap.Name
		usedCms = make(map[string]struct{})
		// key=Secret.Name
		usedSecrets map[string]struct{} = make(map[string]struct{})
		// key=ReplicaSet.Name
		rss map[string]struct{} = make(map[string]struct{})
		// key=Deployment.Name
		deploys map[string]struct{} = make(map[string]struct{})
	)

	ctx := context.Background()

	namespace := o.namespace
	if o.allNamespaces {
		namespace = metav1.NamespaceAll
	}

	if pruneCms || pruneSecrets {
		pods, err := listPods(ctx, clientset, namespace)
		if err != nil {
			return err
		}
		if pruneCms {
			usedCms = detectUsedConfigMaps(pods)
		}
		if pruneSecrets {
			sas, err := listServiceAccounts(ctx, clientset, o.namespace)
			if err != nil {
				return err
			}
			usedSecrets = detectUsedSecrets(pods, sas)
		}
	}
	if prunePods {
		resp, err := listReplicaSets(ctx, clientset, namespace)
		if err != nil {
			return err
		}
		for _, v := range resp {
			rss[v.Name] = struct{}{}
		}
	}
	if pruneRss {
		resp, err := listDeployments(ctx, clientset, namespace)
		if err != nil {
			return err
		}
		for _, v := range resp {
			deploys[v.Name] = struct{}{}
		}
	}

	if err := r.Visit(func(info *resource.Info, err error) error {
		if info.Namespace == metav1.NamespaceSystem {
			return nil // ignore resources in kube-system namespace
		}

		switch kind := info.Object.GetObjectKind().GroupVersionKind().Kind; kind {
		case kindConfigMap:
			if _, ok := usedCms[info.Name]; ok {
				return nil
			}
		case kindSecret:
			if _, ok := usedSecrets[info.Name]; ok {
				return nil
			}
		case kindPod:
			unstructured := info.Object.(runtime.Unstructured).UnstructuredContent()
			var pod corev1.Pod
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &pod); err != nil {
				return err
			}
			for _, ownerRef := range pod.OwnerReferences {
				if _, ok := rss[ownerRef.Name]; ok {
					return nil
				}
			}
		case kindReplicaSet:
			unstructured := info.Object.(runtime.Unstructured).UnstructuredContent()
			var rs appsv1.ReplicaSet
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &rs); err != nil {
				return err
			}
			for _, ownerRef := range rs.OwnerReferences {
				if _, ok := deploys[ownerRef.Name]; ok {
					return nil
				}
			}
		default:
			return fmt.Errorf("unsupported kind: %s", kind)
		}

		if o.dryRunStrategy == cmdutil.DryRunClient && !o.quiet {
			o.printObj(info.Object)
			return nil
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
