package integrations

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	backstageLabelKey = "backstage.io/kubernetes-id"
)

type Integrations struct {
	restMapper meta.RESTMapper
	client     client.Client
	ctx        context.Context
	logger     logr.Logger
}

func NewIntegrations(client client.Client, ctx context.Context) (*Integrations, error) {
	restMapper, err := cmdutil.NewFactory(genericclioptions.NewConfigFlags(true)).ToRESTMapper()
	if err != nil {
		return &Integrations{}, err
	}
	return &Integrations{
		restMapper: restMapper,
		client:     client,
		ctx:        ctx,
		logger:     log.Log.WithName("Integrations"),
	}, nil
}

func (i *Integrations) BackstageLabel(result v1alpha1.ResultSpec) map[string]string {
	namespace, resourceName, _ := strings.Cut(result.Name, "/")
	// Log and don't propagate errors so we won't trigger a new reconciliation
	gvr, err := i.restMapper.ResourceFor(schema.GroupVersionResource{
		Resource: result.Kind,
	})
	if err != nil {
		i.logger.Error(err, "Unable to find Kind", "kind", result.Kind)
		return map[string]string{}
	}

	obj := &unstructured.Unstructured{}
	gvk := schema.GroupVersionKind{
		Group:   gvr.Group,
		Kind:    result.Kind,
		Version: gvr.Version,
	}
	obj.SetGroupVersionKind(gvk)
	// Retrieve the resource using the client
	err = i.client.Get(i.ctx, client.ObjectKey{Name: resourceName, Namespace: namespace}, obj)
	// if we don't find the K8s object we won't trigger a new reconciliation and just log a message
	if err != nil {
		i.logger.Error(err, "Fail to retrieve resource", "resource", resourceName, "namespace", namespace)
		return map[string]string{}
	}
	labels := obj.GetLabels()
	value, exists := labels[backstageLabelKey]
	if !exists {
		i.logger.Info("Label key does not exist in resource", "labelKey", backstageLabelKey, "kind", result.Kind, "resource", resourceName)
	}
	// Assign the same label key/value to result CR
	return map[string]string{backstageLabelKey: value}
}
