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
	client, teamclient := client.GetKubernetesCRDClient()

	teaminformer := util.GetTeamsSharedIndexInformer(client,teamclient)
	queue := util.CreateWorkingQueue()
	util.AddPodsEventHandler(teaminformer, queue)

	// construct the Controller object which has all of the necessary components to
	// handle logging, connections, informing (listing and watching), the queue,
	// and the handler
	controller := controllers.TeamController{
		Logger:    log.NewEntry(log.New()),
		Clientset: client,
		Informer:  teaminformer,
		Queue:     queue,
		Handler:   handler.TeamHandler{},
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