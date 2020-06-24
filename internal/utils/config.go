package utils

import (
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	configMapName              = "kotary-config"
	defaultMaxAllocationMemory = 1
	defaultMaxAllocationCPU    = 1
	defaultOverCommitMemory    = 1
	defaultOverCommitCPU       = 1
	nsSecretPath               = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

var claimSpecByDefault = &v1.ResourceList{
	v1.ResourceCPU:    resource.MustParse("2"),
	v1.ResourceMemory: resource.MustParse("6Gi"),
}

// Hold the configurations specification
type Config struct {

	// The spec of the default ResourceQuotaClaim to apply on Namespaces
	DefaultClaimSpec v1.ResourceList `yaml:"defaultClaimSpec"`

	// Maximum resource size that can be claimed compared to the total cluster size
	// 0.3 -> Max claim size will be a third of the cluster resources
	RatioMaxAllocationMemory float64 `yaml:"ratioMaxAllocationMemory"`
	RatioMaxAllocationCPU    float64 `yaml:"ratioMaxAllocationCPU"`

	// Apply an over provisioning on the available resources of all the nodes
	// Represented as a percentage (could be under 100 to under provision)
	RatioOverCommitMemory float64 `yaml:"ratioOverCommitMemory"`
	RatioOverCommitCPU    float64 `yaml:"ratioOverCommitCPU"`
}

// Hold the config and a clienset to retrieve it
type ConfigurationManager struct {
	clientset kubernetes.Interface
	Conf      Config
}

// Create a new instance
func NewSettingManger(clientset kubernetes.Interface) *ConfigurationManager {
	return &ConfigurationManager{
		clientset: clientset,
	}

}

// Fallback when nothing has been set
func (c *ConfigurationManager) generateDefaultSettings() *Config {
	klog.V(4).Info("Generating default setting ...")
	klog.V(6).Info("Default setting will not select any Namespaces to provision default ResourceQuotaClaim")
	klog.V(6).Info("Default will not apply over commitment to Nodes available resources")

	return &Config{
		DefaultClaimSpec:         *claimSpecByDefault,
		RatioMaxAllocationMemory: defaultMaxAllocationMemory,
		RatioMaxAllocationCPU:    defaultMaxAllocationCPU,
		RatioOverCommitMemory:    defaultOverCommitMemory,
		RatioOverCommitCPU:       defaultOverCommitCPU,
	}

}

// Load configmap based on where the controller is running
func (c *ConfigurationManager) Load() {

	namespace, err := findExecutionNamespace()

	if err != nil {
		klog.Infof("Could not load namespace via %s ", nsSecretPath)
		c.Conf = *c.generateDefaultSettings()
		return
	}

	configMap, err := c.loadConfigMap(namespace)

	if err != nil {
		klog.Infof("Could not find %s configMap in ns %s", configMapName, namespace)
		c.Conf = *c.generateDefaultSettings()
		return
	}

	config, _ := parseConfigMap(configMap)

	c.Conf = *config

	return

}

// Find the namespace where the controller is being executed
func findExecutionNamespace() (namespace string, err error) {

	nsFile, err := ioutil.ReadFile(nsSecretPath)

	namespace = strings.TrimSpace(string(nsFile))

	if err != nil {
		return namespace, err
	} else {
		return namespace, nil
	}
}

// Load the configmap
func (c ConfigurationManager) loadConfigMap(namespace string) (configMap *v1.ConfigMap, err error) {

	configMap, err = c.clientset.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})

	if err != nil {
		return configMap, err
	}

	return configMap, nil

}

// Parse the ConfigMap to the Config struct
func parseConfigMap(configMap *v1.ConfigMap) (parsed *Config, err error) {

	var ratioMaxAllocationMemory float64
	err = yaml.Unmarshal([]byte(configMap.Data["ratioMaxAllocationMemory"]), &ratioMaxAllocationMemory)
	if len(configMap.Data["ratioMaxAllocationMemory"]) == 0 || err != nil {
		ratioMaxAllocationMemory = defaultMaxAllocationMemory
	}

	var ratioMaxAllocationCPU float64
	err = yaml.Unmarshal([]byte(configMap.Data["ratioMaxAllocationCPU"]), &ratioMaxAllocationCPU)
	if len(configMap.Data["ratioMaxAllocationCPU"]) == 0 || err != nil {
		ratioMaxAllocationCPU = defaultMaxAllocationCPU
	}

	var ratioOverCommitMemory float64
	err = yaml.Unmarshal([]byte(configMap.Data["ratioOverCommitMemory"]), &ratioOverCommitMemory)
	if len(configMap.Data["ratioOverCommitMemory"]) == 0 || err != nil {
		ratioOverCommitMemory = defaultOverCommitMemory
	}

	var ratioOverCommitCPU float64
	err = yaml.Unmarshal([]byte(configMap.Data["ratioOverCommitCPU"]), &ratioOverCommitCPU)
	if len(configMap.Data["ratioOverCommitCPU"]) == 0 || err != nil {
		ratioOverCommitCPU = defaultOverCommitCPU
	}

	var defaultClaimSpec v1.ResourceList
	err = yaml.Unmarshal([]byte(configMap.Data["defaultClaimSpec"]), &defaultClaimSpec)
	if len(configMap.Data["defaultClaimSpec"]) == 0 || err != nil {
		defaultClaimSpec = *claimSpecByDefault
	}

	parsed = &Config{
		DefaultClaimSpec:         defaultClaimSpec,
		RatioMaxAllocationMemory: ratioMaxAllocationMemory,
		RatioMaxAllocationCPU:    ratioMaxAllocationCPU,
		RatioOverCommitMemory:    ratioOverCommitMemory,
		RatioOverCommitCPU:       ratioOverCommitCPU,
	}

	klog.Infof("Loaded config map : %+v\n", parsed)

	return

}
