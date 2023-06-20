package controller

import (
	"context"
	"fmt"

	"github.com/ca-gip/kotary/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	cagipv1 "github.com/ca-gip/kotary/pkg/apis/cagip/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// syncHandlerNS creates default ResourceQuotaClaims inside managed NS
func (c *Controller) syncHandlerNS(key string) error {

	// Get the Namespace Resource with the key
	ns, err := c.namespaceLister.Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("namespace '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// Check if the namespace should be treated by this controller
	if !hasTargetedLabel(ns) {
		return nil
	}

	// Check if there is already an existing resource quota
	resourceQuotaExist, err := c.hasResourceQuota(ns)
	if err != nil {
		return err
	}

	// Check if there is already an existing resource quota
	claimWithoutStatusExist, err := c.hasUnevaluatedClaim(ns)
	if err != nil {
		return err
	}

	// The NS does not have quota and pending claims
	// We create a default claim to add one
	if !resourceQuotaExist && !claimWithoutStatusExist {
		klog.V(4).Infof("No Default ResourceQuota for ns %s", ns.Name)
		_, err = c.resourcequotaclaimclientset.CagipV1().ResourceQuotaClaims(ns.Name).Create(context.TODO(), c.newDefaultResourceQuotaClaim(ns.Name), metav1.CreateOptions{})

		// Just in case if the ResourceQuotaClaim already resourceQuotaExist we skip it
		if errors.IsAlreadyExists(err) {
			return nil
		}

		// Could not be created the item is requeue
		if err != nil {
			klog.V(4).Infof("Error creating ResourceQuotaClaim for ns %s", ns.Name)
			utils.DefaultClaimCounter.WithLabelValues("error")
			return err
		}
		utils.DefaultClaimCounter.WithLabelValues("success")

	}

	return nil

}

// Check if the namespace has already a quota
func (c *Controller) hasResourceQuota(ns *v1.Namespace) (bool, error) {
	// Gather all the resources quota on the namespace
	resourceQuotas, err := c.resourceQuotaLister.ResourceQuotas(ns.Name).List(utils.DefaultLabelSelector())

	// Could not find existing resourceQuotas requeue
	if err != nil {
		return false, err
	}

	// Create label to be checked
	controllerLabelOwnership := map[string]string{
		"creator": utils.ControllerName,
	}

	// TODO : Should we check for all or only the rq created by this controller
	for _, resourceQuota := range resourceQuotas {
		if utils.MapIntersects(resourceQuota.Labels, controllerLabelOwnership) {
			return true, nil
		}
	}

	return false, nil
}

// Check if there are unevaluated claims in the NS
func (c *Controller) hasUnevaluatedClaim(ns *v1.Namespace) (bool, error) {

	// Empty selector
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{})

	// List claims
	claims, err := c.resourceQuotaClaimLister.ResourceQuotaClaims(ns.Name).List(selector)

	if err != nil {
		return false, err
	}

	// Return true if a claim is not Rejected
	for _, claim := range claims {
		notReject := claim.Status.Phase != cagipv1.PhaseRejected
		notAccepted := claim.Status.Phase != cagipv1.PhaseAccepted
		if notReject && notAccepted {
			return true, nil
		}
	}

	return false, nil

}

// Check if the namespace show be watch based on the condition that it
// contains the annotation
func hasTargetedLabel(namespace *v1.Namespace) bool {
	// Create label to be checked
	labelToFind := map[string]string{
		"quota": "managed",
	}

	// Check if the Namespace contains the label
	return utils.MapIntersects(namespace.Labels, labelToFind)
}

// Return a default quota
func (c *Controller) newDefaultResourceQuotaClaim(namespace string) *cagipv1.ResourceQuotaClaim {
	return &cagipv1.ResourceQuotaClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
		},
		Spec: c.settings.DefaultClaimSpec.DeepCopy(),
	}
}
