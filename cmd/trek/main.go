// Copyright © 2019 John Harris
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*

calicoctl can-i \
  --from-pod myPod \
  --ns dev \
  --proto TCP \
  their-app.ns.svc

Failed: Blocked by NetworkPolicy X

*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/johnharris85/trek/pkg/buildinfo"
	"github.com/johnharris85/trek/pkg/k8s"
	"github.com/sirupsen/logrus"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

var (
	printVersion bool
	kubeCfgFile  string
	debug        bool
	explain      bool
	namespace    string
	fromPod      string
	protocol     string
	port         string
)

func init() {
	// Add short flags?
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&kubeCfgFile, "config", "", "Location of kubecfg file for access to gimbal system kubernetes api, defaults to service account tokens")
	flag.StringVar(&namespace, "n", "", "Namespace")
	flag.StringVar(&fromPod, "from", "", "From pod")
	flag.StringVar(&protocol, "proto", "TCP", "Protocol")
	flag.StringVar(&port, "port", "", "Port")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging.")
	flag.BoolVar(&explain, "explain", false, "Explain decision.")
	flag.Parse()
}

func main() {
	var log = logrus.New()
	toService := flag.Args()[0]
	if printVersion {
		fmt.Println("trek")
		fmt.Printf("Version: %s\n", buildinfo.Version)
		fmt.Printf("Git commit: %s\n", buildinfo.GitSHA)
		fmt.Printf("Git tree state: %s\n", buildinfo.GitTreeState)
		os.Exit(0)
	}

	if debug {
		log.Level = logrus.DebugLevel
	}

	s := strings.Split(toService, ".")
	serviceName, serviceNamespace := s[0], s[1]

	targetService := k8s.Service{
		Name:               serviceName,
		Namespace:          serviceNamespace,
		Port:               port,
		Protocol:           protocol,
		AffectedByPolicies: []netv1.NetworkPolicy{},
		BlockedByPolicies:  []k8s.PolicyReason{},
		AllowedByPolicies:  []k8s.PolicyReason{},
	}
	if kubeCfgFile == "" {
		kubeCfgFile = os.Getenv("KUBECONFIG")
	}
	if kubeCfgFile == "" {
		log.Fatal("No Kubeconfig specified")
	}
	// Init client
	kubeClient, err := k8s.NewClient(kubeCfgFile, log)
	if err != nil {
		log.Fatal("Could not init k8sclient! ", err)
	}

	policyList, _ := kubeClient.NetworkingV1().NetworkPolicies(targetService.Namespace).List(metav1.ListOptions{})
	for _, p := range policyList.Items {
		a, _ := k8s.GetServiceLabels(kubeClient, targetService.Namespace, targetService.Name)
		s := labels.Set(a)
		e, _ := metav1.LabelSelectorAsSelector(&p.Spec.PodSelector)
		if e.Matches(s) {
			targetService.AffectedByPolicies = append(targetService.AffectedByPolicies, p)
		}
	}

	pod, _ := k8s.GetPodInfo(kubeClient, namespace, fromPod)
	for _, p := range targetService.AffectedByPolicies {
		if len(p.Spec.Ingress) == 0 {
			pr := k8s.PolicyReason{
				Policy: p,
				Reason: "Isolating Policy",
			}
			targetService.BlockedByPolicies = append(targetService.BlockedByPolicies, pr)
			continue
		}
		for _, rule := range p.Spec.Ingress {
			for _, b := range rule.From {
				// fmt.Println(p.Name, "===============================")
				// fmt.Printf("%d, %+v\n", i, b)
				if r, reason, _ := ruleBlocksOrigin(kubeClient, b, namespace, pod.Labels); r {
					pr := k8s.PolicyReason{
						Policy: p,
						Reason: reason,
					}
					targetService.BlockedByPolicies = append(targetService.BlockedByPolicies, pr)
				} else {
					pr := k8s.PolicyReason{
						Policy: p,
						Reason: reason,
					}
					if !k8s.AllowedByPorts(rule.Ports, targetService) {
						pr := k8s.PolicyReason{
							Policy: p,
							Reason: "Blocked by Ports",
						}
						targetService.BlockedByPolicies = append(targetService.BlockedByPolicies, pr)
					} else {
						targetService.AllowedByPolicies = append(targetService.AllowedByPolicies, pr)
					}
				}
			}

		}
		// fmt.Println(p.Spec.Egress)
	}

	if len(targetService.AllowedByPolicies) > 0 || len(targetService.AffectedByPolicies) == 0 {
		fmt.Println("Allowed")
		if explain {
			if len(targetService.AffectedByPolicies) == 0 {
				fmt.Println("-", "Not affected by any policies")
			}
			for _, p := range targetService.AllowedByPolicies {
				fmt.Println("- ", p.Policy.Name, ":", p.Reason, "\n")
			}
		}
		os.Exit(0)
	}

	if len(targetService.BlockedByPolicies) > 0 {
		fmt.Println("Disallowed")
		if explain {
			for _, p := range targetService.BlockedByPolicies {
				fmt.Println("- ", p.Policy.Name, ":", p.Reason, "\n")
			}
		}
		os.Exit(1)
	}

	// e := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)

	//
	// fmt.Println(pod.Labels)
	// fmt.Println(pod.Status.PodIP)
	// err = e.Encode(pod, os.Stdout)
	// if err != nil {
	// 	panic(err)
	// }

}

func ruleBlocksOrigin(kubeClient kubernetes.Interface, b netv1.NetworkPolicyPeer, ns string, l map[string]string) (bool, string, error) {
	if b.IPBlock != nil {
		return false, "Unsupported", errors.New("UNSUPPORTED") // FIX THIS OBVIOUSLY
	}

	if b.NamespaceSelector != nil && b.PodSelector == nil {
		a := k8s.GetNamespace(kubeClient, ns)
		// fmt.Println(a.Labels)
		s := labels.Set(a.Labels)
		e, _ := metav1.LabelSelectorAsSelector(b.NamespaceSelector)
		if e.Matches(s) {
			return false, fmt.Sprintf("Allowed by Namespace Selector : %v", e), nil
		}
		return true, fmt.Sprintf("Disallowed by Namespace Selector : %v", e), nil
	}

	if b.PodSelector != nil && b.NamespaceSelector == nil {
		s := labels.Set(l)
		e, _ := metav1.LabelSelectorAsSelector(b.PodSelector)
		if e.Matches(s) {
			return false, fmt.Sprintf("Allowed by Pod Selector : %v", e), nil
		}
		return true, fmt.Sprintf("Disallowed by Pod Selector : %v", e), nil
	}

	if b.PodSelector != nil && b.NamespaceSelector != nil {
		var nsBool, podBool bool
		a := k8s.GetNamespace(kubeClient, ns)
		s := labels.Set(a.Labels)
		d, _ := metav1.LabelSelectorAsSelector(b.NamespaceSelector)
		if d.Matches(s) {
			nsBool = true
		}
		s = labels.Set(l)
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
