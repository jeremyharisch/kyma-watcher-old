package main

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sync"
)

func watchForChangesWatcher(clientset *kubernetes.Clientset, namespace string, currentEvent *string, mutex *sync.Mutex) {
	for {
		watcher, err := clientset.CoreV1().ConfigMaps(namespace).Watch(
			context.TODO(),
			metav1.ListOptions{},
		)
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
	fmt.Printf("\n Got new event: %v\n", event)
	if updatedMap, ok := event.Object.(*corev1.ConfigMap); ok {
		*currentEvent = fmt.Sprintf("%s\nCM '%s' %s", *currentEvent, updatedMap.Name, event.Type)
	}
	mutex.Unlock()
}
