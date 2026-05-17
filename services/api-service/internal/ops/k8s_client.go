package ops

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubernetesClientset: in-cluster SA или kubeconfig из APP_K8S_KUBECONFIG (локальная отладка).
func NewKubernetesClientset() (kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err == nil {
		return kubernetes.NewForConfig(cfg)
	}
	path := strings.TrimSpace(os.Getenv("APP_K8S_KUBECONFIG"))
	if path != "" {
		cfg2, err2 := clientcmd.BuildConfigFromFlags("", path)
		if err2 != nil {
			return nil, fmt.Errorf("kubeconfig %s: %w", path, err2)
		}
		return kubernetes.NewForConfig(cfg2)
	}
	return nil, fmt.Errorf("kubernetes: %w (in-cluster unavailable; for local dev set APP_K8S_KUBECONFIG)", err)
}
