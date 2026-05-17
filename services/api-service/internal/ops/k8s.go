package ops

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// Соответствие коротких id (как в UI) → имя Deployment в namespace.
var k8sDeploymentByService = map[string]string{
	"api":  "api-service",
	"auth": "auth-service",
	"ref":  "reference-data-service",
	"prc":  "processing-service",
	"jir":  "jira-integration-service",
	"sem":  "semgrep-service",
	"gls":  "gitleaks-service",
}

// K8sRunner читает логи подов и делает rollout restart деплоя (как kubectl).
type K8sRunner struct {
	Client    kubernetes.Interface
	Namespace string
	Container string
}

func NewK8sRunner(c kubernetes.Interface, namespace, container string) *K8sRunner {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "asoc"
	}
	cn := strings.TrimSpace(container)
	if cn == "" {
		cn = "app"
	}
	return &K8sRunner{Client: c, Namespace: ns, Container: cn}
}

func (k *K8sRunner) resolveDeployment(serviceID string) (string, bool) {
	if k == nil || k.Client == nil {
		return "", false
	}
	id := strings.ToLower(strings.TrimSpace(serviceID))
	name, ok := k8sDeploymentByService[id]
	return name, ok
}

func (k *K8sRunner) pickPodName(ctx context.Context, deploymentName string) (string, error) {
	label := "app=" + deploymentName
	pods, err := k.Client.CoreV1().Pods(k.Namespace).List(ctx, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return "", err
	}
	var running []string
	for _, p := range pods.Items {
		if p.DeletionTimestamp != nil {
			continue
		}
		if p.Status.Phase == corev1.PodRunning {
			running = append(running, p.Name)
		}
	}
	sort.Strings(running)
	if len(running) > 0 {
		return running[0], nil
	}
	var any []string
	for _, p := range pods.Items {
		if p.DeletionTimestamp == nil {
			any = append(any, p.Name)
		}
	}
	sort.Strings(any)
	if len(any) > 0 {
		return any[0], nil
	}
	return "", fmt.Errorf("нет подов с селектором %s в namespace %s", label, k.Namespace)
}

// Logs implements Backend.
func (k *K8sRunner) Logs(ctx context.Context, serviceID string, tail int) ([]byte, error) {
	dep, ok := k.resolveDeployment(serviceID)
	if !ok {
		return nil, fmt.Errorf("unknown service id")
	}
	if tail < 1 {
		tail = 200
	}
	if tail > 2000 {
		tail = 2000
	}
	tail64 := int64(tail)
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	pod, err := k.pickPodName(ctx, dep)
	if err != nil {
		return nil, err
	}
	req := k.Client.CoreV1().Pods(k.Namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: k.Container,
		TailLines: &tail64,
		Timestamps: true,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	out, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	if len(out) > maxLogBytes {
		trunc := []byte("(показан хвост лога, объём ограничен)\n\n")
		out = append(trunc, out[len(out)-maxLogBytes:]...)
	}
	return out, nil
}

// Restart implements Backend — strategic-merge patch deployment (kubectl rollout restart).
func (k *K8sRunner) Restart(ctx context.Context, serviceID string) ([]byte, error) {
	dep, ok := k.resolveDeployment(serviceID)
	if !ok {
		return nil, fmt.Errorf("unknown service id")
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	_, err := k.Client.AppsV1().Deployments(k.Namespace).Get(ctx, dep, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, ts)
	_, err = k.Client.AppsV1().Deployments(k.Namespace).Patch(ctx, dep, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("deployment/%s: запланирован rolling restart (restartedAt=%s)\n", dep, ts)
	return []byte(msg), nil
}

var _ Backend = (*K8sRunner)(nil)
