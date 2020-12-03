package controllers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	appsharev1 "github.com/leskil/appshare-operator/api/v1"
)

func (r *AppShareReconciler) createService(ctx context.Context, crd *appsharev1.AppShare, log logr.Logger) error {
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

	log.Info("Creating a new service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
	err := r.Create(ctx, svc)

	return err
}
