package k8s

import (
	"log"
	"os"

	"github.com/chaos-io/chaos/config"
	"github.com/chaos-io/chaos/logs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	k8sCfg    Config
	kubeConf  *rest.Config
	clientSet *kubernetes.Clientset
)

func init() {
	var err error
	if err := config.ScanFrom(&k8sCfg, "k8s"); err != nil {
		log.Panicf("failed to get k8s config, error: %v", err)
	}
	logs.Debugw("get k8sConfig", "config", k8sCfg, "hostname", os.Getenv("HOSTNAME"))

	// 注意：在pod中找不到KubeConfig配置路径
	kubeConf, err = clientcmd.BuildConfigFromFlags(k8sCfg.MasterUrl, k8sCfg.KubeConfig)
	if err != nil {
		log.Panicf("failed to get k8s config, error=%v", err)
	}

	clientSet, err = kubernetes.NewForConfig(kubeConf)
	if err != nil {
		log.Panicf("failed to get k8s clientSet, error=%v", err)
	}
}

func kubeImage(imageName string) string {
	return k8sCfg.KubeHarbor + "/" + imageName
}
