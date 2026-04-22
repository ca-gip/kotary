package controller

import (
	"context"
	"fmt"
	"math"

	"github.com/ca-gip/kotary/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	quota "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	cagipv1 "github.com/ca-gip/kotary/pkg/apis/cagip/v1"
	v1Core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Handle claims from the workqueue
func (c *Controller) syncHandlerClaim(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the ResourceQuotaClaims resource with this namespace/name
	claim, err := c.resourceQuotaClaimLister.ResourceQuotaClaims(namespace).Get(name)
	if err != nil {
		// The ResourceQuotaClaims resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("ResourceQuotaClaims '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// TODO : Add feature gate
	// Get the managed quota
	// It there was an error different than not found the error is return
	// If it was found it's possible to check scaledown
	managedQuota, err := c.resourceQuotaLister.ResourceQuotas(claim.Namespace).Get(utils.ResourceQuotaName)
	if !errors.IsNotFound(err) && err != nil {
		return err
	} else if !errors.IsNotFound(err) {
		// List pod in the claim ns
		pods, err := c.podsLister.Pods(claim.Namespace).List(utils.DefaultLabelSelector())

		pods = utils.FilterRunningPods(pods) // Keep only Running Pods

		if err != nil {
			return err
		}
		// Check is the quota is scaling down
		// If scaling down checks if the claim is higher than the total amount of request on the NS
		if isDownscale := isDownscaleQuota(claim, managedQuota); isDownscale {
			if msg := canDownscaleQuota(claim, utils.TotalRequestNS(pods)); msg != utils.EmptyMsg {
				err = c.claimPending(claim, msg)
				return err
			}
		}
	}

	// Gather Nodes and ResourceQuota ResourceList to evaluate if there is enough capacity to accept
	// the ResourceQuotaClaim
	availableResources, err := c.nodesTotalCapacity()
	if err != nil {
		return err
	}

	// Gather ResourceQuotas on the cluster minus the one of the namespace that is being evaluated
	reservedResources, err := c.totalResourceQuota(claim)
	if err != nil {
		return err
	}

	// Check that the claim respect the allocation limit
	// If it does not the claim is rejected
	if msg := c.checkAllocationLimit(claim, availableResources); msg != utils.EmptyMsg {
		err := c.claimRejected(claim, msg)
		return err
	}

	// Check that there are enough resources to fit the claim
	// If it does not the claim is rejected
	if msg := c.checkResourceFit(claim, availableResources, reservedResources); msg != utils.EmptyMsg {
		err := c.claimRejected(claim, msg)
		return err
	}

	// The claim has passed the verification

	// The managed quota is updated
	err = c.updateResourceQuota(claim)
	if err != nil {
		return err
	}

	// The claim is removed
	err = c.deleteResourceQuotaClaim(claim)
	if err != nil {
		return err
	}

	utils.ClaimCounter.WithLabelValues("success").Inc()
	klog.Infof("< RequestQuotaClaim '%s' ACCEPTED >", claim.Name)

	// Everything went well
	return nil
}

// Update the ResourceQuotaClaimStatus
func (c *Controller) updateResourceQuotaClaimStatus(claim *cagipv1.ResourceQuotaClaim, phase string, details string) (claimCopy *cagipv1.ResourceQuotaClaim, err error) {

	// DeepCopy of the original claim, very important has we area dealing with a SharedInformer
	claimCopy = claim.DeepCopy()

	// Update to the specified Phase
	claimCopy.Status = cagipv1.ResourceQuotaClaimStatus{
		Phase:   phase,
		Details: details,
	}

	// ResourceQuotaClaimStatus feature gate is enabled,
	// we must use UpdateStatus instead of Update to update the Status block.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err = c.resourcequotaclaimclientset.CagipV1().ResourceQuotaClaims(claim.Namespace).UpdateStatus(context.TODO(), claimCopy, metav1.UpdateOptions{})

	// Checking we were able to update the Status, will enable requeue if it's not the case
	if err != nil {
		klog.Errorf("Could not update phase on %s/%s ", claim.Name, claim.Namespace)
		return claimCopy, err
	}

	klog.V(6).Infof("Updated phase on %s/%s ", claim.Name, claim.Namespace)

	return claimCopy, nil

}

// Update claim phase to Rejected with a msg
func (c *Controller) claimRejected(claim *cagipv1.ResourceQuotaClaim, msg string) (err error) {
	klog.Infof("< RequestQuotaClaim '%s' set to REJECTED >", claim.Name)
	// Notify via an event
	c.recorder.Event(claim, v1Core.EventTypeWarning, cagipv1.PhaseRejected, msg)
	// Update ResourceQuotaClaim Status to Rejected Phase
	_, err = c.updateResourceQuotaClaimStatus(claim, cagipv1.PhaseRejected, msg)
	utils.ClaimCounter.WithLabelValues("rejected").Inc()
	return
}

// Update claim phase to Pending with a msg
func (c *Controller) claimPending(claim *cagipv1.ResourceQuotaClaim, msg string) (err error) {
	klog.Infof("< RequestQuotaClaim '%s' set to PENDING >", claim.Name)
	// Notify via an event
	c.recorder.Event(claim, v1Core.EventTypeWarning, cagipv1.PhasePending, msg)
	// Update ResourceQuotaClaim Status to Rejected Phase
	_, err = c.updateResourceQuotaClaimStatus(claim, cagipv1.PhasePending, msg)
	utils.ClaimCounter.WithLabelValues("pending").Inc()
	return
}

// Update the specification of the managed-quota
func (c *Controller) updateResourceQuota(claim *cagipv1.ResourceQuotaClaim) error {
	// Get the Managed ResourceQuota for the current ns
	resourceQuota, err := c.resourceQuotaLister.ResourceQuotas(claim.Namespace).Get(utils.ResourceQuotaName)

	// If the resource doesn't exist, we create it
	if errors.IsNotFound(err) {
		klog.V(4).Infof("No existing ResourceQuota for ns %s", claim.Namespace)
		resourceQuota, err = c.resourcequotaclientset.CoreV1().ResourceQuotas(claim.Namespace).Create(context.TODO(), newResourceQuota(claim), metav1.CreateOptions{})

		// If an error occurs during Create, the item is requeue
		if err != nil {
			klog.V(4).Infof("Error creating ResourceQuota for ns %s", claim.Namespace)
			return err
		}
	} else if !quota.Equals(resourceQuota.Status.Hard, claim.Spec) {
		// If this spec of the ResourceQuota is not the desired one we update it
		klog.V(4).Infof("ResourceQuota not synced, updating for ns %s", claim.Namespace)
		_, err := c.resourcequotaclientset.CoreV1().ResourceQuotas(claim.Namespace).Update(context.TODO(), newResourceQuota(claim), metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Could not update ResourceQuotas for ns %s ", claim.Annotations)
			// If an error occurs during Create, the item is requeue
			return err
		}
	}

	return err
}

// Apply over provisioning on a resource list
func (c *Controller) applyOverProvisioning(current *v1Core.ResourceList) (overProvisioned *v1Core.ResourceList) {
	return &v1Core.ResourceList{
		v1Core.ResourceCPU: *resource.NewMilliQuantity(
			int64(math.Round(float64(current.Cpu().MilliValue())*c.settings.RatioOverCommitCPU)),
			resource.DecimalSI),
		v1Core.ResourceMemory: *resource.NewQuantity(
			int64(math.Round(float64(current.Memory().Value())*c.settings.RatioOverCommitMemory)),
			resource.BinarySI),
	}
}

// Check if a claim is under the allocation limit
// If it doesn't comply return an error msg
// Otherwise return an empty msg
func (c *Controller) checkAllocationLimit(claim *cagipv1.ResourceQuotaClaim, availableResources *v1Core.ResourceList) string {

	allocationLimit := &v1Core.ResourceList{
		v1Core.ResourceMemory: *resource.NewQuantity(
			int64(math.Round(float64(availableResources.Memory().Value())*c.settings.RatioMaxAllocationMemory)),
			resource.BinarySI),
		v1Core.ResourceCPU: *resource.NewMilliQuantity(
			int64(math.Round(float64(availableResources.Cpu().MilliValue())*c.settings.RatioMaxAllocationCPU)),
			resource.DecimalSI),
	}

	if claim.Spec.Memory().Value() > allocationLimit.Memory().Value() {
		return fmt.Sprintf(utils.MessageMemoryAllocationLimit,
			claim.Spec.Memory().String(),
			utils.BytesSize(float64(allocationLimit.Memory().Value())))
	}

	if claim.Spec.Cpu().MilliValue() > allocationLimit.Cpu().MilliValue() {
		return fmt.Sprintf(utils.MessageCpuAllocationLimit,
			claim.Spec.Cpu().String(),
			allocationLimit.Cpu().String())
	}

	return utils.EmptyMsg
}

// Check that they are enough resources to fit the claim
func (c *Controller) checkResourceFit(claim *cagipv1.ResourceQuotaClaim, availableResources *v1Core.ResourceList, reservedResources *v1Core.ResourceList) string {

	// Apply OverProvisioning
	overCommittedResources := c.applyOverProvisioning(availableResources)

	// Calculate
	freeResources := quota.SubtractWithNonNegativeResult(*overCommittedResources, *reservedResources)

	// Difference between the freeResources and the claim
	// If diff is different than zero there is enough capacity to accept the claim
	diff := quota.SubtractWithNonNegativeResult(freeResources, claim.Spec)

	// ResourceQuotaClaims cannot fit because of Memory
	if diff.Memory().IsZero() {
		return fmt.Sprintf(utils.MessageRejectedMemory,
			claim.Spec.Memory().String(),
			utils.BytesSize(float64(freeResources.Memory().Value())))
	}

	// ResourceQuotaClaims cannot fit because of CPU
	if diff.Cpu().IsZero() {
		return fmt.Sprintf(utils.MessageRejectedCPU,
			claim.Spec.Cpu().String(),
			freeResources.Cpu().String())
	}

	return utils.EmptyMsg
}

// Gather the nodes total capacity
func (c *Controller) nodesTotalCapacity() (total *v1Core.ResourceList, err error) {

	// Get worker Nodes
	nodeList, err := c.nodeLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Could not retrieve Nodes : %s", err)
		return total, err
	}

	workerNodes := utils.FilterNodesWithPredicate(nodeList, utils.FilterWorkerNode())

	// Only keep CPU and Memory
	total = &v1Core.ResourceList{
		v1Core.ResourceCPU:    *resource.NewMilliQuantity(utils.NodesCpuAllocatable(workerNodes), resource.DecimalSI),
		v1Core.ResourceMemory: *resource.NewQuantity(utils.NodesMemAllocatable(workerNodes), resource.BinarySI),
	}

	klog.Infof("Found %d Worker Nodes : %s Memory %s CPU", len(workerNodes), total.Memory().String(), total.Cpu().String())

	return total, err

}

// Gather the total of resource quota except the one on the namespace being evaluated
func (c *Controller) totalResourceQuota(claim *cagipv1.ResourceQuotaClaim) (sumResourceQuota *v1Core.ResourceList, err error) {
	sumResourceQuota = &v1Core.ResourceList{}
	// Retrieve ResourceQuotas
	if resourceQuotasAllNS, err := c.resourceQuotaLister.List(utils.DefaultLabelSelector()); err != nil {
		klog.Errorf("Could not retrieve ResourceQuotas : %s", err)
		return sumResourceQuota, err

	} else {
		// Exclude the resource quota of the claim namespace and sum the resources
		for _, resourceQuota := range resourceQuotasAllNS {
			if resourceQuota.Namespace != claim.Namespace {
				*sumResourceQuota = quota.Add(sumResourceQuota.DeepCopy(), resourceQuota.Spec.Hard.DeepCopy())
			}
		}

		klog.V(4).Infof(
			"Found %d ResourceQuotas : %s Memory %s CPU",
			len(*sumResourceQuota),
			sumResourceQuota.Memory().String(),
			sumResourceQuota.Cpu().String())

		return sumResourceQuota, err
	}
}

// Check is the managed quota is scaling down
func isDownscaleQuota(claim *cagipv1.ResourceQuotaClaim, managedQuota *v1Core.ResourceQuota) bool {
	return claim.Spec.Cpu().MilliValue() < managedQuota.Spec.Hard.Cpu().MilliValue() ||
		claim.Spec.Memory().Value() < managedQuota.Spec.Hard.Memory().Value()
}

// Check that it is possible to scale down the quota
// Return an empty message if possible else
// Return the reason
func canDownscaleQuota(claim *cagipv1.ResourceQuotaClaim, totalRequest *v1Core.ResourceList) string {
	if totalRequest.Memory().Value() > claim.Spec.Memory().Value() {
		return fmt.Sprintf(
			utils.MessagePendingMemoryDownscale,
			utils.BytesSize(float64(claim.Spec.Memory().Value())),
			utils.BytesSize(float64(totalRequest.Memory().Value())))

	}

	if totalRequest.Cpu().MilliValue() > claim.Spec.Cpu().MilliValue() {
		return fmt.Sprintf(
			utils.MessagePendingCpuDownscale,
			claim.Spec.Cpu().String(),
			totalRequest.Cpu().String())
	}

	return utils.EmptyMsg
}

// Delete a ResourceQuotaClaims
func (c *Controller) deleteResourceQuotaClaim(claim *cagipv1.ResourceQuotaClaim) (err error) {
	return c.resourcequotaclaimclientset.CagipV1().ResourceQuotaClaims(claim.Namespace).Delete(context.TODO(), claim.Name, metav1.DeleteOptions{})
}

// Create a ResourceQuota from a ResourceQuotaClaim resource.
func newResourceQuota(claim *cagipv1.ResourceQuotaClaim) *v1Core.ResourceQuota {
	labels := map[string]string{
		"creator": utils.ControllerName,
	}
	return &v1Core.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.ResourceQuotaName,
			Namespace: claim.Namespace,
			Labels:    labels,
		},
		Spec: v1Core.ResourceQuotaSpec{
			Hard: quota.Add(v1Core.ResourceList{}, claim.Spec),
		},
	}
}
