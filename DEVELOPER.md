# Play with k8s Controller and extend the API with a CustomResourceDefinition

## Table of Content

  * [Interesting reading](#interesting-reading)
  * [Prerequisites](#prerequisites)
  * [Create a golang project](#create-a-golang-project)
  * [Setup a k8s client to communicate with the platform](#setup-a-k8s-client-to-communicate-with-the-platform)
  * [Design a simple controller watching pods](#design-a-simple-controller-watching-pods)
  * [Developing a CustomResourceDefinition's team](#developing-a-customresourcedefinitions-team)
  * [Develop a TeamController to play with the new resource](#develop-a-teamcontroller-to-play-with-the-new-resource)
  * [Use Operator SDK](#use-operator-sdk)
  
## Interesting readings

- [Doc: Extend Kubernetes](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/)
- [Doc: Extending the Kubernetes API with custom resources](https://docs.openshift.com/container-platform/3.9/admin_guide/custom_resource_definitions.html)

- [Tech-in-depth: informer, reflector, store](http://borismattijssen.github.io/articles/kubernetes-informers-controllers-reflectors-stores)
- [Tech-in-depth: architecture of the components](https://medium.com/@cloudark/kubernetes-custom-controllers-b6c7d0668fdf)

- [Blog: kubernetes crd client](https://www.martin-helmich.de/en/blog/kubernetes-crd-client.html)
- [Blog: Create custom resource](https://medium.com/@trstringer/create-kubernetes-controllers-for-core-and-custom-resources-62fc35ad64a3)
- [Blog: Deep dive Code generation of CRD](https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/)

## Prerequisites

- Go Lang : [>=1.9](https://golang.org/doc/install)
- [Dep tool](https://github.com/golang/dep)
- [GOWORKSPACE](https://golang.org/doc/code.html#Workspaces) variable defined 

## Create a golang project

- Move to your `$GOPATH` directory and create under `src/github.com/$USER` a new project

  ```bash
  export USER="cmoulliard"
  export PROJECT="k8s-controller-crd"
  cd $GOPATH/src/github.com && mkdir -p $USER/$PROJECT
  cd $USER/$PROJECT
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
  go run main.go -kubeconfig=$HOME/.kube/config
  
  INFO[0000] Successfully constructed k8s client          
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
  
## Design a simple controller watching pods

- Create a `proxy.go` file under the folder `pkg/util` where we will create the 
  different objects such as the `informer`, the `event handler` and the `queue`
  used by the controller to be informed about the `create,update,delete` events
   ```bash
   mkdir -p pkg/util && touch pkg/util/proxy.go
   ```
   
- Define a function to return `cache.NewSharedIndexInformer` to watch or list the `pods` published within the `default` namespace
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
  
  func GetPodsSharedIndexInformer(client kubernetes.Interface) cache.SharedIndexInformer {
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
  import (  
	log "github.com/Sirupsen/logrus"
  )
  func AddPodsEventHandler(inf cache.SharedInformer, queue workqueue.RateLimitingInterface) {
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
  				queue.Add(key)
  			}
  		},
  		UpdateFunc: func(oldObj, newObj interface{}) {
  			key, err := cache.MetaNamespaceKeyFunc(newObj)
  			log.Infof("Update pod: %s", key)
  			if err == nil {
  				queue.Add(key)
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
 
- Create the `Simple.go` controller within the folder `pkg/controller/` 
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
   // Queueing, and handling of resource changes
   type Controller struct {
   	Logger    *log.Entry
   	Clientset kubernetes.Interface
   	Queue     workqueue.RateLimitingInterface
   	Informer  cache.SharedIndexInformer
   	Handler   handler.SimpleHandler
   }
   ```
 
- Create the `Simple.go` Handler file within the folder `pkg/handler/` 
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
  ...
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
  	// ignore new items in the Queue but when all goroutines
  	// have completed existing items then shutdown
  	defer c.Queue.ShutDown()
  
  	c.Logger.Info("Controller.Run: initiating")
  
  	// run the informer to start listing and watching resources
  	go c.Informer.Run(stopCh)
  
  	// do the initial synchronization (one time) to populate resources
  	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
  		utilruntime.HandleError(fmt.Errorf("Error syncing cache"))
  		return
  	}
  	c.Logger.Info("Controller.Run: cache sync complete")
  
  	// run the runWorker method every second with a stop channel
  	wait.Until(c.runWorker, time.Second, stopCh)
  }
  
  // HasSynced allows us to satisfy the Controller interface
  // by wiring up the informer's HasSynced method to it
  func (c *Controller) HasSynced() bool {
  	return c.Informer.HasSynced()
  }
  
  // runWorker executes the loop to process new items added to the Queue
  func (c *Controller) runWorker() {
  	log.Info("Controller.runWorker: starting")
  
  	// invoke processNextItem to fetch and consume the next change
  	// to a watched or listed resource
  	for c.processNextItem() {
  		log.Info("Controller.runWorker: processing next item")
  	}
  
  	log.Info("Controller.runWorker: completed")
  }
  
  // processNextItem retrieves each Queued item and takes the
  // necessary handler action based off of if the item was
  // created or deleted
  func (c *Controller) processNextItem() bool {
  	log.Info("Controller.processNextItem: start")
  
  	// fetch the next item (blocking) from the Queue to process or
  	// if a shutdown is requested then return out of this to stop
  	// processing
  	key, quit := c.Queue.Get()
  
  	// stop the worker loop from running as this indicates we
  	// have sent a shutdown message that the Queue has indicated
  	// from the Get method
  	if quit {
  		return false
  	}
  
  	defer c.Queue.Done(key)
  
  	// assert the string out of the key (format `namespace/name`)
  	keyRaw := key.(string)
  
  	// take the string key and get the object out of the indexer
  	//
  	// item will contain the complex object for the resource and
  	// exists is a bool that'll indicate whether or not the
  	// resource was created (true) or deleted (false)
  	//
  	// if there is an error in getting the key from the index
  	// then we want to retry this particular Queue key a certain
  	// number of times (5 here) before we forget the Queue key
  	// and throw an error
  	item, exists, err := c.Informer.GetIndexer().GetByKey(keyRaw)
  	if err != nil {
  		if c.Queue.NumRequeues(key) < 5 {
  			c.Logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", key, err)
  			c.Queue.AddRateLimited(key)
  		} else {
  			c.Logger.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", key, err)
  			c.Queue.Forget(key)
  			utilruntime.HandleError(err)
  		}
  	}
  
  	// if the item doesn't exist then it was deleted and we need to fire off the handler's
  	// ObjectDeleted method. but if the object does exist that indicates that the object
  	// was created (or updated) so run the ObjectCreated method
  	//
  	// after both instances, we want to forget the key from the Queue, as this indicates
  	// a code path of successful Queue key processing
  	if !exists {
  		c.Logger.Infof("Controller.processNextItem: object deleted detected: %s", keyRaw)
  		c.Handler.ObjectDeleted(item)
  		c.Queue.Forget(key)
  	} else {
  		c.Logger.Infof("Controller.processNextItem: object created detected: %s", keyRaw)
  		c.Handler.ObjectCreated(item)
  		c.Queue.Forget(key)
  	}
  
  	// keep the worker loop running by returning true
  	return true
  }
  ``` 
  
- Now, that everything is in place, we can revisit our `main.go` file to :
  - Register the `informer`, `workingqueue`
  - Create a `Controller` object
  - Start the `Controller loop`

- Duplicate the `main.go` file and rename it `main.go`. (`$ cp main.go main2.go`)  
  
  ```go
  import (
  	log "github.com/Sirupsen/logrus"
  
  	"github.com/cmoulliard/$PROJECT/pkg/client"
  	controllers "github.com/cmoulliard/$PROJECT/pkg/controller"
  	"github.com/cmoulliard/$PROJECT/pkg/handler"
  	"github.com/cmoulliard/$PROJECT/pkg/util"
  
  	"os"
  	"os/signal"
  	"syscall"
  )
  ...
  // Register the informer, working queue and events
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
  
## Developing a CustomResourceDefinition's team

- The first step in defining the custom resource is to figure out the following…
  
  The API group name — in my case I’ll use `cmoulliard.com` but this can be whatever you want
  The version — I’ll use `v1` for this custom resource but you are welcome to use any that you like. Some common ones are `v1`, `v1beta2`, `v2alpha1`
  Resource name — how your resource will be individually identified. For my example I’ll use the resource name `Team`
  
- Before we create the resource and necessary items, let’s first create the directory structure: `$ mkdir -p pkg/apis/team/v1`.  

- Create the API group name const in a new file: `$ touch pkg/apis/team/register.go`

  ```go
  package team

  const GroupName = "cmoulliard.com"
  ```
  
- Create the type structs which defines our `Team` resource: `$ touch pkg/apis/team/v1/types.go`  

  ```go
  package v1
  
  import (
  	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  )
  
  // TeamSpec defines the desired state of Team
  type TeamSpec struct {
  	Name         string `json:"name"`
  	Description  string `json:"description"`
  	Size         int    `json:"size,omitempty"`
  }
  
  // TeamStatus defines the observed state of Team
  type TeamStatus struct {
  	State   string `json:"state,omitempty"`
  	Message string `json:"message,omitempty"`
  }
  
  // +genclient
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
  
  // Team
  // +k8s:openapi-gen=true
  // +kubebuilder:resource:path=teams
  type Team struct {
  	meta_v1.TypeMeta   `json:",inline"`
  	meta_v1.ObjectMeta `json:"metadata,omitempty"`
  
  	Spec   TeamSpec   `json:"spec,omitempty"`
  	Status TeamStatus `json:"status,omitempty"`
  }
  ```

**Remark** : The `+<tag_name>[=value]` are `indicators` for the `code generator` tool that direct specific behavior for code generation.
  
**Impirtant** :   
  - `+genclient` : generate a client package containing the code able to discover with the `clientset` the new type of the API
  - `+genclient:noStatus` : when generating the client, there is no status stored for the package
  - `+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object` : generate `deepcopy logic` which is required and implements the `runtime.Object` interface (this is for both Team and TeamList)  
  

- Create a doc source file for the package: `$ touch pkg/apis/myresource/v1/doc.go` with these `tags`
  ```go
  // +k8s:deepcopy-gen=package
  // +groupName=cmoulliard.com
  
  package v1
  ```  
  
**Important** : Global tags are written into the `doc.go` file.   
**Remark**: 
  - `+k8s:deepcopy-gen=package` tag tells to create `deepcopy` methods by default for every type in that package.
  - `+groupName=cmoulliard.com` tag defines the `fully qualified API` group name.  
 
- The client requires a particular API surface area for custom types, and the package needs to include `AddToScheme and Resource`.
  These functions handle adding `types` to the `schemes`. Create the source file for this functionality in the package: `$ touch pkg/apis/team/v1/register.go` 
  and register the new Types to the Scheme as described hereafter
  
  ```go
  package v1
  
  import (
  	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  	"k8s.io/apimachinery/pkg/runtime"
  	"k8s.io/apimachinery/pkg/runtime/schema"
  
  	team "github.com/cmoulliard/$PROJECT/pkg/apis/team"
  )
  
  // GroupVersion is the identifier for the API which includes
  // the name of the group and the version of the API
  var SchemeGroupVersion = schema.GroupVersion{
  	Group:   team.GroupName,
  	Version: "v1",
  }
  
  // create a SchemeBuilder which uses functions to add types to
  // the scheme
  var (
       SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
       AddToScheme   = SchemeBuilder.AddToScheme
  )
  
  func Resource(resource string) schema.GroupResource {
  	return SchemeGroupVersion.WithResource(resource).GroupResource()
  }
  
  // addKnownTypes adds our types to the API scheme by registering
  // Team and TeamList
  func addKnownTypes(scheme *runtime.Scheme) error {
  	scheme.AddKnownTypes(
  		SchemeGroupVersion,
  		&Team{},
  		&TeamList{},
  	)
  
  	// register the type in the scheme
  	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
  	return nil
  }
  ```
  
- Generate the code using the k8s `code-generator` tool to :
  - Create `DeepCopy` functions
  - Create Team's `informers`, `clientset` and `listers`

```bash
CURRENT=$(pwd)
ROOT_PACKAGE="github.com/$USER/$PROJECT"
CUSTOM_RESOURCE_NAME="team"
CUSTOM_RESOURCE_VERSION="v1"

# retrieve the code-generator scripts and bins
go get -u k8s.io/code-generator/...
CODE_GENERATOR="$GOPATH/src/k8s.io/code-generator"

# run the code-generator entrypoint script
$CODE_GENERATOR/generate-groups.sh all "$ROOT_PACKAGE/pkg/client" "$ROOT_PACKAGE/pkg/apis" "$CUSTOM_RESOURCE_NAME:$CUSTOM_RESOURCE_VERSION"
Generating deepcopy funcs
Generating clientset for team:v1 at github.com/cmoulliard/k8s-team-crd/pkg/client/clientset
Generating listers for team:v1 at github.com/cmoulliard/k8s-team-crd/pkg/client/listers
Generating informers for team:v1 at github.com/cmoulliard/k8s-team-crd/pkg/client/informers
```  

## Develop a TeamController to play with the new resource

- In order to be able to create a new controller, we must first revisit the `kube.go` file to return the team's clientset which is is needed by the `informer`
- Enhance `pkg/client/kube.go` to define a new function `GetKubernetesCRDClient` which will return the `clientset` able to deal with  our type and extending the k8s API

  ```go
  ...
	teamclientset "github.com/$USER/$PROJECT/pkg/client/clientset/versioned"
   )
   
   // Retrieve the Kubernetes cluster client from outside of the cluster and add the Team Clienset
   func GetKubernetesCRDClient() (kubernetes.Interface, teamclientset.Interface) {
   	// Generate the client based off of the config
   	client := GetKubernetesClient()
   
   	// Create a Team ClientSet
   	clientset, err := teamclientset.NewForConfig(config)
   	if err != nil {
   		log.Fatalf("Team clienset: %v", err)
   	}
   
   	log.Info("Successfully constructed k8s client")
   	return client, clientset
   }
  ```

- Add a new function `GetTeamsSharedIndexInformer` within the `pkg/util/proxy.go` file to return the Team's Informer where we will scan all the `Namespaces`. 

  ```go
  package util
  
  import (
  	log "github.com/Sirupsen/logrus"
  	api_v1 "k8s.io/api/core/v1"
  	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  	"k8s.io/apimachinery/pkg/runtime"
  	"k8s.io/apimachinery/pkg/watch"
  	"k8s.io/client-go/kubernetes"
  	"k8s.io/client-go/tools/cache"
  	"k8s.io/client-go/util/workqueue"
  
  	teamclientset "github.com/$USER/$PROJECT/pkg/client/clientset/versioned"
  	teaminformer_v1 "github.com/$USER/$PROJECT/pkg/client/informers/externalversions/team/v1"
  )
  
  
  func GetTeamsSharedIndexInformer(client kubernetes.Interface, teamclient teamclientset.Interface ) cache.SharedIndexInformer {
  	return teaminformer_v1.NewTeamInformer(
  		teamclient,
  		meta_v1.NamespaceAll,
  		0,
  		cache.Indexers{},
  	)
  }
  ...
  ```
  
- Revisit `handle.go` in order to process our `team` resource instead of the `pod`. Duplicate the `pkg/handler/simple.go` file and rename it `team.go`. (`$ cp pkg/handler/simple.go pkg/handler/team.go`)
- Rename all the occurrences of `SimpleHandler` to `TeamHandler`
- Rename the Handler Interface to `type HandlerInterface interface {` 
- Modify the function `func (t *TeamHandler) ObjectCreated(obj interface{}) {` to log the information of our team type`
  ```go
  import (
  	log "github.com/Sirupsen/logrus"
  	team_v1 "github.com/$USER/$PROJECT/pkg/apis/team/v1"
  )
  func (t *TeamHandler) ObjectCreated(obj interface{}) {
  	log.Info("TeamHandler.ObjectCreated")
  
  	team := obj.(*team_v1.Team)
  	log.Infof("    ResourceVersion: %s", team.ObjectMeta.ResourceVersion)
  	log.Infof("    Team name: %s", team.Spec.Name)
  	log.Infof("    Team description: %s", team.Spec.Description)
  	log.Infof("    Team size: %s", team.Spec.Size)
  }
  ```

- Create a new controller able to handle the events for our team's resource. Duplicate the `pkg/controller/simple.go` file and rename it `team.go`. (`$ cp pkg/controller/simple.go pkg/controller/team.go`)
- Rename `Controller` to `TeamController`. See snippet example hereafter
  ```go
  ...
  type TeamController struct {
  	Logger    *log.Entry
  	Clientset kubernetes.Interface
  	Queue     workqueue.RateLimitingInterface
  	Informer  cache.SharedIndexInformer
  	Handler   handler.TeamHandler
  }
  
  // Run is the main path of execution for the controller loop
  func (c *TeamController) Run(stopCh <-chan struct{}) {
  ...
  ```
- Adapt the `Controller` struct to use our team handler
  ```go
  type Controller struct {
  	Logger    *log.Entry
  	Clientset kubernetes.Interface
  	Queue     workqueue.RateLimitingInterface
  	Informer  cache.SharedIndexInformer
  	Handler   handler.TeamHandler
  }
  ```

- Duplicate the `main2.go` file and rename it `main3.go`. (`$ cp main2.go main3.go`)
- Modify the functions to use our Team's `clientset` and `informer`
  ```go
  func main() {
  	// Get the Kubernetes client to access the Cloud platform
  	client := client.GetKubernetesClient()
  
  	informer := util.GetPodsSharedIndexInformer(client)
  ```
  
- Update the controller to use the `TeamController`
  ```go
  import (
  	controllers "github.com/$USER/$PROJECT/pkg/controller"
  )
  ...
  controller := controllers.TeamController{
  	Logger:    log.NewEntry(log.New()),
  	Clientset: client,
  	Informer:  teaminformer,
  	Queue:     queue,
  	Handler:   handler.TeamHandler{},
  }
  ```  
  
- Create a `customresourcedefinition` yaml resource file within the folder `$ mkdir -p example`. The name of the file is team-crd.yml (`$ touch example/team-crd.yml`)

  ```yaml
  apiVersion: apiextensions.k8s.io/v1beta1
  kind: CustomResourceDefinition
  metadata:
    name: teams.cmoulliard.com
  spec:
    group: cmoulliard.com
    version: v1
    names:
      kind: Team
      listKind: TeamList
      plural: teams
      shortNames:
      - team
      singular: team
    version: v1
  ```

- Install it on k8s / OpenShift

  ```bash
  oc create -f example/team-crd.yml
  customresourcedefinition "teams.cmoulliard.com" created
  ```   
  
- Create a `team` resource yaml file (`$ touch example/team1.yml`) with these informations

  ```yaml
  apiVersion: cmoulliard.com/v1
  kind: Team
  metadata:
    labels:
      project: ux
    name: team123
    namespace: my-crd
  spec:
    description: Awesome Snowdrop Team !
    name: Spring Boot Team
    size: 4
  ```

- Create a new `Team` resource on k8s / OpenShift

  ```bash
  oc create -f example/team1.yml
  team "team123" created
  ```
  
- Run the new application consuming this CRD type and you will see `INFO` messages about the object created !

  ```bash
  go run main3.go
  INFO[0000] Successfully constructed k8s client          
  INFO[0000] Successfully constructed k8s client          
  INFO[0000] Controller.Run: initiating                   
  INFO[0000] Add pod: my-crd/team123                      
  INFO[0000] Controller.Run: cache sync complete          
  INFO[0000] Controller.runWorker: starting               
  INFO[0000] Controller.processNextItem: start            
  INFO[0000] Controller.processNextItem: object created detected: my-crd/team123 
  INFO[0000] TeamHandler.ObjectCreated                    
  INFO[0000]     ResourceVersion: 185897                  
  INFO[0000]     Team name: Spring Boot Team              
  INFO[0000]     Team description: Awesome Snowdrop Team ! 
  INFO[0000]     Team size: %!s(int=4)                    
  INFO[0000] Controller.runWorker: processing next item   
  INFO[0000] Controller.processNextItem: start     
  ```  
  
## Use Operator SDK

Use Operator SDK to generate for a type, the skeleton of a go project where you will just develop the `business logic` to be handled for some specific resources, the resources
to be watched and that's all !

![Operator SDK architecture](https://cdn-images-1.medium.com/max/1600/1*f0fp4e7RhoKSzC5N7UTxSw.jpeg)

- Get SDK Operator project and compile the `operator-sdk` client using these commands
```bash
$ cd $GOPATH/src
$ go get -u github.com/operator-framework/operator-sdk/... 
$ make dep
$ make install
```

- Create an `team-operator` project that defines the `Team`'s CR.

```bash
$ cd $GOPATH/src/github.com/
$ mkdir -p snowdrop
$ operator-sdk new team-operator --api-version=team.snowdrop.me/v1 --kind=Team

$ cd team-operator
```

- During the creation of the skeleton of the project, the following folders will be created :
  - `deploy`: k8s resources to install the project
  - `cmd` : `main.go` where SDK is configured to watch resources and initiate the logic using `Handler` 
  - `pkg` : `apis/resource/version` definition about `types`, `doc`, `register` and `stub`'s class such as `handler`
  - `tmp` : bash script to generate `deepcopy` functions of the type and `Dockerfile` to build the `operator` and to compile the `operator`
  - `version` : operator version
  - `vendor` : go packages/dependencies needed : `client-go`, `apimachinery`, `code-genarator`, ...
  
- The most important files are : `cmd/$PROJECT/main.go`, `pkg/apis/TYPE/VERSION/types.go` and `pkg/stub/handle.go` as they will take care about the following 
  logic
  
  **Main Applicationcontrol**
  
  ```go
  resource := "team.snowdrop.me/v1"
  kind := "Team"
  namespace, err := k8sutil.GetWatchNamespace()
  resyncPeriod := 5
  
  sdk.Watch(resource, kind, namespace, resyncPeriod) // WATCH TO WATCH, OCCURENCE, NAMESPACE
  sdk.Handle(stub.NewHandler()) // REGISTER LOGIC
  sdk.Run(context.TODO()) // START THE CONTROLLER -> INFORMER, WORKING QUEUE and LOOP TO WAIT EVENTS
  ```

  **Type structure, fields, values**

  ```go
  package v1
  
  import (
  	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  )
  
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
  
  type TeamList struct {
  	metav1.TypeMeta `json:",inline"`
  	metav1.ListMeta `json:"metadata"`
  	Items           []Team `json:"items"`
  }
  
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
  
  type Team struct {
  	metav1.TypeMeta   `json:",inline"`
  	metav1.ObjectMeta `json:"metadata"`
  	Spec              TeamSpec   `json:"spec"`
  	Status            TeamStatus `json:"status,omitempty"`
  }
  ...
  ```

  and **Business logic to handle events**
  
  ```go
  func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
  	switch o := event.Object.(type) {
  	case *v1.Team:  // EVENT's TYPE TO WATCH
	 	if ! event.Deleted { // IF OBJECT HAS BEEN CREATED OR DELETED
  		   err := sdk.Create(newbusyBoxPod(o)) // WHAT TO DO 
  		   if err != nil && !errors.IsAlreadyExists(err) {
  		   	logrus.Errorf("Failed to create busybox pod : %v", err)
  		   	return err
  		   }
		}   
  	}
  	return nil
  }
  ```
  
- Build and push the `team-operator` image to a public registry such as `quay.io`

**Remarks** :
 - Create first on `quai.io` the repository `team-operator` under your `org` and log on to `quai.io`
 - A local docker daemon running on your desktop is needed (E.g. `$ eval $(minishift docker-env)`)
 
```bash
$ operator-sdk build quay.io/snowdrop/team-operator
$ docker images | grep quay.io/snowdrop/team-operator
$ quay.io/snowdrop/team-operator                     latest              bc4dc0cfc9b3        About a minute ago   39.8MB

$ docker login quay.io -u $USERNAME -p $PASSWORD
$ docker push quay.io/snowdrop/team-operator
```

- Deploy the `team-operator`

```bash
$ oc new-project team-operator
$ oc create -f deploy/rbac.yaml
$ oc create -f deploy/crd.yaml
$ oc create -f deploy/operator.yaml
```

- By default, creating a custom resource (Team) triggers the `team-operator` to deploy a busybox pod
```bash
$ oc create -f deploy/cr.yaml
```

- Verify that a `busybox pod` is created with this command
 
```bash
$ oc get pod -l app=busy-box -w
NAME            READY     STATUS    RESTARTS   AGE
busy-box   1/1       Running   0          50s
```

- Cleanup
```bash
$ oc delete -f deploy/cr.yaml
$ oc delete -f deploy/operator.yaml
$ oc delete -f deploy/rbac.yaml
```  


  