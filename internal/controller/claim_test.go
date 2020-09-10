package controller

import (
	"github.com/ca-gip/kotary/internal/utils"
	cagipv1 "github.com/ca-gip/kotary/pkg/apis/ca-gip/v1"
	"github.com/storageos/go-api/types"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestApplyOverProvisioning(t *testing.T) {

	TestResourceList := map[string]struct {
		given      *v1.ResourceList
		overcommit float64
		expect     v1.ResourceList
	}{
		"100% over-commit": {
			given: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			overcommit: 1,
			expect: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
		},
		"150% over-commit": {
			given: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			overcommit: 1.5,
			expect: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("750m"),
				v1.ResourceMemory: resource.MustParse("750Mi")},
		},
		"50% under-commit": {
			given: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			overcommit: 0.5,
			expect: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("250m"),
				v1.ResourceMemory: resource.MustParse("250Mi")},
		},
		"200% over-commit": {
			given: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			overcommit: 2,
			expect: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1000Mi")},
		},
		"0% under-commit": {
			given: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("0"),
				v1.ResourceMemory: resource.MustParse("0")},
			overcommit: 0,
			expect: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("0"),
				v1.ResourceMemory: resource.MustParse("0")},
		},
	}

	for testName, testCase := range TestResourceList {
		t.Run(testName, func(t *testing.T) {
			f := newFixture(t)
			c, _, _, _, _, _ := f.newController()
			c.settings.RatioOverCommitMemory = testCase.overcommit
			c.settings.RatioOverCommitCPU = testCase.overcommit

			result := c.applyOverProvisioning(testCase.given)

			assert.Equal(t, testCase.expect.Cpu().MilliValue(), result.Cpu().MilliValue())
			assert.Equal(t, testCase.expect.Memory().MilliValue(), result.Memory().MilliValue())
		})
	}
}

func TestCheckAllocationLimit(t *testing.T) {

	TestCases := map[string]struct {
		availableResources  *v1.ResourceList
		claim               *cagipv1.ResourceQuotaClaim
		rationMaxAllocation float64
		expectPassed        bool
		expectMsg           string
	}{
		"1 allocation should pass": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500m"),
					v1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
			rationMaxAllocation: 1,
			expectMsg:           utils.EmptyMsg,
		},
		"0.99 allocation should fail because of memory": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500m"),
					v1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
			rationMaxAllocation: 0.99,
			expectMsg:           "Exceeded Memory allocation limit claiming 500Mi but limited to 495Mi",
		},
		"0.33 allocation should pass": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("151m"),
					v1.ResourceMemory: resource.MustParse("151Mi"),
				},
			},
			rationMaxAllocation: 0.33,
			expectMsg:           utils.EmptyMsg,
		},
		"0.33 allocation should fail because of memory": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("0"),
					v1.ResourceMemory: resource.MustParse("166Mi"),
				},
			},
			rationMaxAllocation: 0.33,
			expectMsg:           "Exceeded Memory allocation limit claiming 166Mi but limited to 165Mi",
		},
		"0.33 allocation should fail because of cpu": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("166m"),
					v1.ResourceMemory: resource.MustParse("0"),
				},
			},
			rationMaxAllocation: 0.33,
			expectMsg:           "Exceeded CPU allocation limit claiming 166m but limited to 165m",
		},
		"1.4 allocation should pass": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("700m"),
					v1.ResourceMemory: resource.MustParse("700Mi"),
				},
			},
			rationMaxAllocation: 1.4,
			expectMsg:           utils.EmptyMsg,
		},
		"1.4 allocation should fail because of memory": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("701m"),
					v1.ResourceMemory: resource.MustParse("701Mi"),
				},
			},
			rationMaxAllocation: 1.4,
			expectMsg:           "Exceeded Memory allocation limit claiming 701Mi but limited to 700Mi",
		},
		"1.4 allocation should fail because of cpu": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("701m"),
					v1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
			rationMaxAllocation: 1.4,
			expectMsg:           "Exceeded CPU allocation limit claiming 701m but limited to 700m",
		},
		"0 allocation should pass": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("0"),
					v1.ResourceMemory: resource.MustParse("0"),
				},
			},
			rationMaxAllocation: 0,
			expectMsg:           utils.EmptyMsg,
		},
		"0 allocation should fail because of memory": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("0"),
					v1.ResourceMemory: resource.MustParse("1"),
				},
			},
			rationMaxAllocation: 0,
			expectMsg:           "Exceeded Memory allocation limit claiming 1 but limited to 0",
		},
		"0 allocation should fail because of cpu": {
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("500Mi")},
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1m"),
					v1.ResourceMemory: resource.MustParse("0"),
				},
			},
			rationMaxAllocation: 0,
			expectMsg:           "Exceeded CPU allocation limit claiming 1m but limited to 0",
		},
	}

	for testName, testCase := range TestCases {
		t.Run(testName, func(t *testing.T) {
			f := newFixture(t)
			c, _, _, _, _, _ := f.newController()
			c.settings.RatioMaxAllocationMemory = testCase.rationMaxAllocation
			c.settings.RatioMaxAllocationCPU = testCase.rationMaxAllocation

			resultMsg := c.checkAllocationLimit(testCase.claim, testCase.availableResources)

			assert.Equal(t, resultMsg, testCase.expectMsg)
		})
	}
}

func TestCheckResourceFit(t *testing.T) {

	TestCases := map[string]struct {
		claim              *v1.ResourceList
		availableResources *v1.ResourceList
		reservedResources  *v1.ResourceList
		overcommit         float64
		expectPassed       bool
		expectMsg          string
	}{
		"1 overcommit should pass": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("0.99"),
				v1.ResourceMemory: resource.MustParse("0.97Gi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("4"),
				v1.ResourceMemory: resource.MustParse("9Gi"),
			},
			overcommit: 1,
			expectMsg:  utils.EmptyMsg,
		},
		"1 overcommit should fail because of memory": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("4"),
				v1.ResourceMemory: resource.MustParse("9728Mi"),
			},
			overcommit: 1,
			expectMsg:  "Not enough Memory claiming 1Gi but 512Mi currently available",
		},
		"1 overcommit should fail because of cpu": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("4.5"),
				v1.ResourceMemory: resource.MustParse("8Gi"),
			},
			overcommit: 1,
			expectMsg:  "Not enough CPU claiming 1 but 500m currently available",
		},
		"1.2 overcommit should pass": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("999m"),
				v1.ResourceMemory: resource.MustParse("999Mi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			overcommit: 1.2,
			expectMsg:  utils.EmptyMsg,
		},
		"1.2 overcommit should fail because of memory": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("11.5Gi"),
			},
			overcommit: 1.2,
			expectMsg:  "Not enough Memory claiming 1Gi but 512Mi currently available",
		},
		"1.2 overcommit should fail because of cpu": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5.5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			overcommit: 1.2,
			expectMsg:  "Not enough CPU claiming 1 but 500m currently available",
		},
		"0.8 overcommit should pass": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("0.99"),
				v1.ResourceMemory: resource.MustParse("999Mi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("3"),
				v1.ResourceMemory: resource.MustParse("7Gi"),
			},
			overcommit: 0.8,
			expectMsg:  utils.EmptyMsg,
		},
		"0.8 overcommit should fail because of memory": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("7.5Gi"),
			},
			overcommit: 0.8,
			expectMsg:  "Not enough Memory claiming 1Gi but 512Mi currently available",
		},
		"0.8 overcommit should fail because of cpu": {
			claim: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			availableResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("5"),
				v1.ResourceMemory: resource.MustParse("10Gi"),
			},
			reservedResources: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("3.5"),
				v1.ResourceMemory: resource.MustParse("6Gi"),
			},
			overcommit: 0.8,
			expectMsg:  "Not enough CPU claiming 1 but 500m currently available",
		},
	}

	for testName, testCase := range TestCases {
		t.Run(testName, func(t *testing.T) {
			f := newFixture(t)
			c, _, _, _, _, _ := f.newController()
			c.settings.RatioOverCommitMemory = testCase.overcommit
			c.settings.RatioOverCommitCPU = testCase.overcommit
			claim := newTestResourceQuotaClaim("test", testCase.claim)

			msg := c.checkResourceFit(claim, testCase.availableResources, testCase.reservedResources)

			assert.Equal(t, msg, testCase.expectMsg)

		})
	}
}

func TestNodesTotalCapacity(t *testing.T) {

	testCases := map[string]struct {
		nodes  []*v1.Node
		expect *v1.ResourceList
	}{
		"1 unready workers": {
			nodes: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node-role.kubernetes.io/master": "true",
						},
						Name: "master-01",
					},
					Spec: v1.NodeSpec{
						Unschedulable: true,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionFalse,
							},
						},
					},
				},
			},
			expect: &v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("0"),
				v1.ResourceCPU:    resource.MustParse("0"),
			},
		},
		"1 dangling worker (no conditions)": {
			nodes: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node-role.kubernetes.io/master": "true",
						},
						Name: "master-01",
					},
					Spec: v1.NodeSpec{
						Unschedulable: true,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
					},
				},
			},
			expect: &v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("0"),
				v1.ResourceCPU:    resource.MustParse("0"),
			},
		},
		"1 master 2 workers": {
			nodes: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node-role.kubernetes.io/master": "true",
						},
						Name: "master-01",
					},
					Spec: v1.NodeSpec{
						Unschedulable: true,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							" beta.kubernetes.io/arch": "amd64",
						},
						Name: "worker-01",
					},
					Spec: v1.NodeSpec{
						Unschedulable: false,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							" beta.kubernetes.io/arch": "amd64",
						},
						Name: "worker-02",
					},
					Spec: v1.NodeSpec{
						Unschedulable: false,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
			},
			expect: &v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("64Gi"),
				v1.ResourceCPU:    resource.MustParse("16"),
			},
		},
		"2 masters": {
			nodes: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node-role.kubernetes.io/master": "",
						},
						Name: "master-01",
					},
					Spec: v1.NodeSpec{
						Unschedulable: true,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node-role.kubernetes.io/master": "",
						},
						Name: "master-02",
					},
					Spec: v1.NodeSpec{
						Unschedulable: true,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
			},
			expect: &v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("0"),
				v1.ResourceCPU:    resource.MustParse("0"),
			},
		},
		"3 workers": {
			nodes: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							" beta.kubernetes.io/arch": "amd64",
						},
						Name: "worker-01",
					},
					Spec: v1.NodeSpec{
						Unschedulable: false,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							" beta.kubernetes.io/arch": "amd64",
						},
						Name: "worker-02",
					},
					Spec: v1.NodeSpec{
						Unschedulable: false,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							" beta.kubernetes.io/arch": "amd64",
						},
						Name: "worker-03",
					},
					Spec: v1.NodeSpec{
						Unschedulable: false,
					},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("32Gi"),
							v1.ResourceCPU:    resource.MustParse("8"),
						},
						Phase: "",
						Conditions: []v1.NodeCondition{
							{
								Type:   v1.NodeReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
			},
			expect: &v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("96Gi"),
				v1.ResourceCPU:    resource.MustParse("24"),
			},
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			f := newFixture(t)
			for _, node := range testCase.nodes {
				f.nodeLister = append(f.nodeLister, node)
			}
			c, _, _, _, _, _ := f.newController()

			result, _ := c.nodesTotalCapacity()

			assert.Equal(t, result.Cpu().Value(), testCase.expect.Cpu().Value())
			assert.Equal(t, result.Memory().Value(), testCase.expect.Memory().Value())
		})
	}
}

func TestTotalResourceQuota(t *testing.T) {

	testCases := map[string]struct {
		claim  *cagipv1.ResourceQuotaClaim
		quotas []*v1.ResourceQuota
		expect *v1.ResourceList
	}{
		"3 quotas": {
			claim: &cagipv1.ResourceQuotaClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: types.DefaultNamespace,
				},
			},
			quotas: []*v1.ResourceQuota{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
					},
					Spec: v1.ResourceQuotaSpec{
						Hard: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("8Gi"),
							v1.ResourceCPU:    resource.MustParse("3k"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-01",
					},
					Spec: v1.ResourceQuotaSpec{
						Hard: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("8Gi"),
							v1.ResourceCPU:    resource.MustParse("3k"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-02",
					},
					Spec: v1.ResourceQuotaSpec{
						Hard: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("8Gi"),
							v1.ResourceCPU:    resource.MustParse("3k"),
						},
					},
				},
			},
			expect: &v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("16Gi"),
				v1.ResourceCPU:    resource.MustParse("6k"),
			},
		},
		"1 quota": {
			claim: &cagipv1.ResourceQuotaClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: types.DefaultNamespace,
				},
			},
			quotas: []*v1.ResourceQuota{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
					},
					Spec: v1.ResourceQuotaSpec{
						Hard: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("8Gi"),
							v1.ResourceCPU:    resource.MustParse("3k"),
						},
					},
				},
			},
			expect: &v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("8Gi"),
				v1.ResourceCPU:    resource.MustParse("3k"),
			},
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			f := newFixture(t)
			for _, quota := range testCase.quotas {
				f.resourceQuotaLister = append(f.resourceQuotaLister, quota)
			}
			c, _, _, _, _, _ := f.newController()

			result, err := c.totalResourceQuota(testCase.claim)

			assert.NilError(t, err)
			assert.Equal(t, result.Cpu().Value(), testCase.expect.Cpu().Value())
			assert.Equal(t, result.Memory().Value(), testCase.expect.Memory().Value())
		})
	}
}

func TestDeleteResourceQuotaClaim(t *testing.T) {

	t.Run("should delete correctly", func(t *testing.T) {
		claim := &cagipv1.ResourceQuotaClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Mi"),
			},
		}
		f := newFixture(t)
		f.resourceQuotaClaimLister = append(f.resourceQuotaClaimLister, claim)
		f.rqcobjects = append(f.rqcobjects, claim)
		c := f.RunController()

		resultErr := c.deleteResourceQuotaClaim(claim)

		assert.NilError(t, resultErr)

	})

	t.Run("should fail to delete", func(t *testing.T) {
		claim := &cagipv1.ResourceQuotaClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Mi"),
			},
		}
		f := newFixture(t)
		c := f.RunController()

		resultErr := c.deleteResourceQuotaClaim(claim)

		assert.Error(t, resultErr, "resourcequotaclaims.ca-gip.github.com \"test\" not found")

	})

}

func TestIsDownscaleQuota(t *testing.T) {
	testCases := map[string]struct {
		claim        *cagipv1.ResourceQuotaClaim
		managedQuota *v1.ResourceQuota
		expect       bool
	}{
		"claiming more than the managed quota should return false": {
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			managedQuota: &v1.ResourceQuota{
				Spec: v1.ResourceQuotaSpec{
					Hard: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("900m"),
						v1.ResourceMemory: resource.MustParse("900Mi"),
					},
				},
			},
			expect: false,
		},
		"claiming less than the managed quota should return true": {
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			managedQuota: &v1.ResourceQuota{
				Spec: v1.ResourceQuotaSpec{
					Hard: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("2"),
						v1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
			expect: true,
		},
		"empty resources should return false": {
			claim:        &cagipv1.ResourceQuotaClaim{},
			managedQuota: &v1.ResourceQuota{},
			expect:       false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := isDownscaleQuota(testCase.claim, testCase.managedQuota)
			assert.Equal(t, result, testCase.expect)
		})
	}

}

func TestCanDownscaleQuota(t *testing.T) {
	testCases := map[string]struct {
		claim        *cagipv1.ResourceQuotaClaim
		totalRequest *v1.ResourceList
		expect       string
	}{
		"claim equals the total of request should return empty msg": {
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			totalRequest: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expect: utils.EmptyMsg,
		},
		"claiming more than the total of request should return empty msg": {
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("2"),
					v1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
			totalRequest: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expect: utils.EmptyMsg,
		},
		"claiming less CPU than the total of request should return CPU msg": {
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("900m"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			totalRequest: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expect: "Awaiting lower CPU consumption claiming 900m but current total of CPU request is 1",
		},
		"claiming less Memory than the total of request should return Memory msg": {
			claim: &cagipv1.ResourceQuotaClaim{
				Spec: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1"),
					v1.ResourceMemory: resource.MustParse("900Mi"),
				},
			},
			totalRequest: &v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("1"),
				v1.ResourceMemory: resource.MustParse("1Gi"),
			},
			expect: "Awaiting lower Memory consumption claiming 900Mi but current total of request is 1Gi",
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := canDownscaleQuota(testCase.claim, testCase.totalRequest)
			assert.Equal(t, result, testCase.expect)
		})
	}

}
