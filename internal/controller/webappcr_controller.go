/*
Copyright 2023.

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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	crwebappv1 "webappcr.io/api/v1"
)

// WebappCRReconciler reconciles a WebappCR object
type WebappCRReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crwebapp.my.domain,resources=webappcrs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crwebapp.my.domain,resources=webappcrs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crwebapp.my.domain,resources=webappcrs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WebappCR object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *WebappCRReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	webappCR := &crwebappv1.WebappCR{}
	if err := r.Get(ctx, req.NamespacedName, webappCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	echoMsg := "echo " + webappCR.Spec.URI
	// Define the desired state of the CronJob based on the WebappCR instance
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webappCR.Name + "-cronjob",
			Namespace: webappCR.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/1 * * * *", // Example: every 5 minutes
			// we need it from specs of cr

			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: pointer.Int32Ptr(3),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "ubuntu",
									Image:   "ubuntu", // Replace with your container image
									Command: []string{"/bin/bash", "-c", echoMsg},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}

	// Set WebappCR instance as the owner and controller
	if err := ctrl.SetControllerReference(webappCR, cronJob, r.Scheme); err != nil {

		return ctrl.Result{}, err
	}

	// Check if this CronJob already exists
	found := &batchv1.CronJob{}
	err := r.Get(ctx, types.NamespacedName{Name: cronJob.Name, Namespace: cronJob.Namespace}, found)
	if err != nil {
		// If the CronJob does not exist, create it
		if err = r.Create(ctx, cronJob); err != nil {
			return ctrl.Result{}, err
		}
	} else if err == nil {
		// Update the CronJob if it already exists and an update is needed
		// Note: You'll need to determine the logic for when an update is necessary
	}

	//

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebappCRReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crwebappv1.WebappCR{}).
		Complete(r)
}
