package k8s

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetPodInfo(c kubernetes.Interface, ns, name string) (*v1.Pod, error) {
	pod, _ := c.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
	return pod, nil
}

func GetPodsByLabelSelector(c kubernetes.Interface, ns string, labels map[string]string) []v1.Pod {
	var labelSelectors []string
	for k, v := range labels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", k, v))
	}
	ls := strings.Join(labelSelectors, ",")
	pods, _ := c.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: ls})
	return pods.Items
}
