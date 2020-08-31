package cmd

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest/fake"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

func TestOptions_Run(t *testing.T) {
	testPods, _, _ := cmdtesting.TestData()
	testPods.Items[0].Status.Phase = corev1.PodFailed  // name="foo"
	testPods.Items[1].Status.Phase = corev1.PodRunning // name="bar"

	type fields struct {
		printFlags     *genericclioptions.PrintFlags
		dryRunStrategy cmdutil.DryRunStrategy
		determiner     *determiner
		result         *resource.Result
		IOStreams      genericclioptions.IOStreams
	}

	tests := []struct {
		name    string
		fields  fields
		wantOut string
		wantErr bool
	}{
		{
			name: "delete pod that should be deleted",
			fields: fields{
				determiner: &determiner{
					pods: func() []*corev1.Pod {
						pods := make([]*corev1.Pod, 0, len(testPods.Items))
						for i := range testPods.Items {
							pods = append(pods, &testPods.Items[i])
						}
						return pods
					}(),
				},
			},
			wantOut: "pod/foo deleted\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := cmdtesting.NewTestFactory().WithNamespace("test")
			defer f.Cleanup()

			codec := scheme.Codecs.LegacyCodec(scheme.Scheme.PrioritizedVersionsAllGroups()...)

			f.UnstructuredClient = &fake.RESTClient{
				NegotiatedSerializer: resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer,
				Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
					switch p, m := req.URL.Path, req.Method; {
					case p == "/namespaces/test/pods" && m == http.MethodGet:
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     cmdtesting.DefaultHeader(),
							Body:       cmdtesting.ObjBody(codec, testPods),
						}, nil

					case p == "/namespaces/test/pods/foo" && m == http.MethodDelete:
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     cmdtesting.DefaultHeader(),
							Body:       cmdtesting.ObjBody(codec, &testPods.Items[0]),
						}, nil

					default:
						t.Errorf("unexpected request: %#v\n%#v", req.URL, req)
					}

					return nil, nil
				}),
			}

			streams, _, out, _ := genericclioptions.NewTestIOStreams()

			o := &Options{
				printFlags:     genericclioptions.NewPrintFlags("deleted").WithTypeSetter(scheme.Scheme),
				namespace:      "test",
				chunkSize:      10,
				dryRunStrategy: tt.fields.dryRunStrategy,
				IOStreams:      streams,
			}

			o.printFlags = cmdutil.PrintFlagsWithDryRunStrategy(o.printFlags, o.dryRunStrategy)
			printer, err := o.printFlags.ToPrinter()
			if err != nil {
				t.Errorf("failed to build printer: %v\n", err)
				return
			}
			o.printObj = func(obj runtime.Object) error {
				return printer.PrintObj(obj, o.Out)
			}

			o.result = f.
				NewBuilder().
				Unstructured().
				ContinueOnError().
				NamespaceParam(o.namespace).
				DefaultNamespace().
				AllNamespaces(o.allNamespaces).
				ResourceTypeOrNameArgs(false, "pods").
				RequestChunksOf(o.chunkSize).
				SelectAllParam(true).
				Flatten().
				Do()

			if err := o.result.Err(); err != nil {
				t.Errorf("failed to fetch resources: %v\n", err)
				return
			}

			if err := o.Run(f); (err != nil) != tt.wantErr {
				t.Errorf("Options.Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantOut, out.String()); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
