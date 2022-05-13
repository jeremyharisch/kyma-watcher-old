package main

import (
	"context"
	"flag"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func init() {
}

func main() {

	var (
		mutex        *sync.Mutex
		namespace    string
		currentEvent string
	)

	namespace = "default"

	useKubeconfig := flag.Bool("kubeconfig", false, "Use exported kubeconfig, when running inlocal mode")
	clientset := createClientset(useKubeconfig)

	mutex = &sync.Mutex{}
	go watchForChanges(clientset, namespace, &currentEvent, mutex)

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		body := []byte(fmt.Sprintf(`{"Current event: %s""}`, currentEvent))
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		mutex.Unlock()
	})
	fmt.Printf("Listening on port 8080\n")
	http.ListenAndServe(":8080", nil)
}

func createClientset(useKubeconfig *bool) *kubernetes.Clientset {
	var clientCfg *rest.Config
	var err error
	if *useKubeconfig {
		// local mode
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig != "" {
			clientCfg, err = clientcmd.BuildConfigFromKubeconfigGetter("", func() (config *clientcmdapi.Config, e error) {
				return clientcmd.Load([]byte(kubeconfig))
			})
		} else {
			panic("'KUBECONFIG' is empty or not set")
		}
	} else {
		// in-cluster mode
		clientCfg, err = rest.InClusterConfig()
		if err == rest.ErrNotInCluster {
			panic("Application not running inside of K8s cluster")
		} else if err != nil {
			panic(fmt.Sprintf("Unable to get our client configuration: %s", err))
		}
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		panic("Unable to create our clientset")
	}
	return clientset
}
func watchForChanges(clientset *kubernetes.Clientset, namespace string, currentEvent *string, mutex *sync.Mutex) {
	for {
		watcher, err := clientset.CoreV1().ConfigMaps(namespace).Watch(
			context.TODO(),
			metav1.SingleObject(metav1.ObjectMeta{
				Namespace: namespace}))
		if err != nil {
			panic("Unable to create watcher")
		}
		detectChanges(watcher.ResultChan(), currentEvent, mutex)
	}
}

func detectChanges(eventChannel <-chan watch.Event, currentEvent *string, mutex *sync.Mutex) {
	for {
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				updateEvent(event, currentEvent, mutex)
			case watch.Modified:
				updateEvent(event, currentEvent, mutex)
			case watch.Deleted:
				updateEvent(event, currentEvent, mutex)
			default:
				mutex.Lock()
				*currentEvent = "Default Case"
				mutex.Unlock()
			}
		} else {
			// Server closed connection
			return
		}
	}
}

func updateEvent(event watch.Event, currentEvent *string, mutex *sync.Mutex) {
	mutex.Lock()
	if updatedMap, ok := event.Object.(*corev1.ConfigMap); ok {
		if exampleData, ok := updatedMap.Data["example"]; ok {
			*currentEvent = fmt.Sprintf("CM '%s' %s: %s", updatedMap.Name, event.Type, exampleData)
		}
	}
	mutex.Unlock()
}
