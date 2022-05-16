package main

import (
	"flag"
	"fmt"
	main2 "github.com/jeremyharisch/kyma-watcher/watcher"
	"net/http"
	"sync"
)

func main() {

	var (
		mutex        *sync.Mutex
		namespace    string
		currentEvent string
	)

	namespace = "default"
	mutex = &sync.Mutex{}

	// Check flag to see if application runs in in-cluster or local mode + create clientset
	useKubeconfig := flag.Bool("kubeconfig", false, "Use exported kubeconfig, when running inlocal mode")
	flag.Parse()
	clientset := main2.createClientset(useKubeconfig)

	// Trigger main watch mechanism
	go main2.watchForChanges(clientset, namespace, &currentEvent, mutex)

	// Handle endpoints
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		body := []byte(fmt.Sprintf(`Events on cluster: %s`, currentEvent))
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		mutex.Unlock()
	})

	// Start listener
	fmt.Printf("Listening on port 8080\n")
	http.ListenAndServe(":8080", nil)
}
