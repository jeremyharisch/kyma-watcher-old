package main

import (
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"os"
	"path"
)

func readKubeconfig() string {
	kubecfgFile := os.Getenv("KUBECONFIG")
	if kubecfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(fmt.Sprintf("Could not get USerHomeDir: %s", err))
		}
		kubecfgFile = path.Join(home, ".kube", "config")
	}
	if !exists(kubecfgFile) {
		panic("Set the KUBECONFIG env var before executing")
	}
	kubecfg, err := ioutil.ReadFile(kubecfgFile)
	if err != nil {
		panic(fmt.Sprintf("Could read kubeconfig: %s", err))
	}
	return string(kubecfg)
}

func createClientset(useKubeconfig *bool) *kubernetes.Clientset {
	var clientCfg *rest.Config
	var err error
	if *useKubeconfig {
		// local mode
		fmt.Println("Watcher runs in local mode")
		kubeconfig := readKubeconfig()
		if kubeconfig != "" {
			tmpclientCfg, err := clientcmd.BuildConfigFromKubeconfigGetter("", func() (config *clientcmdapi.Config, e error) {
				return clientcmd.Load([]byte(kubeconfig))
			})
			if err != nil {
				panic(fmt.Sprintf("Could not create clientCfg: %s", err))
			}
			clientCfg = tmpclientCfg
		} else {
			panic("'KUBECONFIG' is empty or not set")
		}
	} else {
		// in-cluster mode
		fmt.Println("Watcher runs in in-cluster mode")
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

func exists(file string) bool {
	if file == "" {
		return false
	}
	stats, err := os.Stat(file)
	return !os.IsNotExist(err) && (stats != nil && !stats.IsDir())
}
