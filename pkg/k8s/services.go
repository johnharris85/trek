package k8s

import (
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Service struct {
	Name               string
	Port               string
	Protocol           string
	Namespace          string
	AffectedByPolicies []netv1.NetworkPolicy
	BlockedByPolicies  []PolicyReason
	AllowedByPolicies  []PolicyReason
}

type PolicyReason struct {
	Policy netv1.NetworkPolicy
	Reason string
}

// func parseSvcString(svc string) *TargetService {

// }

func GetServiceLabels(c kubernetes.Interface, ns, name string) (map[string]string, error) {
	svc, _ := c.CoreV1().Services(ns).Get(name, metav1.GetOptions{})
	return svc.Spec.Selector, nil
}

func AllowedByPorts(ports []netv1.NetworkPolicyPort, svc Service) bool {
	if len(ports) == 0 {
		return true
	}
	for _, port := range ports {
		if (*port.Port).String() == svc.Port && string(*port.Protocol) == svc.Protocol {
			return true
		}
	}
	return false
}
