package trek

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Service represents a kubernetes target service
type Service struct {
	Name               string
	Port               string
	Protocol           string
	Namespace          string
	AffectedByPolicies *netv1.NetworkPolicyList
	BlockedByPolicies  []PolicyDecision
	AllowedByPolicies  []PolicyDecision
}

// ^^ just add a single list of policydecisions with a type "affected, blocked, allowed, etc..."?

// ParseServiceString ...
func ParseServiceString(svc string) (string, string) {
	svcSplit := strings.Split(svc, ".")
	return svcSplit[0], svcSplit[1]
}

// NewService ...
func NewService(name, namespace, port, protocol string) *Service {
	return &Service{
		Name:               name,
		Namespace:          namespace,
		Port:               port,
		Protocol:           protocol,
		AffectedByPolicies: &netv1.NetworkPolicyList{},
		BlockedByPolicies:  []PolicyDecision{},
		AllowedByPolicies:  []PolicyDecision{},
	}
}

// CalculateMatchingPolicies inspects all NetworkPolicies in the namespace of a service and adds them to the AffectedByPolicies slice
func (s *Service) CalculateMatchingPolicies(kubeClient kubernetes.Interface) error {
	policyList, err := kubeClient.NetworkingV1().NetworkPolicies(s.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, p := range policyList.Items {
		a, _ := getServiceLabels(kubeClient, s.Namespace, s.Name)
		set := labels.Set(a)
		e, _ := metav1.LabelSelectorAsSelector(&p.Spec.PodSelector)
		if e.Matches(set) {
			s.AffectedByPolicies.Items = append(s.AffectedByPolicies.Items, p)
		}
	}
	return nil
}

// CalculateBlockingPolicies inspects all ImpactingNetworkPolicies for a service and determines if the source pod is blocked by them
func (s *Service) CalculateBlockingPolicies(kubeClient kubernetes.Interface, sourcePod *corev1.Pod) error {
	for _, p := range s.AffectedByPolicies.Items {
		if len(p.Spec.Ingress) == 0 {
			pr := PolicyDecision{
				Policy: p,
				Reason: "Isolating Policy",
			}
			s.BlockedByPolicies = append(s.BlockedByPolicies, pr)
			continue
		}
		for _, rule := range p.Spec.Ingress {
			for _, b := range rule.From {
				// fmt.Println(p.Name, "===============================")
				// fmt.Printf("%d, %+v\n", i, b)
				if r, reason, _ := ruleBlocksSource(kubeClient, &b, sourcePod); r {
					pr := PolicyDecision{
						Policy: p,
						Reason: reason,
					}
					s.BlockedByPolicies = append(s.BlockedByPolicies, pr)
				} else {
					pr := PolicyDecision{
						Policy: p,
						Reason: reason,
					}
					if !allowedByPorts(rule.Ports, s) {
						pr := PolicyDecision{
							Policy: p,
							Reason: "Blocked by Ports",
						}
						s.BlockedByPolicies = append(s.BlockedByPolicies, pr)
					} else {
						s.AllowedByPolicies = append(s.AllowedByPolicies, pr)
					}
				}
			}

		}
		// fmt.Println(p.Spec.Egress)
	}
	return nil
}

func getServiceLabels(c kubernetes.Interface, ns, name string) (map[string]string, error) {
	svc, _ := c.CoreV1().Services(ns).Get(name, metav1.GetOptions{})
	return svc.Spec.Selector, nil
}
