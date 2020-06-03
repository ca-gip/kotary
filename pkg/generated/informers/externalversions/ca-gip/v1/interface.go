// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/ca-gip/kotary/pkg/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ResourceQuotaClaims returns a ResourceQuotaClaimInformer.
	ResourceQuotaClaims() ResourceQuotaClaimInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// ResourceQuotaClaims returns a ResourceQuotaClaimInformer.
func (v *version) ResourceQuotaClaims() ResourceQuotaClaimInformer {
	return &resourceQuotaClaimInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
