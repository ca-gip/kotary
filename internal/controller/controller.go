package controller

import (
	"fmt"
	"time"

	"github.com/ca-gip/kotary/internal/utils"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	cagipv1 "github.com/ca-gip/kotary/pkg/apis/cagip/v1"
	clientset "github.com/ca-gip/kotary/pkg/generated/clientset/versioned"
	resourcequotaclaimcheme "github.com/ca-gip/kotary/pkg/generated/clientset/versioned/scheme"
	informers "github.com/ca-gip/kotary/pkg/generated/informers/externalversions/cagip/v1"
	listers "github.com/ca-gip/kotary/pkg/generated/listers/cagip/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// Controller is the controller implementation for ResourceQuotaClaims resources
type Controller struct {
	// clientset
	namespaceclientset          kubernetes.Interface
	nodesclientset              kubernetes.Interface
	resourcequotaclientset      kubernetes.Interface
	podsclientset               kubernetes.Interface
	resourcequotaclaimclientset clientset.Interface

	// ns
	namespaceLister  corelisters.NamespaceLister
	namespacesSynced cache.InformerSynced

	// resourcequota
	resourceQuotaLister corelisters.ResourceQuotaLister
	resourceQuotaSynced cache.InformerSynced
	// nodes
	nodeLister  corelisters.NodeLister
	nodesSynced cache.InformerSynced

	// pods
	podsLister corelisters.PodLister
	podsSynced cache.InformerSynced

	/// resourcequotaclaim
	resourceQuotaClaimLister listers.ResourceQuotaClaimLister
	resourceQuotaClaimSynced cache.InformerSynced

	// resourceQuotaClaimWorkQueue and namespaceWorkQueue are a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	resourceQuotaClaimWorkQueue workqueue.RateLimitingInterface
	namespaceWorkQueue          workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// Settings
	settings utils.Config
}

// NewController returns a new resourcequotaclaim controller
func NewController(
	settings utils.Config,
	namespaceclientset kubernetes.Interface,
	resourcequotaclientset kubernetes.Interface,
	nodesclientset kubernetes.Interface,
	podsclientset kubernetes.Interface,
	resourcequotaclaimclientset clientset.Interface,
	namespaceInformer coreinformers.NamespaceInformer,
	resourceQuotaInformer coreinformers.ResourceQuotaInformer,
	nodesInformer coreinformers.NodeInformer,
	podsInformer coreinformers.PodInformer,
	resourceQuotaClaimInformer informers.ResourceQuotaClaimInformer) *Controller {

	// Create event broadcaster
	// Add resourcequotaclaim-controller types to the default Kubernetes Scheme so Events can be
	// logged for resourcequotaclaim-controller types.
	utilruntime.Must(resourcequotaclaimcheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: namespaceclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: utils.ControllerName})

	controller := &Controller{
		namespaceclientset:          namespaceclientset,
		resourcequotaclientset:      resourcequotaclientset,
		nodesclientset:              nodesclientset,
		podsclientset:               podsclientset,
		resourcequotaclaimclientset: resourcequotaclaimclientset,
		namespaceLister:             namespaceInformer.Lister(),
		namespacesSynced:            namespaceInformer.Informer().HasSynced,
		resourceQuotaLister:         resourceQuotaInformer.Lister(),
		resourceQuotaSynced:         resourceQuotaInformer.Informer().HasSynced,
		nodeLister:                  nodesInformer.Lister(),
		nodesSynced:                 nodesInformer.Informer().HasSynced,
		podsLister:                  podsInformer.Lister(),
		podsSynced:                  podsInformer.Informer().HasSynced,
		resourceQuotaClaimLister:    resourceQuotaClaimInformer.Lister(),
		resourceQuotaClaimSynced:    resourceQuotaClaimInformer.Informer().HasSynced,
		resourceQuotaClaimWorkQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ResourceQuotaClaims"),
		namespaceWorkQueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Namespaces"),
		recorder:                    recorder,
		settings:                    settings,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for claim creation and
	// update (in the case it was reject otherwise the claim is removed)
	resourceQuotaClaimInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueResourceQuotaClaim,
		UpdateFunc: func(old, new interface{}) {
			newQuotaClaim := new.(*cagipv1.ResourceQuotaClaim)
			oldQuotaClaim := old.(*cagipv1.ResourceQuotaClaim)
			if newQuotaClaim.ResourceVersion == oldQuotaClaim.ResourceVersion {
				return
			}
			controller.enqueueResourceQuotaClaim(new)
		},
	})

	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNamespace,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueNamespace(new)
		},
	})

	return controller
}

// Check if Shared informer have synced, use for liveness probe
func (c *Controller) SharedInformersState() error {

	if synced := c.namespacesSynced(); !synced {
		return fmt.Errorf(fmt.Sprintf(utils.SharedInformerNotSync, "Namespace"))
	}

	if synced := c.resourceQuotaSynced(); !synced {
		return fmt.Errorf(fmt.Sprintf(utils.SharedInformerNotSync, "ResourceQuota"))
	}

	if synced := c.podsSynced(); !synced {
		return fmt.Errorf(fmt.Sprintf(utils.SharedInformerNotSync, "Pods"))
	}

	if synced := c.resourceQuotaClaimSynced(); !synced {
		return fmt.Errorf(fmt.Sprintf(utils.SharedInformerNotSync, "ResourceQuotaClaim"))
	}

	return nil

}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the resourceQuotaClaimWorkQueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.resourceQuotaClaimWorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting ResourceQuotaClaim controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.namespacesSynced, c.resourceQuotaSynced, c.nodesSynced, c.podsSynced, c.resourceQuotaClaimSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")

	// Launch at least two workers one for the claim the other for namespace
	for i := 0; i < threadiness/2; i++ {
		go wait.Until(c.runWorkerClaim, time.Second, stopCh)
		go wait.Until(c.runWorkerNS, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorkerClaim and runWorkerNS are a long-running function that will continually call the
// processNext...() function in order to read and process a message on the WorkQueue
func (c *Controller) runWorkerClaim() {
	for c.processNextWorkClaim() {
	}
}

func (c *Controller) runWorkerNS() {
	for c.processNextWorkNS() {
	}
}

// processNextWorkClaim will read a single work item off the resourceQuotaClaimWorkQueue and
// attempt to process it, by calling the syncHandlerClaim.
func (c *Controller) processNextWorkClaim() bool {
	obj, shutdown := c.resourceQuotaClaimWorkQueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.resourceQuotaClaimWorkQueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the resourceQuotaClaimWorkQueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the resourceQuotaClaimWorkQueue and attempted again after a back-off
		// period.
		defer c.resourceQuotaClaimWorkQueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the resourceQuotaClaimWorkQueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// resourceQuotaClaimWorkQueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// resourceQuotaClaimWorkQueue.
		if key, ok = obj.(string); !ok {
			// As the item in the resourceQuotaClaimWorkQueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.resourceQuotaClaimWorkQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in resourceQuotaClaimWorkQueue but got %#v", obj))
			return nil
		}
		// Run the syncHandlerClaim, passing it the namespace/name string of the
		// ResourceQuotaClaims resource to be synced.
		if err := c.syncHandlerClaim(key); err != nil {
			// Put the item back on the resourceQuotaClaimWorkQueue to handle any transient errors.
			c.resourceQuotaClaimWorkQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.resourceQuotaClaimWorkQueue.Forget(obj)
		klog.Infof("Successfully synced claim '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) processNextWorkNS() bool {
	obj, shutdown := c.namespaceWorkQueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.namespaceWorkQueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.namespaceWorkQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in Namespace but got %#v", obj))
			return nil
		}

		if err := c.syncHandlerNS(key); err != nil {
			c.namespaceWorkQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.namespaceWorkQueue.Forget(obj)
		klog.Infof("Successfully synced ns '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// enqueueResourceQuotaClaim takes a resourceQuotaClaim resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than resourceQuotaClaim.
func (c *Controller) enqueueResourceQuotaClaim(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.resourceQuotaClaimWorkQueue.Add(key)
}

func (c *Controller) enqueueNamespace(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.namespaceWorkQueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the ResourceQuotaClaims resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that ResourceQuotaClaims resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a ResourceQuotaClaims, we should not do anything more
		// with it.
		if ownerRef.Kind != "ResourceQuotaClaims" {
			return
		}

		claim, err := c.resourceQuotaClaimLister.ResourceQuotaClaims(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of ResourceQuotaClaims '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueResourceQuotaClaim(claim)
		return
	}
}
