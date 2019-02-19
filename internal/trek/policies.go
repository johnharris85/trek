package trek

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// PolicyDecision ...
type PolicyDecision struct {
	Policy netv1.NetworkPolicy
	Reason string
}

func ruleBlocksSource(kubeClient kubernetes.Interface, b *netv1.NetworkPolicyPeer, sourcePod *corev1.Pod) (bool, string, error) {
	if b.IPBlock != nil {
		return false, "Unsupported", errors.New("UNSUPPORTED") // FIX THIS OBVIOUSLY
	}

	if b.NamespaceSelector != nil && b.PodSelector == nil {
		a := GetNamespace(kubeClient, sourcePod.Namespace)
		// fmt.Println(a.Labels)
		s := labels.Set(a.Labels)
		e, _ := metav1.LabelSelectorAsSelector(b.NamespaceSelector)
		if e.Matches(s) {
			return false, fmt.Sprintf("Allowed by Namespace Selector : %v", e), nil
		}
		return true, fmt.Sprintf("Disallowed by Namespace Selector : %v", e), nil
	}

	if b.PodSelector != nil && b.NamespaceSelector == nil {
		s := labels.Set(sourcePod.Labels)
		e, _ := metav1.LabelSelectorAsSelector(b.PodSelector)
		if e.Matches(s) {
			return false, fmt.Sprintf("Allowed by Pod Selector : %v", e), nil
		}
		return true, fmt.Sprintf("Disallowed by Pod Selector : %v", e), nil
	}

	if b.PodSelector != nil && b.NamespaceSelector != nil {
		var nsBool, podBool bool
		a := GetNamespace(kubeClient, sourcePod.Namespace)
		s := labels.Set(a.Labels)
		d, _ := metav1.LabelSelectorAsSelector(b.NamespaceSelector)
		if d.Matches(s) {
			nsBool = true
		}
		s = labels.Set(sourcePod.Labels)
		e, _ := metav1.LabelSelectorAsSelector(b.PodSelector)
		if e.Matches(s) {
			podBool = true
		}
		if nsBool && podBool {
			return false, fmt.Sprintf("Allowed by Namespace (%v) && Pod (%v) Selectors", d, e), nil
		} else if !nsBool {
			return true, fmt.Sprintf("Allowed by Pod Selector (%v) but Disallowed by Namespace (%v)", e, d), nil
		} else if !podBool {
			return true, fmt.Sprintf("Allowed by Namespace Selector (%v) but Disallowed by Pod (%v)", d, e), nil
		}
	}
	return false, "SHOULD NEVER GET HERE", errors.New("SHOULD NEVER GET HERE")
}

func allowedByPorts(ports []netv1.NetworkPolicyPort, svc *Service) bool {
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
