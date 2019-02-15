package k8s

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NameSpacesFromLabels(c kubernetes.Interface, labels map[string]string) []string {
	var labelSelectors []string
	for k, v := range labels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", k, v))
	}
	ls := strings.Join(labelSelectors, ",")
	namespaces, _ := c.CoreV1().Namespaces().List(metav1.ListOptions{LabelSelector: ls})
	var ns []string
	for _, o := range namespaces.Items {
		ns = append(ns, o.Name)
	}
	return ns
}

func GetNamespace(c kubernetes.Interface, name string) *corev1.Namespace {
	namespace, _ := c.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	return namespace
}
