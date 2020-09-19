package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cliresource "k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest/fake"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/micnncim/kubectl-prune/pkg/determiner"
)

func TestOptions_Run(t *testing.T) {
	const (
		fakeNamespace                = "fake-ns"
		fakeResourceType             = "pod"
		fakeResourceTypePlural       = "pods"
		fakeObjectToBeDeleted1Name   = "fake-pod-to-be-deleted-1"
		fakeObjectToBeDeleted2Name   = "fake-pod-to-be-deleted-2"
		fakeObjectNotToBeDeletedName = "fake-pod-not-to-be-deleted"
	)

	var (
		fakeObjectToBeDeleted1 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fakeObjectToBeDeleted1Name,
				Namespace: fakeNamespace,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}
		fakeObjectToBeDeleted2 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fakeObjectToBeDeleted2Name,
				Namespace: fakeNamespace,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}
		fakeObjectNotToBeDeleted = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fakeObjectNotToBeDeletedName,
				Namespace: fakeNamespace,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}
		fakeObjectList = &corev1.PodList{
			Items: []corev1.Pod{
				*fakeObjectToBeDeleted1,
				*fakeObjectToBeDeleted2,
				*fakeObjectNotToBeDeleted,
			},
		}
		fakeObjectMap = map[string]*corev1.Pod{
			fakeObjectToBeDeleted1Name:   fakeObjectToBeDeleted1,
			fakeObjectToBeDeleted2Name:   fakeObjectToBeDeleted2,
			fakeObjectNotToBeDeletedName: fakeObjectNotToBeDeleted,
		}
	)

	tf := cmdtesting.NewTestFactory().WithNamespace(fakeNamespace)
	defer tf.Cleanup()

	codec := scheme.Codecs.LegacyCodec(scheme.Scheme.PrioritizedVersionsAllGroups()...)

	tf.UnstructuredClient = &fake.RESTClient{
		NegotiatedSerializer: cliresource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			switch p, m := req.URL.Path, req.Method; {
			case p == fmt.Sprintf("/namespaces/%s/%s", fakeNamespace, fakeResourceTypePlural) && m == http.MethodGet:
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     cmdtesting.DefaultHeader(),
					Body:       cmdtesting.ObjBody(codec, fakeObjectList),
				}, nil

			case strings.HasPrefix(p, fmt.Sprintf("/namespaces/%s/%s/", fakeNamespace, fakeResourceTypePlural)) && m == http.MethodDelete:
				s := strings.Split(p, "/")
				objName := s[len(s)-1]
				obj, ok := fakeObjectMap[objName]
				if !ok {
					t.Errorf("unexpected object: %s", objName)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     cmdtesting.DefaultHeader(),
					Body:       cmdtesting.ObjBody(codec, obj),
				}, nil

			default:
				t.Errorf("unexpected request: %#v\n%#v", req.URL, req)
			}

			return nil, nil
		}),
	}

	type fields struct {
		dryRunStrategy cmdutil.DryRunStrategy
	}

	tests := []struct {
		name    string
		fields  fields
		wantOut string
		wantErr bool
	}{
		{
			name:   "delete resources that should be deleted",
			fields: fields{},
			wantOut: makeOperationMessage(
				fakeResourceType,
				[]string{
					fakeObjectToBeDeleted1Name,
					fakeObjectToBeDeleted2Name,
				},
				printedOperationTypeDeleted,
				cmdutil.DryRunNone,
			),
			wantErr: false,
		},
		{
			name: "does not delete resources that should be deleted when dry-run is set as client",
			fields: fields{
				dryRunStrategy: cmdutil.DryRunClient,
			},
			wantOut: makeOperationMessage(
				fakeResourceType,
				[]string{fakeObjectToBeDeleted1Name, fakeObjectToBeDeleted2Name},
				printedOperationTypeDeleted,
				cmdutil.DryRunClient,
			),
			wantErr: false,
		},
		{
			name: "does not delete resources that should be deleted when dry-run is set as server",
			fields: fields{
				dryRunStrategy: cmdutil.DryRunServer,
			},
			wantOut: makeOperationMessage(
				fakeResourceType,
				[]string{fakeObjectToBeDeleted1Name, fakeObjectToBeDeleted2Name},
				printedOperationTypeDeleted,
				cmdutil.DryRunServer,
			),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			streams, _, out, _ := genericclioptions.NewTestIOStreams()

			o := &Options{
				printFlags:     genericclioptions.NewPrintFlags(printedOperationTypeDeleted).WithTypeSetter(scheme.Scheme),
				namespace:      fakeNamespace,
				chunkSize:      10,
				determiner:     &determiner.Determiner{},
				dryRunStrategy: tt.fields.dryRunStrategy,
				IOStreams:      streams,
			}

			if err := o.completePrintObj(); err != nil {
				t.Errorf("failed to complete printObj: %v\n", err)
				return
			}

			if err := o.completeResources(tf, fakeResourceTypePlural); err != nil {
				t.Errorf("failed to complete resources: %v\n", err)
				return
			}

			if err := o.Run(context.Background(), tf); (err != nil) != tt.wantErr {
				t.Errorf("Options.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.wantOut, out.String()); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
				return
			}
		})
	}
}

func makeOperationMessage(resourceType string, objectNames []string, operation string, dryRunStrategy cmdutil.DryRunStrategy) string {
	b := strings.Builder{}

	for _, name := range objectNames {
		msg := fmt.Sprintf("%s/%s %s", resourceType, name, operation)
		switch dryRunStrategy {
		case cmdutil.DryRunClient:
			msg += " (dry run)"
		case cmdutil.DryRunServer:
			msg += " (server dry run)"
		}
		msg += "\n"

		b.WriteString(msg)
	}

	return b.String()
}
