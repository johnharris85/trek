package trek

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetNamespace...
func GetNamespace(c kubernetes.Interface, name string) *corev1.Namespace {
	namespace, _ := c.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	return namespace
}
