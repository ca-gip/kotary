package e2etest

import (
	
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	cagipv1 "github.com/ca-gip/kotary/pkg/apis/ca-gip/v1"
	versioned "github.com/ca-gip/kotary/pkg/generated/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const testnamespace="native-development"


func NewVersionedClientSet(kubeconfig string) (*versioned.Clientset, error) {
	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientsetVersioned, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientsetVersioned, nil
}

// check e2etest ResourceQuotaClaim
func TestCreateQuotaClaim(t *testing.T) {
	
	clientsetVersioned, err := NewVersionedClientSet("")
	if err != nil {
		panic(err)
	}
	_,err=clientsetVersioned.CagipV1().ResourceQuotaClaims(testnamespace).Create(&cagipv1.ResourceQuotaClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:testnamespace,
			Name: "e2etest",
		},
		Spec: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("151m"),
								v1.ResourceMemory: resource.MustParse("151Mi"),
							}})
	assert.NoError(t, err, "Failed to create ResourceQuotaClaim %s in namespace %s", "e2etest", testnamespace)
	
}
// check e2etest ResourceQuotaClaim
func TestGetQuotaClaim(t *testing.T) {
	
	clientsetVersioned, err := NewVersionedClientSet("")
	if err != nil {
		panic(err)
	}
	
	quota, err := clientsetVersioned.CagipV1().ResourceQuotaClaims(testnamespace).Get("e2etest",metav1.GetOptions{})
	assert.NoError(t, err, "Failed to get ResourceQuotaClaim %s in namespace %s", "e2etest", testnamespace)
	assert.NotEmpty(t, quota, "Empty ResourceQuotaClaim %s in namespace %s", "e2etest", testnamespace)
	
}

//  Check Update ResourceQuotaClaim
func TestUpddateQuotaClaim(t *testing.T) {

	clientsetVersioned, err := NewVersionedClientSet("")
	if err != nil {
		panic(err)
	}
	clientsetVersioned.CagipV1().ResourceQuotaClaims(testnamespace).Update(&cagipv1.ResourceQuotaClaim{
																					ObjectMeta: metav1.ObjectMeta{
																						Namespace:testnamespace,
																						Name: "e2etest",
																					},
																					Spec: v1.ResourceList{
																											v1.ResourceCPU:    resource.MustParse("151m"),
																											v1.ResourceMemory: resource.MustParse("512Gi"),
																										}})
	
	assert.NoError(t, err, "Failed to Update ResourceQuotaClaim %s in namespace %s", "e2etest",testnamespace)
	
}

