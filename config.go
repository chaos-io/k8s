package k8s

type Config struct {
	MasterUrl       string `json:"masterUrl"`
	KubeConfig      string `json:"kubeConfig"`
	NFSServer       string `json:"nfsServer"`
	NFSPath         string `json:"nfsPath"`
	KubeHarbor      string `json:"kubeHarbor"`
	ImagePullSecret string `json:"imagePullSecret"`
	Namespace       string `json:"namespace" default:"default"`
}
