/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/jeremyharisch/kyma-watcher/pkg/config"
	"github.com/jeremyharisch/kyma-watcher/pkg/contract"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

type EventType string

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Logger  logr.Logger
	KcpIp   string
	KcpPort string
	SkrId   string
}

//+kubebuilder:rbac:groups="*",resources="*",verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the ConfigMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// logger := log.FromContext(ctx).WithName(req.NamespacedName.String())
	// Should do nothing
	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciler) CreateFunc(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	r.Logger.Info(fmt.Sprintf("Create Event: %s", e.Object.GetName()))
	_, err := r.sendRequest(e)
	if err != nil {
		r.Logger.Error(err, "Error occured while sending request")
		return
	}

}

func (r *ConfigMapReconciler) UpdateFunc(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	r.Logger.Info(fmt.Sprintf("Update Event: %s", e.ObjectNew.GetName()))
	_, err := r.sendRequest(e)
	if err != nil {
		r.Logger.Error(err, "Error occured while sending request")
		return
	}
}

func (r *ConfigMapReconciler) DeleteFunc(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	r.Logger.Info(fmt.Sprintf("Delete Event: %s", e.Object.GetName()))
	_, err := r.sendRequest(e)
	if err != nil {
		r.Logger.Error(err, "Error occured while sending request")
		return
	}
}

func (r *ConfigMapReconciler) GenericFunc(e event.GenericEvent, q workqueue.RateLimitingInterface) {
	r.Logger.Info(fmt.Sprintf("Generic Event: %s", e.Object.GetName()))
	_, err := r.sendRequest(e)
	if err != nil {
		r.Logger.Error(err, "Error occured while sending request")
		return
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Create ControllerBuilder
	controllerBuilder := ctrl.NewControllerManagedBy(mgr).For(&v1.ConfigMap{})

	// Create Dynamic InformerFactory
	c, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	informers := dynamicinformer.NewFilteredDynamicSharedInformerFactory(c, time.Minute*30, "", func(options *metav1.ListOptions) {
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"kyma-watchable": "true"}}
		options.LabelSelector = labels.Set(labelSelector.MatchLabels).String()

	})
	err = mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		informers.Start(ctx.Done())
		return nil
	}))
	if err != nil {
		return err
	}

	// Create K8s-Client
	cs, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}

	// GroupVersions to be watched
	gvs := []schema.GroupVersion{
		{
			Group:   "apiextensions.k8s.io",
			Version: "v1",
		},
		{
			Group:   "",
			Version: "v1",
		},
	}

	for _, gv := range gvs {
		resources, err := cs.ServerResourcesForGroupVersion(gv.String())
		if client.IgnoreNotFound(err) != nil {
			return err
		}

		// resources found
		if err == nil {
			dynamicInformerSet := make(map[string]*source.Informer)
			for _, resource := range resources.APIResources {
				if strings.Contains(resource.Name, "/") || !strings.Contains(resource.Verbs.String(), "watch") {
					// Skip no listable resources, i.e. nodes/proxy
					continue
				}
				r.Logger.Info(fmt.Sprintf("Resource `%s` from GroupVersion `%s` will be watched", resource.Name, gv))
				gvr := gv.WithResource(resource.Name)
				dynamicInformerSet[gvr.String()] = &source.Informer{Informer: informers.ForResource(gvr).Informer()}
			}

			for gvr, informer := range dynamicInformerSet {
				controllerBuilder = controllerBuilder.
					Watches(informer, &handler.Funcs{
						CreateFunc:  r.CreateFunc,
						UpdateFunc:  r.UpdateFunc,
						DeleteFunc:  r.DeleteFunc,
						GenericFunc: r.GenericFunc,
					})
				r.Logger.Info("initialized dynamic watching", "source", gvr)
			}
		}
	}

	return controllerBuilder.Complete(r)
}

func (r *ConfigMapReconciler) sendRequest(newEvent interface{}) (string, error) {
	var component string
	var namespace string
	var name string
	switch newEvent.(type) {
	case event.CreateEvent:
		component = r.getComponent(newEvent.(event.CreateEvent).Object)
		namespace = newEvent.(event.CreateEvent).Object.GetNamespace()
		name = newEvent.(event.CreateEvent).Object.GetName()
	case event.UpdateEvent:
		component = r.getComponent(newEvent.(event.UpdateEvent).ObjectNew)
		namespace = newEvent.(event.UpdateEvent).ObjectNew.GetNamespace()
		name = newEvent.(event.UpdateEvent).ObjectNew.GetName()
	case event.DeleteEvent:
		component = r.getComponent(newEvent.(event.DeleteEvent).Object)
		namespace = newEvent.(event.DeleteEvent).Object.GetNamespace()
		name = newEvent.(event.DeleteEvent).Object.GetName()
	case event.GenericEvent:
		component = r.getComponent(newEvent.(event.GenericEvent).Object)
		namespace = newEvent.(event.GenericEvent).Object.GetNamespace()
		name = newEvent.(event.GenericEvent).Object.GetName()
	default:
		r.Logger.Info(fmt.Sprintf("Undefined eventType: %#v", newEvent))
	}

	watcherEvent := &contract.WatcherEvent{
		SkrClusterID: r.SkrId,
		Namespace:    namespace,
		Name:         name,
	}
	postBody, _ := json.Marshal(watcherEvent)

	responseBody := bytes.NewBuffer(postBody)
	url := fmt.Sprintf("%s:%s/%s/%s/%s", r.KcpIp, r.KcpPort, config.ContractVersion, component, config.EventEndpoint)
	resp, err := http.Post(url, "application/json", responseBody)
	//Handle Error
	if err != nil {
		r.Logger.Info(fmt.Sprintf("Error POST %#v", err))
		return "", err
	}
	defer resp.Body.Close()
	//Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	sb := string(body)
	r.Logger.Info("Request to KCP successfull!")
	return sb, nil
}

func (r *ConfigMapReconciler) getComponent(object client.Object) string {
	labels := object.GetLabels()
	component, ok := labels[config.ComponentLabel]
	if ok {
		r.Logger.Info(fmt.Sprintf("Component of new Event: %s", component))
		return component
	}
	r.Logger.Info(fmt.Sprintf("Label `%s` not found", config.ComponentLabel))
	return ""
}
