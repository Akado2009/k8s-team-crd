# Play with a k8s controller and CustomeResourceDefinition

   * [Play with a k8s controller and CustomeResourceDefinition](#play-with-a-k8s-controller-and-customeresourcedefinition)
      * [Prerequisites](#prerequisites)
      * [Create a golang project](#create-a-golang-project)
      * [Setup a k8s client to communicate with the platform](#setup-a-k8s-client-to-communicate-with-the-platform)
      * [Design a simple controler](#design-a-simple-controler)

## Prerequisites

- Go Lang : [>=1.9](https://golang.org/doc/install)
- [Dep tool](https://github.com/golang/dep)
- [GOWORKSPACE](https://golang.org/doc/code.html#Workspaces) variable defined 

## Create a golang project

- Move to your `$GOPATH` directory and create under `src/github.com/$USER` a new project

  ```bash
  export USER="cmoulliard"
  cd $GOPATH/src/github.com && mkdir -p $USER/k8s-controller-demo
  cd $USER/k8s-controller-demo
  ```
  
- Move to the new project and create the following folders's tree and files
  using the bash commands `mkdir -p {pkg/client,vendor}` and `touch {Gopkg.toml,main.go}`

  ```bash
  .
  |-- Gopkg.toml
  |-- main.go
  |-- pkg
  |   `-- client
  |       `-- kube.go
  `-- vendor
  ```

## Setup a k8s client to communicate with the platform

- Add a `pkg/client/kube.go` file using the command `touch pkg/client/kube.go` and develop the `GetKubernetClient` function to return a k8s go client.
  
  **Remark**: We will pass as command's line parameter the path to access the file the `$HOME/.kube/config`. It contains the `default context` to be used to access the k8s platform.
  If the parameter is not passed on the command, then it will be calculated from the `home` directory.

  ```go
  package client
  
  import (
  	"flag"
  	"path/filepath"
  	"os"
  
  	log "github.com/Sirupsen/logrus"
  	"k8s.io/client-go/kubernetes"
  	"k8s.io/client-go/tools/clientcmd"
  )
  
  // retrieve the Kubernetes cluster client from outside of the cluster
  func GetKubernetesClient() kubernetes.Interface {
  	var kubeconfig *string
  	if home := homeDir(); home != "" {
  		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
  	} else {
  		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
  	}
  	// Parse the command line arguments
  	flag.Parse()
  
  	// create the config from the path
  	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
  	if err != nil {
  		log.Fatalf("getClusterConfig: %v", err)
  	}
  
  	// generate the client based off of the config
  	client, err := kubernetes.NewForConfig(config)
  	if err != nil {
  		log.Fatalf("getClusterConfig: %v", err)
  	}
  
  	log.Info("Successfully constructed k8s client")
  	return client
  }
  
  func homeDir() string {
  	if h := os.Getenv("HOME"); h != "" {
  		return h
  	}
  	return os.Getenv("USERPROFILE") // windows
  }
  ```

- Next, develop the `main.go` file which contains the instructions to `create a K8s client`

  **WARNING**: Replace the `$USER` and `$PROJECT` variables with their respective values to define the package to be imported

  ```go
  package main
  
  import (
  	log "github.com/Sirupsen/logrus"
    "github.com/$USER/$PROJECT/pkg/client"
  	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  )
  
  func main() {
  	// Get the Kubernetes client to access the Cloud platform
  	client := client.GetKubernetesClient()
  }  
  ```

- Call the k8s APi to get by example the list of the `namespaces`

  ```go
  ns, nsError := client.CoreV1().Namespaces().List(metav1.ListOptions{})
  if nsError != nil {
  	log.Fatalf("Can't list namespaces ", nsError)
  }
  for i := range ns.Items {
  	log.Info("Namespace/project : ", ns.Items[i].Name)
  }
  ```

- Initialize dep tool locally

  ```bash
  dep init
  ```

- Define the additional packages required by editing the `Gopkg.toml`
  ```toml
  [[constraint]]
    name = "k8s.io/client-go"
    version = "6.0.0"
  
  [[constraint]]
    name = "k8s.io/apimachinery"
    version = "kubernetes-1.9.9"
  ```

- Grab the packages.
  
  **Remark** : The packages downloaded will be stored under the local `vendor` directory

  ```bash
  dep ensure
  ```
 
- Run the application locally

  ```bash
  go run main.go -kubeconfig=$HOME/.kube/configINFO[0000] Successfully constructed k8s client          
  
  INFO[0000] Namespace/project : default                  
  INFO[0000] Namespace/project : k8s-info                 
  INFO[0000] Namespace/project : k8s-supervisord          
  INFO[0000] Namespace/project : kube-public              
  INFO[0000] Namespace/project : kube-system              
  INFO[0000] Namespace/project : my-crd                   
  INFO[0000] Namespace/project : myproject                
  INFO[0000] Namespace/project : openshift                
  INFO[0000] Namespace/project : openshift-infra          
  INFO[0000] Namespace/project : openshift-node           
  INFO[0000] Namespace/project : openshift-web-console   

  ```   
  
## Design a simple controller

TODO

- Create a `proxy.go` file under the folder `pkg/util` where we will create the 
  different objects such as the `informer`, the `event handler` and the `queue`
  used by the controller to be informed about the `create,update,delete` events
   ```bash
   mkdir -p pkg/util && touch pkg/util/proxy.go
   ```
   
- Define a functuion to return a `cache.NewSharedIndexInformer` to watch or list the `pods` published within thenamespace `default`
  ```go
  package util
  
  import (
  	api_v1 "k8s.io/api/core/v1"
  	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  	"k8s.io/apimachinery/pkg/runtime"
  	"k8s.io/apimachinery/pkg/watch"
  	"k8s.io/client-go/kubernetes"
  	"k8s.io/client-go/tools/cache"
  )
  
  func GetPodsSharedIndexInformer(client kubernetes.Interface) cache.NewSharedIndexInformer {
  	return cache.NewSharedIndexInformer(
  		// the ListWatch contains two different functions that our
  		// informer requires: ListFunc to take care of listing and watching
  		// the resources we want to handle
  		&cache.ListWatch{
  			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
  				// list all of the pods (core resource) in the deafult namespace
  				return client.CoreV1().Pods(meta_v1.NamespaceDefault).List(options)
  			},
  			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
  				// watch all of the pods (core resource) in the default namespace
  				return client.CoreV1().Pods(meta_v1.NamespaceDefault).Watch(options)
  			},
  		},
  		&api_v1.Pod{}, // the target type (Pod)
  		0,             // no resync (period of 0)
  		cache.Indexers{},
  	)
  }
  ```
  
- Next, create a function which is responsible to create a `workqueue`
  ```go
  import (
      	"k8s.io/client-go/util/workqueue"  
  )
  func CreateWorkingQueue() workqueue.RateLimitingInterface {
  	// a result of listing or watching, we can add an idenfitying key to the queue
  	// so that it can be handled in the handler
  	return workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
  }
  ```
    
- Add an `eventHandler` to manage the pods's resources
  ```go
  func (inf *cache.SharedIndexInformer) AddPodsEventHandler() {
  	// add event handlers to handle the three types of events for resources:
  	//  - adding new resources
  	//  - updating existing resources
  	//  - deleting resources
  	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
  		AddFunc: func(obj interface{}) {
  			// convert the resource object into a key (in this case
  			// we are just doing it in the format of 'namespace/name')
  			key, err := cache.MetaNamespaceKeyFunc(obj)
  			log.Infof("Add pod: %s", key)
  			if err == nil {
  				// add the key to the queue for the handler to get
  				workqueue.queue.Add(key)
  			}
  		},
  		UpdateFunc: func(oldObj, newObj interface{}) {
  			key, err := cache.MetaNamespaceKeyFunc(newObj)
  			log.Infof("Update pod: %s", key)
  			if err == nil {
  				workqueue.queue.Add(key)
  			}
  		},
  		DeleteFunc: func(obj interface{}) {
  			// DeletionHandlingMetaNamsespaceKeyFunc is a helper function that allows
  			// us to check the DeletedFinalStateUnknown existence in the event that
  			// a resource was deleted but it is still contained in the index
  			//
  			// this then in turn calls MetaNamespaceKeyFunc
  			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
  			log.Infof("Delete pod: %s", key)
  			if err == nil {
  				queue.Add(key)
  			}
  		},
  	})
  }
  ```
 
- Create the `Simple.go` file within the folder `pkg/controller/` 
   ```bash
   mkdir -p pkg/controller && touch pkg/controller/simple.go
   ```
- Specify the type's definition of the `Controller` struct with a
  - `log.Entry`
  - `kubernetes.Interface` containing the API types
  - `workqueue.RateLimitingInterface` where events are published
  - `cache.SharedIndexInformer` listening to the K8s APi `list` or `watch` 
  - `Handler` managing the logic of this simple controller
  
  **WARNING**: Replace the `$USER` and `$PROJECT` variables with their respective values to define the package to be imported  

   ```go
   package controller
   
   import (
   	log "github.com/Sirupsen/logrus"
   	"k8s.io/client-go/kubernetes"
   	"k8s.io/client-go/tools/cache"
   	"k8s.io/client-go/util/workqueue" 	
 	"github.com/$USER/$PROJECT/pkg/handler"
   )
   
   // Controller struct defines how a controller should encapsulate
   // logging, client connectivity, informing (list and watching)
   // queueing, and handling of resource changes
   type Controller struct {
   	logger    *log.Entry
   	clientset kubernetes.Interface
   	queue     workqueue.RateLimitingInterface
   	informer  cache.SharedIndexInformer
	handler   handler.SimpleHandler
   }
   ```
 
- Create the Handler `Simple.go` file within the folder `pkg/handler/` 
  ```bash
  mkdir -p pkg/handler && touch pkg/handler/simple.go
  ```  
  
- Define the methods that we will use to handle the operations `Create`,`Delete`,`Update`
  which are executed when a k8s resource is created, deleted or updated

  ```go
  package handler

  // Handler interface contains the methods that are required
  type Handler interface {
  	Init() error
  	ObjectCreated(obj interface{})
  	ObjectDeleted(obj interface{})
  	ObjectUpdated(objOld, objNew interface{})
  }
  ```  
  
- Define a `SimpleHandler struct{}`
  ```go
  // SimpleHandler is a sample implementation of Handler
  type SimpleHandler struct{}
  ```  
  
- Add first an `init() function` which is executed during the creation of the object

  ```go
  import (
  	log "github.com/Sirupsen/logrus"
  )	
  ...
  
  // Init handles any handler initialization
  func (t *SimpleHandler) Init() error {
  	log.Info("SimpleHandler.Init")
  	return nil
  }
  ```  
  
- Develop the logic of the 3 operations where we will log the pod created and log a message when a pod is deleted or updated  
  ```go
  import (  
      core_v1 "k8s.io/api/core/v1"
  )
  // Init handles any handler initialization
  func (t *SimpleHandler) Init() error {
  	log.Info("SimpleHandler.Init")
  	return nil
  }
  
  // ObjectCreated is called when an object is created
  func (t *SimpleHandler) ObjectCreated(obj interface{}) {
  	log.Info("SimpleHandler.ObjectCreated")
  	// assert the type to a Pod object to pull out relevant data
  	pod := obj.(*core_v1.Pod)
  	log.Infof("    ResourceVersion: %s", pod.ObjectMeta.ResourceVersion)
  	log.Infof("    NodeName: %s", pod.Spec.NodeName)
  	log.Infof("    Phase: %s", pod.Status.Phase)
  }
  
  // ObjectDeleted is called when an object is deleted
  func (t *SimpleHandler) ObjectDeleted(obj interface{}) {
  	log.Info("SimpleHandler.ObjectDeleted")
  }
  
  // ObjectUpdated is called when an object is updated
  func (t *SimpleHandler) ObjectUpdated(objOld, objNew interface{}) {
  	log.Info("SimpleHandler.ObjectUpdated")
  }
  ```  
  
- We will now finish to develop the code of the controller to :
  - Launch a loop to wait messages
  - Run the informer listening to the k8s API for pods' `list` or `watch` 
  - Start to process to get messages published on the queue  
  - Call the different handler operations if a key exists within the index cache 
  
  ```go
  import (
  	"fmt"
  	"time"
  	log "github.com/Sirupsen/logrus"
  	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
  	"k8s.io/apimachinery/pkg/util/wait"
  	"k8s.io/client-go/kubernetes"
  	"k8s.io/client-go/tools/cache"
  	"k8s.io/client-go/util/workqueue"
  )
  ...
  // Run is the main path of execution for the controller loop
  func (c *Controller) Run(stopCh <-chan struct{}) {
  	// handle a panic with logging and exiting
  	defer utilruntime.HandleCrash()
  	// ignore new items in the queue but when all goroutines
  	// have completed existing items then shutdown
  	defer c.queue.ShutDown()
  
  	c.logger.Info("Controller.Run: initiating")
  
  	// run the informer to start listing and watching resources
  	go c.informer.Run(stopCh)
  
  	// do the initial synchronization (one time) to populate resources
  	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
  		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
  		return
  	}
  	c.logger.Info("Controller.Run: cache sync complete")
  
  	// run the runWorker method every second with a stop channel
  	wait.Until(c.runWorker, time.Second, stopCh)
  }
  
  // HasSynced allows us to satisfy the Controller interface
  // by wiring up the informer's HasSynced method to it
  func (c *Controller) HasSynced() bool {
  	return c.informer.HasSynced()
  }
  
  // runWorker executes the loop to process new items added to the queue
  func (c *Controller) runWorker() {
  	log.Info("Controller.runWorker: starting")
  
  	// invoke processNextItem to fetch and consume the next change
  	// to a watched or listed resource
  	for c.processNextItem() {
  		log.Info("Controller.runWorker: processing next item")
  	}
  
  	log.Info("Controller.runWorker: completed")
  }
  
  // processNextItem retrieves each queued item and takes the
  // necessary handler action based off of if the item was
  // created or deleted
  func (c *Controller) processNextItem() bool {
  	log.Info("Controller.processNextItem: start")
  
  	// fetch the next item (blocking) from the queue to process or
  	// if a shutdown is requested then return out of this to stop
  	// processing
  	key, quit := c.queue.Get()
  
  	// stop the worker loop from running as this indicates we
  	// have sent a shutdown message that the queue has indicated
  	// from the Get method
  	if quit {
  		return false
  	}
  
  	defer c.queue.Done(key)
  
  	// assert the string out of the key (format `namespace/name`)
  	keyRaw := key.(string)
  
  	// take the string key and get the object out of the indexer
  	//
  	// item will contain the complex object for the resource and
  	// exists is a bool that'll indicate whether or not the
  	// resource was created (true) or deleted (false)
  	//
  	// if there is an error in getting the key from the index
  	// then we want to retry this particular queue key a certain
  	// number of times (5 here) before we forget the queue key
  	// and throw an error
  	item, exists, err := c.informer.GetIndexer().GetByKey(keyRaw)
  	if err != nil {
  		if c.queue.NumRequeues(key) < 5 {
  			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", key, err)
  			c.queue.AddRateLimited(key)
  		} else {
  			c.logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", key, err)
  			c.queue.Forget(key)
  			utilruntime.HandleError(err)
  		}
  	}
  
  	// if the item doesn't exist then it was deleted and we need to fire off the handler's
  	// ObjectDeleted method. but if the object does exist that indicates that the object
  	// was created (or updated) so run the ObjectCreated method
  	//
  	// after both instances, we want to forget the key from the queue, as this indicates
  	// a code path of successful queue key processing
  	if !exists {
  		c.logger.Infof("Controller.processNextItem: object deleted detected: %s", keyRaw)
  		c.handler.ObjectDeleted(item)
  		c.queue.Forget(key)
  	} else {
  		c.logger.Infof("Controller.processNextItem: object created detected: %s", keyRaw)
  		c.handler.ObjectCreated(item)
  		c.queue.Forget(key)
  	}
  
  	// keep the worker loop running by returning true
  	return true
  }
  ``` 
  
- Now, that everything is in place, we can revisit our `main.go` file to :
  - Register the `informer`, `workingqueue`
  - Create a `Controller` object
  - Start the `Controller loop`
  
  ```go
  // Register the informer, working queue and events
  informer := util.GetPodsSharedIndexInformer(client)
  queue := util.CreateWorkingQueue()
  util.AddPodsEventHandler(informer, queue)
  
  // construct the Controller object which has all of the necessary components to
  // handle logging, connections, informing (listing and watching), the queue,
  // and the handler
  controller := controller.Controller{
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
  ```
  
- Add new dependencies to the `Gopkgo.toml` and re-run `dep ensure`

  ```toml
  [[constraint]]
    branch = "master"
    name = "github.com/hashicorp/golang-lru"
  ```  
  
- Run the application locally and you will get infos about pods created within the `default` namespace

  ```bash
  go run main.go -kubeconfig=$HOME/.kube/config
  
  INFO[0000] Controller.Run: initiating                   
  INFO[0000] Add pod: default/router-1-hmrss              
  INFO[0000] Add pod: default/docker-registry-1-ld6rh     
  INFO[0000] Add pod: default/persistent-volume-setup-kmkfh 
  INFO[0000] Controller.Run: cache sync complete          
  INFO[0000] Controller.runWorker: starting               
  INFO[0000] Controller.processNextItem: start            
  INFO[0000] Controller.processNextItem: object created detected: default/router-1-hmrss 
  INFO[0000] SimpleHandler.ObjectCreated                  
  INFO[0000]     ResourceVersion: 64850                   
  INFO[0000]     NodeName: localhost                      
  INFO[0000]     Phase: Running                           
  INFO[0000] Controller.runWorker: processing next item   
  INFO[0000] Controller.processNextItem: start            
  INFO[0000] Controller.processNextItem: object created detected: default/docker-registry-1-ld6rh 
  INFO[0000] SimpleHandler.ObjectCreated                  
  INFO[0000]     ResourceVersion: 64838                   
  INFO[0000]     NodeName: localhost                      
  INFO[0000]     Phase: Running                           
  INFO[0000] Controller.runWorker: processing next item   
  INFO[0000] Controller.processNextItem: start            
  INFO[0000] Controller.processNextItem: object created detected: default/persistent-volume-setup-kmkfh 
  INFO[0000] SimpleHandler.ObjectCreated                  
  INFO[0000]     ResourceVersion: 1435                    
  INFO[0000]     NodeName: localhost                      
  INFO[0000]     Phase: Succeeded                         
  ```   