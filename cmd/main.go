package main

import (
	"flag"
	"github.com/ca-gip/kotary/internal/controller"
	"github.com/ca-gip/kotary/internal/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	clientset "github.com/ca-gip/kotary/pkg/generated/clientset/versioned"
	informers "github.com/ca-gip/kotary/pkg/generated/informers/externalversions"
)

var (
	masterURL  string
	kubeconfig string
)

const resyncPeriod = time.Minute * 30

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig(), "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	klog.InitFlags(nil)

	flag.Parse()

	// Load kube config
	cfg, err := rest.InClusterConfig()
	if err != nil {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			klog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	// Generate clientsets
	settingsClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	namespaceClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	quotaClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	nodeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	podClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	quotaClaimClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	// Load config
	settingsManger := utils.NewSettingManger(settingsClient)
	settingsManger.Load()

	namespaceInformerFactory := kubeinformers.NewSharedInformerFactory(namespaceClient, resyncPeriod)
	quotaInformerFactory := kubeinformers.NewSharedInformerFactory(quotaClient, resyncPeriod)
	nodeInformerFactory := kubeinformers.NewSharedInformerFactory(nodeClient, resyncPeriod)
	podInformerFactory := kubeinformers.NewSharedInformerFactory(podClient, resyncPeriod)
	quotaClaimInformerFactory := informers.NewSharedInformerFactory(quotaClaimClient, resyncPeriod)

	kotaryController := controller.NewController(
		settingsManger.Conf,
		namespaceClient, quotaClient, nodeClient, podClient, quotaClaimClient,
		namespaceInformerFactory.Core().V1().Namespaces(),
		quotaInformerFactory.Core().V1().ResourceQuotas(),
		nodeInformerFactory.Core().V1().Nodes(),
		podInformerFactory.Core().V1().Pods(),
		quotaClaimInformerFactory.Cagip().V1().ResourceQuotaClaims())

	namespaceInformerFactory.Start(wait.NeverStop)
	quotaInformerFactory.Start(wait.NeverStop)
	nodeInformerFactory.Start(wait.NeverStop)
	podInformerFactory.Start(wait.NeverStop)
	quotaClaimInformerFactory.Start(wait.NeverStop)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9080", nil)

	if err = kotaryController.Run(2, wait.NeverStop); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

}

func defaultKubeconfig() string {
	fname := os.Getenv("KUBECONFIG")
	if fname != "" {
		return fname
	}
	home, err := os.UserHomeDir()
	if err != nil {
		klog.Warningf("failed to get home directory: %v", err)
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
