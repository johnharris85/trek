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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/johnharris85/trek/internal/buildinfo"
	"github.com/johnharris85/trek/internal/k8s"
	"github.com/johnharris85/trek/internal/trek"

	"github.com/sirupsen/logrus"
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

	if debug {
		log.Level = logrus.DebugLevel
	}

	if printVersion {
		fmt.Println("trek")
		fmt.Printf("Version: %s\n", buildinfo.Version)
		fmt.Printf("Git commit: %s\n", buildinfo.GitSHA)
		fmt.Printf("Git tree state: %s\n", buildinfo.GitTreeState)
		os.Exit(0)
	}

	if kubeCfgFile == "" {
		v, exists := os.LookupEnv("KUBECONFIG")
		if !exists {
			log.Fatal("No Kubeconfig specified")
		}
		kubeCfgFile = v
	}

	// Init client
	kubeClient, err := k8s.NewClient(kubeCfgFile)
	if err != nil {
		log.Fatal("Could not init k8sclient! ", err)
	}

	target := flag.Args()[0]

	svcName, svcNamespace := trek.ParseServiceString(target)

	targetService := trek.NewService(svcName, svcNamespace, port, protocol)

	err = targetService.CalculateMatchingPolicies(kubeClient)

	sourcePod, _ := trek.GetPodInfo(kubeClient, namespace, fromPod)

	err = targetService.CalculateBlockingPolicies(kubeClient, sourcePod)

	if len(targetService.AllowedByPolicies) > 0 || len(targetService.AffectedByPolicies.Items) == 0 {
		fmt.Println("Allowed")
		if explain {
			if len(targetService.AffectedByPolicies.Items) == 0 {
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
