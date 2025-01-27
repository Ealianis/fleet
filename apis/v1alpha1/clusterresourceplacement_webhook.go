/*
Copyright (c) Microsoft Corporation.
Licensed under the MIT license.
*/

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiErrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"go.goms.io/fleet/pkg/utils/informer"
)

var ResourceInformer informer.Manager

func (c *ClusterResourcePlacement) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}

var _ webhook.Validator = &ClusterResourcePlacement{}

func (c *ClusterResourcePlacement) ValidateCreate() error {
	return ValidateClusterResourcePlacement(c)
}

func (c *ClusterResourcePlacement) ValidateUpdate(old runtime.Object) error {
	// TODO: validate changes against old if needed
	return ValidateClusterResourcePlacement(c)
}

func (c *ClusterResourcePlacement) ValidateDelete() error {
	// do nothing for delete request
	return nil
}

// ValidateClusterResourcePlacement validate a ClusterResourcePlacement object
func ValidateClusterResourcePlacement(clusterResourcePlacement *ClusterResourcePlacement) error {
	allErr := make([]error, 0)

	for _, selector := range clusterResourcePlacement.Spec.ResourceSelectors {
		if selector.LabelSelector != nil {
			if len(selector.Name) != 0 {
				allErr = append(allErr, fmt.Errorf("the labelSelector and name fields are mutually exclusive in selector %+v", selector))
			}
			if _, err := metav1.LabelSelectorAsSelector(selector.LabelSelector); err != nil {
				allErr = append(allErr, fmt.Errorf("the labelSelector in resource selector %+v is invalid: %w", selector, err))
			}
		}
	}

	if clusterResourcePlacement.Spec.Policy != nil && clusterResourcePlacement.Spec.Policy.Affinity != nil &&
		clusterResourcePlacement.Spec.Policy.Affinity.ClusterAffinity != nil {
		for _, selector := range clusterResourcePlacement.Spec.Policy.Affinity.ClusterAffinity.ClusterSelectorTerms {
			if _, err := metav1.LabelSelectorAsSelector(&selector.LabelSelector); err != nil {
				allErr = append(allErr, fmt.Errorf("the labelSelector in cluster selector %+v is invalid: %w", selector, err))
			}
		}
	}

	// we leverage the informermanager in the changedetector controller to do the resource scope validation
	if ResourceInformer == nil {
		allErr = append(allErr, fmt.Errorf("cannot perform resource scope check for now, please retry"))
	} else {
		for _, selector := range clusterResourcePlacement.Spec.ResourceSelectors {
			gvk := schema.GroupVersionKind{
				Group:   selector.Group,
				Version: selector.Version,
				Kind:    selector.Kind,
			}

			if !ResourceInformer.IsClusterScopedResources(gvk) {
				allErr = append(allErr, fmt.Errorf("the resource is not found in schema (please retry) or it is not a cluster scoped resource: %v", gvk))
			}
		}
	}

	return apiErrors.NewAggregate(allErr)
}
