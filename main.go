package main

import (
	log "github.com/Sirupsen/logrus"

	// "github.com/cmoulliard/k8s-team-crd/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// Get the Kubernetes client to access the Cloud platform
	client := GetKubernetesClient()

	ns, nsError := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if nsError != nil {
		log.Fatalf("Can't list namespaces ", nsError)
	}
	for i := range ns.Items {
		log.Info("Namespace/project : ", ns.Items[i].Name)
	}
}
