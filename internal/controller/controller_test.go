package controller

import (
	"fmt"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/quota/v1"
	"reflect"
	"testing"
	"time"

	"github.com/ca-gip/kotary/internal/utils"
	cagipv1 "github.com/ca-gip/kotary/pkg/apis/ca-gip/v1"
	"github.com/ca-gip/kotary/pkg/generated/clientset/versioned/fake"
	informers "github.com/ca-gip/kotary/pkg/generated/informers/externalversions"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type reactorErr struct {
	verb string
}

type fixture struct {
	t *testing.T

	// cliensset for each resource
	namespaceclientset          *k8sfake.Clientset
	nodesclientset              *k8sfake.Clientset
	resourcequotaclientset      *k8sfake.Clientset
	podsclientset               *k8sfake.Clientset
	resourcequotaclaimclientset *fake.Clientset
	// Objects to put in the store.
	namespaceLister          []*v1.Namespace
	resourceQuotaLister      []*v1.ResourceQuota
	nodeLister               []*v1.Node
	podLister                []*v1.Pod
	resourceQuotaClaimLister []*cagipv1.ResourceQuotaClaim
	// Actions expected to happen on the client.
	kubeactions []core.Action
	actions     []core.Action
	// Objects from here preloaded into NewSimpleFake.
	nsobjects   []runtime.Object
	nodeobjects []runtime.Object
	rqobjects   []runtime.Object
	podobjects  []runtime.Object
	rqcobjects  []runtime.Object
	// Add errors as reactors
	nserrors  []reactorErr
	noderrors []reactorErr
	rqerrors  []reactorErr
	poderrors []reactorErr
	rqcerrors []reactorErr
	// settings for the controller
	settings utils.Config
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.nsobjects = []runtime.Object{}
	f.nodeobjects = []runtime.Object{}
	f.rqobjects = []runtime.Object{}
	f.podobjects = []runtime.Object{}
	f.rqcobjects = []runtime.Object{}
	return f
}

func newTestResourceQuotaClaim(name string, spec *v1.ResourceList) *cagipv1.ResourceQuotaClaim {
	return &cagipv1.ResourceQuotaClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: quota.Add(v1.ResourceList{}, spec.DeepCopy()),
	}
}

func newTestResourceQuota(namespace string, name string, spec *v1.ResourceList) *v1.ResourceQuota {
	return &v1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"creator": utils.ControllerName,
			},
		},
		Spec: v1.ResourceQuotaSpec{Hard: *spec},
	}
}

func newTestNodes(number int, spec *v1.ResourceList) (nodes []*v1.Node) {
	for i := 0; i < number; i++ {
		nodes = append(nodes, &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("worker-%d", i),
			},
			Spec: v1.NodeSpec{
				Unschedulable: false,
			},
			Status: v1.NodeStatus{
				Capacity:    spec.DeepCopy(),
				Allocatable: spec.DeepCopy(),
				Conditions: []v1.NodeCondition{{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				}},
			},
		})
	}
	return
}

func newTestPods(number int, request *v1.ResourceList) (pods []*v1.Pod) {
	for i := 0; i < number; i++ {
		pods = append(pods, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Resources: v1.ResourceRequirements{
							Requests: request.DeepCopy(),
						},
					},
				},
			},
		})
	}
	return
}

func (f *fixture) newController() (*Controller, kubeinformers.SharedInformerFactory, kubeinformers.SharedInformerFactory, kubeinformers.SharedInformerFactory, kubeinformers.SharedInformerFactory, informers.SharedInformerFactory) {

	f.namespaceclientset = k8sfake.NewSimpleClientset(f.nsobjects...)
	f.nodesclientset = k8sfake.NewSimpleClientset(f.nodeobjects...)
	f.resourcequotaclientset = k8sfake.NewSimpleClientset(f.rqobjects...)
	f.podsclientset = k8sfake.NewSimpleClientset(f.podobjects...)
	f.resourcequotaclaimclientset = fake.NewSimpleClientset(f.rqcobjects...)

	f.settings = utils.Config{
		DefaultClaimSpec: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2"),
			v1.ResourceMemory: resource.MustParse("6Gi"),
		},
		RatioMaxAllocationMemory: 0.33,
		RatioMaxAllocationCPU:    0.33,
		RatioOverCommitMemory:    1,
		RatioOverCommitCPU:       1,
	}

	nsI := kubeinformers.NewSharedInformerFactory(f.namespaceclientset, noResyncPeriodFunc())
	nodeI := kubeinformers.NewSharedInformerFactory(f.namespaceclientset, noResyncPeriodFunc())
	rqI := kubeinformers.NewSharedInformerFactory(f.namespaceclientset, noResyncPeriodFunc())
	poI := kubeinformers.NewSharedInformerFactory(f.podsclientset, noResyncPeriodFunc())
	rqcI := informers.NewSharedInformerFactory(f.resourcequotaclaimclientset, noResyncPeriodFunc())

	c := NewController(
		f.settings,
		f.namespaceclientset, f.resourcequotaclientset, f.nodesclientset, f.podsclientset, f.resourcequotaclaimclientset,
		nsI.Core().V1().Namespaces(),
		rqI.Core().V1().ResourceQuotas(),
		nodeI.Core().V1().Nodes(),
		poI.Core().V1().Pods(),
		rqcI.Cagip().V1().ResourceQuotaClaims())

	c.namespacesSynced = alwaysReady
	c.resourceQuotaSynced = alwaysReady
	c.nodesSynced = alwaysReady
	c.podsSynced = alwaysReady
	c.resourceQuotaClaimSynced = alwaysReady

	c.recorder = &record.FakeRecorder{}

	for _, ns := range f.namespaceLister {
		_ = nsI.Core().V1().Namespaces().Informer().GetIndexer().Add(ns)
	}

	for _, rq := range f.resourceQuotaLister {
		_ = rqI.Core().V1().ResourceQuotas().Informer().GetIndexer().Add(rq)
	}

	for _, node := range f.nodeLister {
		_ = nodeI.Core().V1().Nodes().Informer().GetIndexer().Add(node)
	}

	for _, pod := range f.podLister {
		_ = poI.Core().V1().Pods().Informer().GetIndexer().Add(pod)
	}

	for _, rqc := range f.resourceQuotaClaimLister {
		_ = rqcI.Cagip().V1().ResourceQuotaClaims().Informer().GetIndexer().Add(rqc)
	}

	for _, nserror := range f.nserrors {
		f.namespaceclientset.PrependReactor(nserror.verb, "namespaces", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("fake error")
		})
	}

	for _, noderror := range f.noderrors {
		f.nodesclientset.PrependReactor(noderror.verb, "nodes", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("fake error")
		})
	}

	for _, rqerror := range f.rqerrors {
		f.resourcequotaclientset.PrependReactor(rqerror.verb, "resourcequotas", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("fake error")
		})
	}

	for _, poderror := range f.poderrors {
		f.podsclientset.PrependReactor(poderror.verb, "pods", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("fake error")
		})
	}

	for _, rqcerror := range f.rqcerrors {
		f.resourcequotaclaimclientset.PrependReactor(rqcerror.verb, "resourcequotaclaims", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("fake error")
		})
	}

	return c, nsI, nodeI, rqI, poI, rqcI
}

func (f *fixture) runClaim(name string) {
	f.runClaimControllerWithAction(name, true, false)
}

func (f *fixture) runClaimExpectError(name string) {
	f.runClaimControllerWithAction(name, true, true)
}

func (f *fixture) RunController() *Controller {
	c, nsI, nodeI, rqI, poI, rqcI := f.newController()

	stopCh := make(chan struct{})
	defer close(stopCh)
	nsI.Start(stopCh)
	nodeI.Start(stopCh)
	rqI.Start(stopCh)
	poI.Start(stopCh)
	rqcI.Start(stopCh)

	return c
}

func (f *fixture) runClaimControllerWithAction(rqcName string, startInformers bool, expectError bool) {
	c, nsI, nodeI, rqI, poI, rqcI := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		nsI.Start(stopCh)
		nodeI.Start(stopCh)
		rqI.Start(stopCh)
		poI.Start(stopCh)
		rqcI.Start(stopCh)
	}

	err := c.syncHandlerClaim(rqcName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing rqc: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing rqc, got nil")
	}

	actions := filterInformerActions(f.resourcequotaclaimclientset.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.resourcequotaclientset.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

func (f *fixture) runNS(name string) {
	f.runNSControllerWithAction(name, true, false)
}

func (f *fixture) runNSExpectError(name string) {
	f.runNSControllerWithAction(name, true, true)
}

func (f *fixture) runNSControllerWithAction(nsName string, startInformers bool, expectError bool) {
	c, nsI, nodeI, rqI, poI, rqcI := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		nsI.Start(stopCh)
		nodeI.Start(stopCh)
		rqI.Start(stopCh)
		poI.Start(stopCh)
		rqcI.Start(stopCh)
	}

	err := c.syncHandlerNS(nsName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing rqc: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing rqc, got nil")
	}

	actions := filterInformerActions(f.resourcequotaclaimclientset.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.resourcequotaclientset.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	case core.DeleteActionImpl:
		e, _ := expected.(core.DeleteActionImpl)
		if a.GetName() != e.GetName() || a.GetNamespace() != e.GetNamespace() {
			t.Errorf("Action %s %s is wrong \nExpected %s/%s got %s/%s ",
				a.GetVerb(), a.GetResource().Resource, e.GetName(), e.GetNamespace(), a.GetName(), a.GetNamespace())
		}

	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "resourcequotaclaims") ||
				action.Matches("watch", "resourcequotaclaims") ||
				action.Matches("list", "resourcequotas") ||
				action.Matches("watch", "resourcequotas")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func (f *fixture) expectCreateResourceQuotaAction(quota *v1.ResourceQuota) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "resourcequotas"}, quota.Namespace, quota))
}

func (f *fixture) expectUpdateResourceQuotaAction(quota *v1.ResourceQuota) {
	f.kubeactions = append(f.kubeactions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "resourcequotas"}, quota.Namespace, quota))
}

func (f *fixture) expectCreateResourceQuotaClaimAction(claim *cagipv1.ResourceQuotaClaim) {
	f.actions = append(f.actions, core.NewCreateAction(schema.GroupVersionResource{Resource: "resourcequotaclaims"}, claim.Namespace, claim))
}

func (f *fixture) expectDeleteResourceQuotaClaimAction(claim *cagipv1.ResourceQuotaClaim) {
	f.actions = append(f.actions, core.NewDeleteAction(schema.GroupVersionResource{Resource: "resourcequotaclaims"}, claim.Namespace, claim.Name))
}

func (f *fixture) expectUpdateStatusResourceQuotaClaimAction(rqc *cagipv1.ResourceQuotaClaim) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "resourcequotaclaims"}, rqc.Namespace, rqc)
	action.Subresource = "status"
	f.actions = append(f.actions, action)
}

func (f *fixture) expectClaimStatus(t *testing.T, claim *cagipv1.ResourceQuotaClaim, details string) {
	updatedClaim, err := f.resourcequotaclaimclientset.CagipV1().ResourceQuotaClaims(claim.Namespace).Get(claim.Namespace, metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, updatedClaim.Status.Details, details)
}

func getClaimKey(rqc *cagipv1.ResourceQuotaClaim, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(rqc)
	if err != nil {
		t.Errorf("Unexpected error getting key for rqc %v: %v", rqc.Name, err)
		return ""
	}
	return key
}

func getNSKey(ns *v1.Namespace, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(ns)
	if err != nil {
		t.Errorf("Unexpected error getting key for ns %v: %v", ns.Name, err)
		return ""
	}
	return key
}

func TestClaimCreateNewQuota(t *testing.T) {

	t.Run("1 Node 8Gi 1CPU - Claim 2Gi 300m", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectCreateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("3 Node 8Gi 1CPU - Claim 7Gi 900m", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(3, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("900m"),
			v1.ResourceMemory: resource.MustParse("7Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectCreateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("9 Node 8Gi 1CPU - Claim 20Gi 2.5CPU", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(9, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2.5"),
			v1.ResourceMemory: resource.MustParse("20Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectCreateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("error while creating quota should requeue", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Inject error clientset
		f.rqerrors = append(f.rqerrors, reactorErr{verb: "create"})

		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectCreateResourceQuotaAction(expResourceQuota)

		f.runClaimExpectError(getClaimKey(claim, t))
	})

	t.Run("error while deleting claim should requeue", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Inject error clientset
		f.rqcerrors = append(f.rqcerrors, reactorErr{verb: "delete"})

		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectCreateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaimExpectError(getClaimKey(claim, t))
	})

}

func TestClaimUpdateQuota(t *testing.T) {

	t.Run("1 Node 8Gi 1CPU - Claim 2Gi 300m", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Existing Quota
		managedQuota := &v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ResourceQuotaName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		}
		f.resourceQuotaLister = append(f.resourceQuotaLister, managedQuota)
		f.rqobjects = append(f.rqcobjects, managedQuota)
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)

		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectUpdateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("3 Node 8Gi 1CPU - Claim 7Gi 900m", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(3, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Existing Quota
		managedQuota := &v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ResourceQuotaName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		}
		f.resourceQuotaLister = append(f.resourceQuotaLister, managedQuota)
		f.rqobjects = append(f.rqcobjects, managedQuota)
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("900m"),
			v1.ResourceMemory: resource.MustParse("7Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectUpdateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("9 Node 8Gi 1CPU - Claim 20Gi 2.5CPU", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(9, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Existing Quota
		managedQuota := &v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ResourceQuotaName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		}
		f.resourceQuotaLister = append(f.resourceQuotaLister, managedQuota)
		f.rqobjects = append(f.rqcobjects, managedQuota)
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2.5"),
			v1.ResourceMemory: resource.MustParse("20Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectUpdateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("error while updating quota should requeue", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Existing Quota
		managedQuota := &v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ResourceQuotaName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		}
		f.resourceQuotaLister = append(f.resourceQuotaLister, managedQuota)
		f.rqobjects = append(f.rqcobjects, managedQuota)
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Inject error clientset
		f.rqerrors = append(f.rqerrors, reactorErr{verb: "update"})

		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectUpdateResourceQuotaAction(expResourceQuota)

		f.runClaimExpectError(getClaimKey(claim, t))
	})

	t.Run("error while deleting claim should requeue", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Existing Quota
		managedQuota := &v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ResourceQuotaName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		}
		f.resourceQuotaLister = append(f.resourceQuotaLister, managedQuota)
		f.rqobjects = append(f.rqcobjects, managedQuota)
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Inject error clientset
		f.rqcerrors = append(f.rqcerrors, reactorErr{verb: "delete"})

		// Expected Actions
		expResourceQuota := newResourceQuota(claim)
		f.expectUpdateResourceQuotaAction(expResourceQuota)
		f.expectDeleteResourceQuotaClaimAction(claim)

		f.runClaimExpectError(getClaimKey(claim, t))
	})

}

func TestClaimPending(t *testing.T) {
	t.Run("1 Node 16Gi 4CPU - Claim 5Gi 600m - Request 6Gi 750m - Should be Pending Memory", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("4"),
			v1.ResourceMemory: resource.MustParse("16Gi"),
		})
		// Existing Quota
		managedQuota := &v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ResourceQuotaName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("800m"),
					v1.ResourceMemory: resource.MustParse("6Gi"),
				},
			},
		}
		f.resourceQuotaLister = append(f.resourceQuotaLister, managedQuota)
		f.rqobjects = append(f.rqcobjects, managedQuota)
		// Scheduled Pods
		pods := newTestPods(3, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("250m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.podLister = pods
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("600m"),
			v1.ResourceMemory: resource.MustParse("5Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Status
		claim.Status.Phase = cagipv1.PhasePending
		claim.Status.Details = "Awaiting lower Memory consumption claiming 5Gi but current total of request is 6Gi"
		f.expectUpdateStatusResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("1 Node 16Gi 4CPU - Claim 6Gi 600m - Request 6Gi 750m - Should be Pending CPU", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("4"),
			v1.ResourceMemory: resource.MustParse("16Gi"),
		})
		// Existing Quota
		managedQuota := &v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ResourceQuotaName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceQuotaSpec{
				Hard: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("800m"),
					v1.ResourceMemory: resource.MustParse("6Gi"),
				},
			},
		}
		f.resourceQuotaLister = append(f.resourceQuotaLister, managedQuota)
		f.rqobjects = append(f.rqcobjects, managedQuota)
		// Scheduled Pods
		pods := newTestPods(3, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("250m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.podLister = pods
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("600m"),
			v1.ResourceMemory: resource.MustParse("6Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Status
		claim.Status.Phase = cagipv1.PhasePending
		claim.Status.Details = "Awaiting lower CPU consumption claiming 600m but current total of CPU request is 750m"
		f.expectUpdateStatusResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})
}

func TestClaimRejected(t *testing.T) {

	t.Run("1 Node 8Gi 1CPU - Claim 10Gi 300m - Max Allocation Memory", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("10Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)

		// Expected Status
		claim.Status.Phase = cagipv1.PhaseRejected
		claim.Status.Details = "Exceeded Memory allocation limit claiming 10Gi but limited to 2.64Gi"
		f.expectUpdateStatusResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("1 Node 8Gi 1CPU - Claim 2Gi 500m - Max Allocation CPU", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("500m"),
			v1.ResourceMemory: resource.MustParse("2Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)

		// Expected Status
		claim.Status.Phase = cagipv1.PhaseRejected
		claim.Status.Details = "Exceeded CPU allocation limit claiming 500m but limited to 330m"
		f.expectUpdateStatusResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("1 Node 8Gi 1CPU - Claim 2.5Gi 100m - Not Enough Memory", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Quota (already reserved resource)
		f.resourceQuotaLister = append(f.resourceQuotaLister,
			newTestResourceQuota("otherns", "managed", &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("800m"),
				v1.ResourceMemory: resource.MustParse("6Gi"),
			}))
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("100m"),
			v1.ResourceMemory: resource.MustParse("2.50Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Status
		claim.Status.Phase = cagipv1.PhaseRejected
		claim.Status.Details = "Not enough Memory claiming 2560Mi but 2Gi currently available"
		f.expectUpdateStatusResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("1 Node 8Gi 1CPU - Claim 1.8Gi 300m - Not Enough CPU", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Quota (already reserved resource)
		f.resourceQuotaLister = append(f.resourceQuotaLister,
			newTestResourceQuota("otherns", "managed", &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("800m"),
				v1.ResourceMemory: resource.MustParse("6Gi"),
			}))
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("1.8Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		// Expected Status
		claim.Status.Phase = cagipv1.PhaseRejected
		claim.Status.Details = "Not enough CPU claiming 300m but 200m currently available"
		f.expectUpdateStatusResourceQuotaClaimAction(claim)

		f.runClaim(getClaimKey(claim, t))
	})

	t.Run("error while updating claim status should requeue", func(t *testing.T) {
		f := newFixture(t)
		// Nodes
		f.nodeLister = newTestNodes(1, &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("1"),
			v1.ResourceMemory: resource.MustParse("8Gi"),
		})
		// Test against claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("300m"),
			v1.ResourceMemory: resource.MustParse("10Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)

		// Inject error clientset
		f.rqcerrors = append(f.rqcerrors, reactorErr{verb: "update"})

		// Expected Status
		claim.Status.Phase = cagipv1.PhaseRejected
		claim.Status.Details = "Exceeded Memory allocation limit claiming 10Gi but limited to 2.64Gi"
		f.expectUpdateStatusResourceQuotaClaimAction(claim)

		f.runClaimExpectError(getClaimKey(claim, t))
	})

}

func TestAddDefaultClaimToNS(t *testing.T) {

	t.Run("blank namespace with target annotation should generate default claim", func(t *testing.T) {
		f := newFixture(t)
		// Test against NS
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
				Labels: map[string]string{
					"quota": "managed",
				},
			},
		}
		f.namespaceLister = append(f.namespaceLister, ns)
		f.nsobjects = append(f.nsobjects, ns)
		// Expect Claim
		expectedClaim := newTestResourceQuotaClaim("default", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2"),
			v1.ResourceMemory: resource.MustParse("6Gi"),
		})
		f.expectCreateResourceQuotaClaimAction(expectedClaim)

		f.runNS(getNSKey(ns, t))
	})

	t.Run("blank namespace without target annotation should not generate default claim", func(t *testing.T) {
		f := newFixture(t)
		// Test against NS
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
				Labels: map[string]string{
					"quota": "unmanaged",
				},
			},
		}
		f.namespaceLister = append(f.namespaceLister, ns)
		f.nsobjects = append(f.nsobjects, ns)

		f.runNS(getNSKey(ns, t))
	})

	t.Run("namespace with target annotation containing a quota should not generate default claim", func(t *testing.T) {
		f := newFixture(t)
		// Test against NS
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
				Labels: map[string]string{
					"quota": "managed",
				},
			},
		}
		f.namespaceLister = append(f.namespaceLister, ns)
		f.nsobjects = append(f.nsobjects, ns)
		// Quota
		existingQuota := newTestResourceQuota(metav1.NamespaceDefault, "test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2"),
			v1.ResourceMemory: resource.MustParse("6Gi"),
		})
		f.resourceQuotaLister = append(f.resourceQuotaLister, existingQuota)

		f.runNS(getNSKey(ns, t))
	})

	t.Run("namespace with target annotation containing a claim should not generate default claim", func(t *testing.T) {
		f := newFixture(t)
		// Test against NS
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
				Labels: map[string]string{
					"quota": "managed",
				},
			},
		}
		f.namespaceLister = append(f.namespaceLister, ns)
		f.nsobjects = append(f.nsobjects, ns)
		// Claim
		claim := newTestResourceQuotaClaim("test", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2"),
			v1.ResourceMemory: resource.MustParse("6Gi"),
		})
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)

		f.runNS(getNSKey(ns, t))
	})

	t.Run("namespace with target annotation containing a Rejected claim should generate default claim", func(t *testing.T) {
		f := newFixture(t)
		// Test against NS
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
				Labels: map[string]string{
					"quota": "managed",
				},
			},
		}
		f.namespaceLister = append(f.namespaceLister, ns)
		f.nsobjects = append(f.nsobjects, ns)
		// Rejected Claim
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, &cagipv1.ResourceQuotaClaim{
			Status: cagipv1.ResourceQuotaClaimStatus{
				Phase: cagipv1.PhaseRejected,
			},
		})
		// Expected Claim
		expectedClaim := newTestResourceQuotaClaim("default", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2"),
			v1.ResourceMemory: resource.MustParse("6Gi"),
		})
		f.expectCreateResourceQuotaClaimAction(expectedClaim)

		f.runNS(getNSKey(ns, t))
	})

	t.Run("error while creating claim should requeue", func(t *testing.T) {
		f := newFixture(t)
		// Test against NS
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
				Labels: map[string]string{
					"quota": "managed",
				},
			},
		}
		f.namespaceLister = append(f.namespaceLister, ns)
		f.nsobjects = append(f.nsobjects, ns)
		// Expect
		expectedClaim := newTestResourceQuotaClaim("default", &v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("2"),
			v1.ResourceMemory: resource.MustParse("6Gi"),
		})
		f.expectCreateResourceQuotaClaimAction(expectedClaim)
		// Inject client error
		f.rqcerrors = append(f.rqcerrors, reactorErr{verb: "create"})

		f.runNSExpectError(getNSKey(ns, t))
	})

	t.Run("non existing ns should be ignored ", func(t *testing.T) {
		f := newFixture(t)
		// Non existing NS
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
				Labels: map[string]string{
					"quota": "managed",
				},
			},
		}
		// Don't expect anything
		f.runNS(getNSKey(ns, t))
	})

}
