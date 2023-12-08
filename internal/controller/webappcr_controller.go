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
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	// "sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	crwebappv1 "webappcr.io/api/v1"
)

// WebappCRReconciler reconciles a WebappCR object
type WebappCRReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crwebapp.my.domain,resources=webappcrs,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=crwebapp.my.domain,resources=webappcrs/status,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=crwebapp.my.domain,resources=webappcrs/finalizers,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get;list;watch

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
	log := log.FromContext(ctx)
	webappCR := &crwebappv1.WebappCR{}
	if err := r.Get(ctx, req.NamespacedName, webappCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// echoMsg := "echo " + webappCR.Spec.URI

	// log.Info("Reconcile")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webappCR.Name,
			Namespace: webappCR.Namespace,
		},
		Data: map[string]string{
			"NumRetries": strconv.Itoa(int(webappCR.Spec.NumRetries)),
			"URI":        webappCR.Spec.URI,
		},
	}

	numRetries, error1 := strconv.ParseInt(configMap.Data["NumRetries"], 10, 32)
	if error1 != nil {
		log.Error(error1, "Error:")
		// return
	}

	// Define the desired state of the CronJob based on the WebappCR instance
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webappCR.Name,
			Namespace: webappCR.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/15 * * * *", // Example: every 5 minutes
			// we need it from specs of cr

			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: pointer.Int32Ptr(int32(numRetries)),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"owner-cronjob": webappCR.Name,
								"app": "kafka-producer",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "kafka-producer",
									Image: "sumanthksai/kafka-producer:latest",
									Env: []corev1.EnvVar{
										{
											Name:  "URI",
											Value: configMap.Data["URI"],
										},
										{
											Name:  "Retries",
											Value: configMap.Data["NumRetries"],
										},
										{
											Name:  "BROKER_ENDPOINT",
											Value: "dev-kafka.consumer.svc.cluster.local:9092",
										},
										{
											Name:  "SASL_USERNAME",
											Value: "user1",
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
							ImagePullSecrets: []corev1.LocalObjectReference{
								{Name: "webappcr-docker-regcred"},
							},
						},
					},
				},
			},
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime: &metav1.Time{Time: time.Now()},
		},
	}

	// Set WebappCR instance as the owner and controller
	if err := ctrl.SetControllerReference(webappCR, cronJob, r.Scheme); err != nil {

		return ctrl.Result{}, err
	}

	foundCronJob := &batchv1.CronJob{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: cronJob.Name, Namespace: cronJob.Namespace}, foundCronJob)
	if err != nil {

		if err = r.Create(ctx, cronJob); err != nil {
			return ctrl.Result{}, err
		}
	} else if err == nil {
		// Update the CronJob if it already exists and an update is needed
		// Note: You'll need to determine the logic for when an update is necessary
		cronJob.Spec.JobTemplate.Spec.BackoffLimit = pointer.Int32Ptr(webappCR.Spec.NumRetries)

		// Status Update
		if err := r.Update(ctx, cronJob); err != nil {
			// log.Error(err, "unable to update CronJob spec")
			return ctrl.Result{}, err
		}

	}

	if err := ctrl.SetControllerReference(webappCR, configMap, r.Scheme); err != nil {

		return ctrl.Result{}, err
	}

	foundConfigMap := &corev1.ConfigMap{}
	err2 := r.Client.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err2 != nil {
		if err2 = r.Create(ctx, configMap); err2 != nil {
			return ctrl.Result{}, err2
		}

	} else if err2 == nil {
		// Update the CronJob if it already exists and an update is needed
		// Note: You'll need to determine the logic for when an update is necessary
		configMap.Data["NumRetries"] = strconv.Itoa(int(webappCR.Spec.NumRetries))
		configMap.Data["URI"] = webappCR.Spec.URI
		// Status Update
		if err2 := r.Update(ctx, configMap); err2 != nil {
			return ctrl.Result{}, err2
		}

	}

	finalizerName := "core.webappcr.io/finalizer"
	// CronJobfinalizerName := "batch.webappcr.io/finalizer"

	if webappCR.ObjectMeta.DeletionTimestamp.IsZero() {
		// log.Info("webappCR DeletionTimestamp IsZero")
		// CR is not being deleted
		if !containsString(webappCR.ObjectMeta.Finalizers, finalizerName) {
			// Add finalizer to CR
			webappCR.ObjectMeta.Finalizers = append(webappCR.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, webappCR); err != nil {
				return ctrl.Result{}, err
			}

		}
		// if !containsString(webappCR.ObjectMeta.Finalizers, finalizerName) {
		// 	// Add finalizer to CR
		// 	webappCR.ObjectMeta.Finalizers = append(webappCR.ObjectMeta.Finalizers, finalizerName)
		// 	if err := r.Update(ctx, webappCR); err != nil {
		// 		return ctrl.Result{}, err
		// 	}

		// }
	} else {
		log.Info("webappCR DeletionTimestamp else")
		// CR is being deleted
		if containsString(webappCR.ObjectMeta.Finalizers, finalizerName) {
			// Run your cleanup logic, e.g., delete associated CronJob
			if err := r.Get(ctx, req.NamespacedName, cronJob); err == nil {
				// return ctrl.Result{}, client.IgnoreNotFound(err)
				cronJob.ObjectMeta.Finalizers = removeString(cronJob.ObjectMeta.Finalizers, finalizerName)
				if err := r.Update(ctx, cronJob); err != nil {
					return ctrl.Result{}, err
				}
				if err := r.deleteCronJob(ctx, cronJob); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
		if containsString(webappCR.ObjectMeta.Finalizers, finalizerName) {
			// Run your cleanup logic, e.g., delete associated CronJob
			if err := r.Get(ctx, req.NamespacedName, configMap); err == nil {
				// return ctrl.Result{}, client.IgnoreNotFound(err)
				configMap.ObjectMeta.Finalizers = removeString(configMap.ObjectMeta.Finalizers, finalizerName)
				if err := r.Update(ctx, configMap); err != nil {
					return ctrl.Result{}, err
				}
				if err := r.deleteConfigMap(ctx, configMap); err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		// Remove finalizer from CR
		webappCR.ObjectMeta.Finalizers = removeString(webappCR.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, webappCR); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
		// The CR will be garbage collected by Kubernetes
	}

	if err := r.Get(ctx, req.NamespacedName, cronJob); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Adding the finalizer in CronJob
	if cronJob.ObjectMeta.DeletionTimestamp.IsZero() {
		// log.Info("cronjob DeletionTimestamp IsZero")
		if !controllerutil.ContainsFinalizer(cronJob, finalizerName) {
			controllerutil.AddFinalizer(cronJob, finalizerName)
			if err := r.Update(ctx, cronJob); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err

		}
	}
	// else {
	// 	log.Info("cronjob DeletionTimestamp else")
	// 	if controllerutil.ContainsFinalizer(cronJob, finalizerName) {
	// 		log.Info("cronjob Contains Finalizer")
	// 		controllerutil.RemoveFinalizer(cronJob, finalizerName)
	// 		if err := r.Update(ctx, cronJob); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 		return ctrl.Result{}, err

	// 	}
	// }

	if err := r.Get(ctx, req.NamespacedName, configMap); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Adding the finalizer in ConfigMap
	if configMap.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("configMap DeletionTimestamp IsZero")
		if !controllerutil.ContainsFinalizer(configMap, finalizerName) {
			controllerutil.AddFinalizer(configMap, finalizerName)
			if err := r.Update(ctx, cronJob); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err

		}
	}
	// else {
	// 	if controllerutil.ContainsFinalizer(configMap, finalizerName) {
	// 		log.Info("cronjob DeletionTimestamp else")
	// 		controllerutil.RemoveFinalizer(configMap, finalizerName)
	// 		if err := r.Update(ctx, configMap); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 		return ctrl.Result{}, err

	// 	}
	// }

	//execute status

	labels := cronJob.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["owner-cronjob"] = webappCR.Name
	cronJob.SetLabels(labels)

	// Check if there are any remaining child resources
	// if err := r.areChildResourcesDeleted(ctx, cronJob); err != nil {
	// 	// log.Error(err, "error checking child resources")
	// 	return ctrl.Result{}, err
	// }

	// Check if this CronJob already exists

	cronJobStatus := &batchv1.CronJob{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(cronJob), cronJobStatus); err != nil {
		log.Error(err, "unable to get CronJob status")
		return ctrl.Result{}, err
	}
	// Check if the status field is not nil before accessing LastScheduleTime
	if cronJobStatus.Status.LastScheduleTime != nil {
		webappCR.Status.LastExecutionTime = metav1.Time{Time: cronJobStatus.Status.LastScheduleTime.Time}
	} else {
		log.Info("CronJob status is nil")
		// Handle the case where the status is nil, log an error, or take appropriate action
	}

	// execute status

	var childJobs kbatch.JobList
	var activeJobs []*kbatch.Job

	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	isJobFinished := func(job *kbatch.Job) (bool, kbatch.JobConditionType) {
		for _, c := range job.Status.Conditions {

			if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}

	for i, job := range childJobs.Items {
		finished, _ := isJobFinished(&job)
		// log.Info(job.Name)
		if !finished {
			activeJobs = append(activeJobs, &childJobs.Items[i])
		}
	}

	if len(activeJobs) > 0 {
		webappCR.Status.ExecutionStatus = "Active"
		r.Status().Update(ctx, webappCR)
	} else {
		webappCR.Status.ExecutionStatus = "Inactive"
		r.Status().Update(ctx, webappCR)

	}

	if err := r.Status().Update(ctx, webappCR); err != nil {
		log.Error(err, "unable to update WebappCR status")
		return ctrl.Result{}, err
	}

	// containsString checks if a slice contains a specific string.

	return ctrl.Result{}, nil
}

// deleteCronJob deletes the CronJob associated with the given WebappCR.
func (r *WebappCRReconciler) deleteCronJob(ctx context.Context, cronJob *batchv1.CronJob) error {
	log := log.FromContext(ctx)

	// Create an instance of CronJob corresponding to the WebappCR
	// cronJob := &batchv1.CronJob{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      webappCR.Name,
	// 		Namespace: webappCR.Namespace,
	// 	},
	// }

	// Attempt to delete the CronJob
	if err := r.Client.Delete(ctx, cronJob); err != nil {
		log.Error(err, "Failed to delete CronJob", "Name", cronJob.Name, "Namespace", cronJob.Namespace)
		return err
	}

	log.Info("Deleted CronJob", "Name", cronJob.Name, "Namespace", cronJob.Namespace)
	return nil
}

func (r *WebappCRReconciler) deleteConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	log := log.FromContext(ctx)

	// Create an instance of CronJob corresponding to the WebappCR
	// cronJob := &batchv1.CronJob{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      webappCR.Name,
	// 		Namespace: webappCR.Namespace,
	// 	},
	// }

	// Attempt to delete the CronJob
	if err := r.Client.Delete(ctx, configMap); err != nil {
		log.Error(err, "Failed to delete CronJob", "Name", configMap.Name, "Namespace", configMap.Namespace)
		return err
	}

	log.Info("Deleted ConfigMap", "Name", configMap.Name, "Namespace", configMap.Namespace)
	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// removeString removes a specific string from a slice.
func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebappCRReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crwebappv1.WebappCR{}).
		Owns(&batchv1.CronJob{}).  // Watch CronJob
		Owns(&corev1.ConfigMap{}). // Watch ConfigMap
		Complete(r)
}
