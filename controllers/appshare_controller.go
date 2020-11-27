/*


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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsharev1 "github.com/leskil/appshare-operator/api/v1"
)

// AppShareReconciler reconciles a AppShare object
type AppShareReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller to reconcile on any changes to the AppShare CRD.
func (r *AppShareReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsharev1.AppShare{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=appshare.appshare.co,resources=appshares,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=appshare.appshare.co,resources=appshares/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=appshare.appshare.co,resources=appshares/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *AppShareReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("appshare", req.NamespacedName)

	appshare := &appsharev1.AppShare{}
	err := r.Get(ctx, req.NamespacedName, appshare)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("AppShare resource not found. Assuming it has been deleted.")
			return ctrl.Result{}, err
		}
	}

	existingDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: appshare.Name, Namespace: appshare.Namespace}, existingDeployment)

	if err != nil && errors.IsNotFound(err) {
		dep := r.createDeployment(appshare)
		log.Info("Creating a new deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)

		if err != nil {
			log.Error(err, "Failed to create new deployment", "Deployment.Namespace", dep.Name, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get deployment. Check permissions.")
		return ctrl.Result{}, err
	}

	existingService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: appshare.Name, Namespace: appshare.Namespace}, existingService)

	if err != nil && errors.IsNotFound(err) {
		svc := r.createService(appshare)
		log.Info("Creating a new service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)

		if err != nil {
			log.Error(err, "Failed to create new service", "Service.Namespace", svc.Name, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get service. Check permissions.")
		return ctrl.Result{}, err
	}

	hasChanges := applyCrdChangesToDeployment(existingDeployment, appshare)
	if hasChanges {
		r.Update(ctx, existingDeployment)
	}

	return ctrl.Result{}, nil
}

func applyCrdChangesToDeployment(dep *appsv1.Deployment, crd *appsharev1.AppShare) bool {

	hasChanges := false

	if dep.Spec.Replicas != &crd.Spec.Replicas {
		dep.Spec.Replicas = &crd.Spec.Replicas
		hasChanges = true
	}

	return hasChanges
}

func (r *AppShareReconciler) createDeployment(crd *appsharev1.AppShare) *appsv1.Deployment {
	labels := getLabels(crd.Name)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crd.Name,
			Namespace: crd.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &crd.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "appshare",
						Image: "appshareco/appshare:3.3",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3000,
							Protocol:      "TCP",
						}},
						ReadinessProbe: &corev1.Probe{
							FailureThreshold: 3,
							PeriodSeconds:    10,
							SuccessThreshold: 1,
							TimeoutSeconds:   1,
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/readiness-check",
									Port:   intstr.FromInt(3000),
									Scheme: corev1.URISchemeHTTP,
								},
							},
						},
						LivenessProbe: &corev1.Probe{
							FailureThreshold: 3,
							PeriodSeconds:    10,
							SuccessThreshold: 1,
							TimeoutSeconds:   1,
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/live-check",
									Port:   intstr.FromInt(3000),
									Scheme: corev1.URISchemeHTTP,
								},
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("4000m"),
								"memory":           resource.MustParse("4096Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("500m"),
								"memory":           resource.MustParse("512Mi"),
							},
						},
					},
					}},
			},
		},
	}

	ctrl.SetControllerReference(crd, dep, r.Scheme)
	return dep
}

func (r *AppShareReconciler) createService(crd *appsharev1.AppShare) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crd.Name,
			Namespace: crd.Namespace,
			Labels:    getLabels(crd.Name),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       3000,
				TargetPort: intstr.FromInt(3000),
				Protocol:   "TCP",
			}},
			Selector: getLabels(crd.Name),
		},
	}

	ctrl.SetControllerReference(crd, svc, r.Scheme)

	return svc
}

func getLabels(name string) map[string]string {
	return map[string]string{"app": "appshare", "appshare_cr": name}
}
