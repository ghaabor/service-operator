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
	"context"

	kapps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/ghaabor/service-operator/api/v1"
)

// WebServiceReconciler reconciles a WebService object
type WebServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.ghaabor.io,resources=webservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.ghaabor.io,resources=webservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.ghaabor.io,resources=webservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WebService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *WebServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconciling WebService")

	var webService appsv1.WebService
	if err := r.Get(ctx, req.NamespacedName, &webService); err != nil {
		// if WebService not found, delete related resources
		if apierrors.IsNotFound(err) {
			// delete childService
			var childService corev1.Service
			if err := r.Get(ctx, req.NamespacedName, &childService); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
			log.Info("deleting child service", "service", childService.Name)
			if err := r.Delete(ctx, &childService); err != nil {
				log.Error(err, "failed to delete child service", "service", childService.Name)
				return ctrl.Result{}, err
			}

			// delete deployment last
			var childDeployment kapps.Deployment
			if err := r.Get(ctx, req.NamespacedName, &childDeployment); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}

			log.Info("deleting child deployment", "deployment", childDeployment.Name)
			if err := r.Delete(ctx, &childDeployment); err != nil {
				log.Error(err, "failed to delete child deployment", "deployment", childDeployment.Name)
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var childDeployment kapps.Deployment
	if err := r.Get(ctx, req.NamespacedName, &childDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			// child deployment is not created yet, create it
			log.Info("child deployment not found, creating", "deployment", webService.Name)
			childDeployment = kapps.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      webService.Name,
					Namespace: webService.Namespace,
					Labels:    webService.Labels,
				},
				Spec: kapps.DeploymentSpec{
					Replicas: &webService.Spec.Replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": webService.Name},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": webService.Name},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  webService.Name,
									Image: webService.Spec.Image,
									Ports: []corev1.ContainerPort{{
										Name:          "http",
										ContainerPort: 80,
									}},
								},
							},
						},
					},
				},
			}

			if err := r.Create(ctx, &childDeployment); err != nil {
				log.Error(err, "failed to create child deployment", "deployment", childDeployment.Name)
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// deployment found, check if it needs to be updated
	if r.deploymentUpdateNeeded(webService, childDeployment) {
		log.Info("updating child deployment", "deployment", childDeployment.Name)
		childDeployment.Spec.Replicas = &webService.Spec.Replicas
		childDeployment.Spec.Template.Spec.Containers[0].Image = webService.Spec.Image
		if err := r.Update(ctx, &childDeployment); err != nil {
			log.Error(err, "failed to update child deployment", "deployment", childDeployment.Name)
			return ctrl.Result{}, err
		}
	}

	var childService corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &childService); err != nil {
		if apierrors.IsNotFound(err) {
			// child service is not created yet, create it
			log.Info("child service not found, creating", "service", webService.Name)
			childService = corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      webService.Name,
					Namespace: webService.Namespace,
					Labels:    webService.Labels,
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "http",
							Port:       80,
							TargetPort: intstr.FromString("http"),
						},
					},
					Selector: map[string]string{"app": webService.Name},
				},
			}

			if err := r.Create(ctx, &childService); err != nil {
				log.Error(err, "failed to create child service", "service", childService.Name)
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	log.Info("WebService successfully reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.WebService{}).
		Complete(r)
}

func (r *WebServiceReconciler) deploymentUpdateNeeded(webService appsv1.WebService, childDeployment kapps.Deployment) bool {
	if childDeployment.Spec.Replicas != nil && *childDeployment.Spec.Replicas != webService.Spec.Replicas {
		return true
	}

	if childDeployment.Spec.Template.Spec.Containers[0].Image != webService.Spec.Image {
		return true
	}

	return false
}
