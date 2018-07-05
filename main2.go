package main

import (
	log "github.com/Sirupsen/logrus"

	"github.com/cmoulliard/k8s-team-crd/pkg/client"
	controllers "github.com/cmoulliard/k8s-team-crd/pkg/controller"
	"github.com/cmoulliard/k8s-team-crd/pkg/handler"
	"github.com/cmoulliard/k8s-team-crd/pkg/util"

	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Get the Kubernetes client to access the Cloud platform
	client := client.GetKubernetesClient()

	informer := util.GetPodsSharedIndexInformer(client)
	queue := util.CreateWorkingQueue()
	util.AddPodsEventHandler(informer, queue)

	// construct the Controller object which has all of the necessary components to
	// handle logging, connections, informing (listing and watching), the queue,
	// and the handler
	controller := controllers.Controller{
		Logger:    log.NewEntry(log.New()),
		Clientset: client,
		Informer:  informer,
		Queue:     queue,
		Handler:   handler.SimpleHandler{},
	}

	// use a channel to synchronize the finalization for a graceful shutdown
	stopCh := make(chan struct{})
	defer close(stopCh)

	// run the controller loop to process items
	go controller.Run(stopCh)

	// use a channel to handle OS signals to terminate and gracefully shut
	// down processing
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)
	<-sigTerm
}