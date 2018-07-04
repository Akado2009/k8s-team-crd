# Instructions to create a CRD and Controller

- Move to your `$GOPATH` directory and create under `src/github.com/$GITHUBACCOUNT` a new project

```bash
export GITHUBACCOUNT="cmoulliard"
cd $GOPATH/src/github.com && mkdir -p $GITHUBACCOUNT/k8s-controller-demo
cd $GITHUBACCOUNT/k8s-controller-demo
```
- Move to the new project created and create the following folders's tree and files
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

- Add a `pkg/client/kube.go` file using the command `touch pkg/client/kube.go` and develop the `GetKubernetClient` function to return a k8s go client.
  
  You can pass as parameter the location of the `$HOME/.kube/config` file which contains the default context to be used to access the k8s platform

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

- Next, develop the `main.go` file which contains the instructions to :
  - Create a K8s client
  - Call the k8s APi to get by example the list of the `namespaces`

```go
package main

import (
	log "github.com/Sirupsen/logrus"

	"github.com/cmoulliard/k8s-team-crd/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// Get the Kubernetes client to access the Cloud platform
	client := client.GetKubernetesClient()

	ns, nsError := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if nsError != nil {
		log.Fatalf("Can't list namespaces ", nsError)
	}
	for i := range ns.Items {
		log.Info("Namespace/project : ", ns.Items[i].Name)
	}
}
```

- To be able execute locally this project, it is required to define the packages to be downloaded and next to import them into a local vendor directory
- Edit the `Gopkg.toml` to include the following dependencies
```toml
[[constraint]]
  name = "k8s.io/client-go"
  version = "6.0.0"

[[constraint]]
  name = "k8s.io/apimachinery"
  version = "kubernetes-1.9.9"

[[constraint]]
  name = "github.com/Sirupsen/logrus"
  version = "1.0.5"
```

- Grab the packages using [`dep tool`](https://github.com/golang/dep)
```bash
dep ensure
```