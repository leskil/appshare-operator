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

	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
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
		err = r.createDeployment(ctx, appshare, log)

		if err != nil {
			log.Error(err, "Failed to create new deployment", "Deployment.Namespace", appshare.Name, "Deployment.Name", appshare.Name)
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
		err = r.createService(ctx, appshare, log)

		if err != nil {
			log.Error(err, "Failed to create new service", "Service.Namespace", appshare.Name, "Service.Name", appshare.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get service. Check permissions.")
		return ctrl.Result{}, err
	}

	err = updateResources(ctx, r, appshare, existingDeployment)

	return ctrl.Result{}, err
}

func updateResources(ctx context.Context, r *AppShareReconciler, crd *appsharev1.AppShare, deployment *appsv1.Deployment) error {
	hasChanges := applyCrdChangesToDeployment(deployment, crd)
	if hasChanges {
		err := r.Update(ctx, deployment)
		return err
	}

	return nil
}
