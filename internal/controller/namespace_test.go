package controller

import (
	cagipv1 "github.com/ca-gip/kotary/pkg/apis/ca-gip/v1"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewDefaultResourceQuotaClaim(t *testing.T) {

	TestCases := map[string]struct {
		namespace       string
		defaultResource v1.ResourceList
		want            cagipv1.ResourceQuotaClaim
	}{
		"default ns 500/500Mi": {
			namespace: "default",
			defaultResource: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			want: cagipv1.ResourceQuotaClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "default",
				},
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500"),
					v1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
		},
		"test ns 5k/1Gi": {
			namespace: "test",
			defaultResource: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5k"),
				v1.ResourceMemory: resource.MustParse("1Gi")},
			want: cagipv1.ResourceQuotaClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "test",
				},
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("5k"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		},
	}

	f := newFixture(t)
	c, _, _, _, _, _ := f.newController()

	for testName, testCase := range TestCases {
		t.Run(testName, func(t *testing.T) {
			c.settings.DefaultClaimSpec = testCase.defaultResource

			result := c.newDefaultResourceQuotaClaim(testCase.namespace)

			assert.DeepEqual(t, *result, testCase.want)
		})
	}
}

func TestHasTargetedLabel(t *testing.T) {

	TestCases := map[string]struct {
		namespace v1.Namespace
		expect    bool
	}{
		"nginx labels should not be targeted": {
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ingress-nginx",
					Labels: map[string]string{
						"app.kubernetes.io/name":    "ingress-nginx",
						"app.kubernetes.io/part-of": "ingress-nginx",
						"name":                      "ingress-nginx",
					},
				},
			},
			expect: false,
		},
		"empty labels should not be targeted": {
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "default",
					Labels: map[string]string{},
				},
			},
			expect: false,
		},
		"monitoring labels should not be targeted": {
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "monitoring",
					Labels: map[string]string{
						"name": "monitoring",
					},
				},
			},
			expect: false,
		},
		"quota labels should be target": {
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "team-1-development",
					Labels: map[string]string{
						"quota": "managed",
						"name":  "team-1-development",
						"type":  "customer",
					},
				},
			},
			expect: true,
		},
		"kubi labels should be target": {
			namespace: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "team-1-development",
					Labels: map[string]string{
						"creator": "kubi",
						"name":    "team-1-development",
						"type":    "customer",
					},
				},
			},
			expect: false,
		},
	}

	for testName, testCase := range TestCases {
		t.Run(testName, func(t *testing.T) {
			result := hasTargetedLabel(&testCase.namespace)
			assert.Equal(t, result, testCase.expect)
		})
	}

}

func TestHasResourceQuota(t *testing.T) {
	f := newFixture(t)

	resourceQuota := newTestResourceQuota(metav1.NamespaceDefault, "test", &v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse("1k"),
		v1.ResourceMemory: resource.MustParse("1Gi"),
	})
	f.resourceQuotaLister = append(f.resourceQuotaLister, resourceQuota)

	c := f.RunController()

	t.Run("default namespace should have resourcequota", func(t *testing.T) {
		result, err := c.hasResourceQuota(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: metav1.NamespaceDefault,
			},
		})

		assert.NilError(t, err)
		assert.Equal(t, result, true)
	})

	t.Run("other namespace should not have resourcequota", func(t *testing.T) {
		result, err := c.hasResourceQuota(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "other",
			},
		})

		assert.NilError(t, err)
		assert.Equal(t, result, false)
	})
}

func TestHasClaimWithoutStatus(t *testing.T) {

	var defaultNS = &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: metav1.NamespaceDefault,
		},
	}

	testCases := map[string]struct {
		claim  *cagipv1.ResourceQuotaClaim
		expect bool
	}{
		"claim without status should return true": {
			claim: &cagipv1.ResourceQuotaClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: metav1.NamespaceDefault,
				},
				Status: cagipv1.ResourceQuotaClaimStatus{},
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1k"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			expect: true,
		},
		"claim with rejected status should return false": {
			claim: &cagipv1.ResourceQuotaClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: metav1.NamespaceDefault,
				},
				Status: cagipv1.ResourceQuotaClaimStatus{Phase: cagipv1.PhaseRejected},
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1k"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			expect: false,
		},
		"claim with accepted status should return false": {
			claim: &cagipv1.ResourceQuotaClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: metav1.NamespaceDefault,
				},
				Status: cagipv1.ResourceQuotaClaimStatus{Phase: cagipv1.PhaseAccepted},
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1k"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			expect: false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			f := newFixture(t)
			f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, testCase.claim)
			c := f.RunController()

			result, err := c.hasUnevaluatedClaim(defaultNS)

			assert.NilError(t, err)
			assert.Equal(t, result, testCase.expect)
		})
	}
}
