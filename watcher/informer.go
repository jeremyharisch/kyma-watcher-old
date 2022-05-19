package main

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func watchForChangesInformer(clientset *kubernetes.Clientset, namespace string) {

	// Stop signal for the informer
	stopper := make(chan struct{})
	defer close(stopper)

	factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace(namespace), informers.WithTweakListOptions(tweakListOptions))
	cmInformer := factory.Core().V1().ConfigMaps()
	informer := cmInformer.Informer()

	defer runtime.HandleCrash()

	// Start informer ->
	go factory.Start(stopper)

	// Start to sync
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd, // register add eventhandler
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	})

	<-stopper

}

func tweakListOptions(opt *v1.ListOptions) {
	//TODO
}

func onAdd(obj interface{}) {
	// TODO
	fmt.Printf("\nCM '%s' %s", obj.(*corev1.ConfigMap).Name, "ADDED")
}

func onUpdate(oldObj interface{}, newObj interface{}) {
	// TODO
	fmt.Printf("\nCM '%s' %s", oldObj.(*corev1.ConfigMap).Name, "MODIFIED")
}

func onDelete(obj interface{}) {
	// TODO
	fmt.Printf("\nCM '%s' %s", obj.(*corev1.ConfigMap).Name, "DELETED")
}
