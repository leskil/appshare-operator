package controllers

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	appsharev1 "github.com/leskil/appshare-operator/api/v1"

	"github.com/google/go-cmp/cmp"
)

func applyCrdChangesToDeployment(dep *appsv1.Deployment, crd *appsharev1.AppShare) bool {

	hasChanges := false

	if dep.Spec.Replicas != &crd.Spec.Replicas {
		dep.Spec.Replicas = &crd.Spec.Replicas
		hasChanges = true
	}

	if !cmp.Equal(dep.Spec.Template.Spec.Containers[0].Resources, crd.Spec.Resources) {
		dep.Spec.Template.Spec.Containers[0].Resources = crd.Spec.Resources
		hasChanges = true
	}

	return hasChanges
}

func (r *AppShareReconciler) createDeployment(ctx context.Context, crd *appsharev1.AppShare, log logr.Logger) error {
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
						Resources: crd.Spec.Resources,
					},
					}},
			},
		},
	}

	ctrl.SetControllerReference(crd, dep, r.Scheme)

	log.Info("Creating a new deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
	err := r.Create(ctx, dep)

	return err
}
