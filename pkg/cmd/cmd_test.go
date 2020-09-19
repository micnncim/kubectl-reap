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
	"k8s.io/apimachinery/pkg/runtime"
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
		fakeAPIVersion               = "v1"
		fakeKind                     = "Pod"
		fakeResourceType             = "pod"
		fakeResourceTypePlural       = "pods"
		fakeObjectToBeDeleted1Name   = "fake-obj-to-be-deleted-1"
		fakeObjectToBeDeleted2Name   = "fake-obj-to-be-deleted-2"
		fakeObjectNotToBeDeletedName = "fake-obj-not-to-be-deleted"
	)

	fakeObjectBase := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       fakeKind,
			APIVersion: fakeAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: fakeNamespace,
		},
	}

	fakeObjectToBeDeleted1 := fakeObjectBase.DeepCopy()
	fakeObjectToBeDeleted1.Name = fakeObjectToBeDeleted1Name

	fakeObjectToBeDeleted2 := fakeObjectBase.DeepCopy()
	fakeObjectToBeDeleted2.Name = fakeObjectToBeDeleted2Name

	fakeObjectNotToBeDeleted := fakeObjectBase.DeepCopy()
	fakeObjectNotToBeDeleted.Name = fakeObjectNotToBeDeletedName

	fakeObjectList := &corev1.PodList{
		Items: []corev1.Pod{
			*fakeObjectToBeDeleted1,
			*fakeObjectToBeDeleted2,
			*fakeObjectNotToBeDeleted,
		},
	}
	fakeObjectMap := map[string]*corev1.Pod{
		fakeObjectToBeDeleted1Name:   fakeObjectToBeDeleted1,
		fakeObjectToBeDeleted2Name:   fakeObjectToBeDeleted2,
		fakeObjectNotToBeDeletedName: fakeObjectNotToBeDeleted,
	}

	testFactory := cmdtesting.NewTestFactory().WithNamespace(fakeNamespace)
	defer testFactory.Cleanup()

	codec := scheme.Codecs.LegacyCodec(scheme.Scheme.PrioritizedVersionsAllGroups()...)

	testFactory.UnstructuredClient = &fake.RESTClient{
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
					return nil, nil
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

	fakeDeterminer, err := determiner.NewFakeDeterminer([]runtime.Object{fakeObjectToBeDeleted1, fakeObjectToBeDeleted2}...)
	if err != nil {
		t.Fatalf("failed to construct fake determiner")
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
				determiner:     fakeDeterminer,
				dryRunStrategy: tt.fields.dryRunStrategy,
				IOStreams:      streams,
			}

			if err := o.completePrintObj(); err != nil {
				t.Errorf("failed to complete printObj: %v\n", err)
				return
			}

			if err := o.completeResources(testFactory, fakeResourceTypePlural); err != nil {
				t.Errorf("failed to complete resources: %v\n", err)
				return
			}

			if err := o.Run(context.Background(), testFactory); (err != nil) != tt.wantErr {
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
