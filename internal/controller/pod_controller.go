/*
Copyright 2026.

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

package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "rishab.io/scavenger/api/scavenger/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update
// +kubebuilder:rbac:groups=scavenger.rishab.io,resources=scavengerconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile

func getResourceType(resourceType string) (client.ObjectList, error) {
	switch resourceType {
	case "pod", "pods", "Pod":
		return &corev1.PodList{}, nil
	case "service", "services", "Service":
		return &corev1.ServiceList{}, nil
	case "configmap", "configmaps", "ConfigMap":
		return &corev1.ConfigMapList{}, nil
	case "secret", "secrets", "Secret":
		return &corev1.SecretList{}, nil
	case "namespace", "namespaces", "Namespace":
		return &corev1.NamespaceList{}, nil
	case "deployment", "deployments", "Deployment":
		return &appsv1.DeploymentList{}, nil
	default:
		return nil, fmt.Errorf("unsupported core resource type: %s", resourceType)
	}
}

func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	// var pods corev1.PodList

	configKey := client.ObjectKey{
		Namespace: "default",
		Name:      "scavengerconfig",
	}

	scavengerConfig := v1.ScavengerConfig{}

	if err := r.Get(ctx, configKey, &scavengerConfig); err != nil {
		log.Error(err, "unable to get the Scavenger Configuration")
	}

	fmt.Println(scavengerConfig)

	namespaces := scavengerConfig.Spec.Namespaces

	for _, namespace := range namespaces {

		for _, resource := range scavengerConfig.Spec.Resources {

			element, _ := getResourceType(resource)

			if err := r.List(ctx, element, client.InNamespace(namespace)); err != nil {
				log.Error(err, "unable to list pods")
			}

			meta.EachListItem(element, func(obj runtime.Object) error {
				item, ok := obj.(client.Object)
				if !ok {
					return fmt.Errorf("expected client.Object, got %T", obj)
				}

				if annotation, ttlExists := item.GetAnnotations()["ttl"]; ttlExists {

					fmt.Println("TTL", annotation)
					fmt.Println("Creation Time", item.GetCreationTimestamp().Time)

					ttl, _ := strconv.ParseInt(annotation, 10, 64)
					creationTime := item.GetCreationTimestamp().Time
					duration := time.Since(creationTime)

					fmt.Println("Duration", int64(duration.Seconds()))
					fmt.Println("Time now", time.Now())

					graceperiod := client.GracePeriodSeconds(5)

					if ttl < int64(duration.Seconds()) {
						fmt.Println("Deleting resource as its ttl has expired", resource, item.GetName())
						r.Delete(ctx, item, graceperiod)
					} else {
						fmt.Println("Not deleting...")
					}

				} else {

					graceperiod := client.GracePeriodSeconds(5)
					fmt.Println("TTL expired deleting the resource : ", resource, item.GetName())
					r.Delete(ctx, item, graceperiod)
				}

				fmt.Println("----------------------------------------------------------------------------------")

				return nil
			})
			// for _, v := range &element {

			// 	if annotation, ttlExists := v.Annotations["ttl"]; ttlExists {

			// 		fmt.Println("TTL", annotation)
			// 		fmt.Println("Creation Time", v.CreationTimestamp)

			// 		ttl, _ := strconv.ParseInt(annotation, 10, 64)
			// 		creationTime := v.CreationTimestamp.Time
			// 		duration := time.Since(creationTime)

			// 		fmt.Println("Duration", int64(duration.Seconds()))
			// 		fmt.Println("Time now", time.Now())

			// 		if ttl < int64(duration.Seconds()) {
			// 			fmt.Println("Deleting Pod as its ttl has expired")
			// 			r.Delete(ctx, &v)
			// 		} else {
			// 			fmt.Println("Not deleting...")
			// 		}

			// 	} else {

			// 		graceperiod := client.GracePeriodSeconds(5)
			// 		fmt.Println("TTL expired deleting the pod")
			// 		r.Delete(ctx, &v, graceperiod)
			// 	}

			// 	fmt.Println("----------------------------------------------------------------------------------")

			// }

			// return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil

		}

	}

	// return ctrl.Result{}, nil
	return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Named("pod").
		Complete(r)
}
