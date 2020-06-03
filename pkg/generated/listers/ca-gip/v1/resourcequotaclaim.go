// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/ca-gip/kotary/pkg/apis/ca-gip/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ResourceQuotaClaimLister helps list ResourceQuotaClaims.
type ResourceQuotaClaimLister interface {
	// List lists all ResourceQuotaClaims in the indexer.
	List(selector labels.Selector) (ret []*v1.ResourceQuotaClaim, err error)
	// ResourceQuotaClaims returns an object that can list and get ResourceQuotaClaims.
	ResourceQuotaClaims(namespace string) ResourceQuotaClaimNamespaceLister
	ResourceQuotaClaimListerExpansion
}

// resourceQuotaClaimLister implements the ResourceQuotaClaimLister interface.
type resourceQuotaClaimLister struct {
	indexer cache.Indexer
}

// NewResourceQuotaClaimLister returns a new ResourceQuotaClaimLister.
func NewResourceQuotaClaimLister(indexer cache.Indexer) ResourceQuotaClaimLister {
	return &resourceQuotaClaimLister{indexer: indexer}
}

// List lists all ResourceQuotaClaims in the indexer.
func (s *resourceQuotaClaimLister) List(selector labels.Selector) (ret []*v1.ResourceQuotaClaim, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ResourceQuotaClaim))
	})
	return ret, err
}

// ResourceQuotaClaims returns an object that can list and get ResourceQuotaClaims.
func (s *resourceQuotaClaimLister) ResourceQuotaClaims(namespace string) ResourceQuotaClaimNamespaceLister {
	return resourceQuotaClaimNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ResourceQuotaClaimNamespaceLister helps list and get ResourceQuotaClaims.
type ResourceQuotaClaimNamespaceLister interface {
	// List lists all ResourceQuotaClaims in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.ResourceQuotaClaim, err error)
	// Get retrieves the ResourceQuotaClaim from the indexer for a given namespace and name.
	Get(name string) (*v1.ResourceQuotaClaim, error)
	ResourceQuotaClaimNamespaceListerExpansion
}

// resourceQuotaClaimNamespaceLister implements the ResourceQuotaClaimNamespaceLister
// interface.
type resourceQuotaClaimNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ResourceQuotaClaims in the indexer for a given namespace.
func (s resourceQuotaClaimNamespaceLister) List(selector labels.Selector) (ret []*v1.ResourceQuotaClaim, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ResourceQuotaClaim))
	})
	return ret, err
}

// Get retrieves the ResourceQuotaClaim from the indexer for a given namespace and name.
func (s resourceQuotaClaimNamespaceLister) Get(name string) (*v1.ResourceQuotaClaim, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("resourcequotaclaim"), name)
	}
	return obj.(*v1.ResourceQuotaClaim), nil
}
