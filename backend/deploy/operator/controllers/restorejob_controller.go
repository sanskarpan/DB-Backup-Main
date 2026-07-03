package controllers

import (
	"context"
	"fmt"
	"math"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/sanskarpan/db-backup/deploy/operator/api/v1"
)

const restoreJobFinalizer = "backup.db.sanskarpan.com/finalizer"

// RestoreJobReconciler reconciles a RestoreJob object.
type RestoreJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=restorejobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=restorejobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=restorejobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is the main reconciliation loop.
func (r *RestoreJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the RestoreJob instance
	restoreJob := &backupv1.RestoreJob{}
	err := r.Get(ctx, req.NamespacedName, restoreJob)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("RestoreJob resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get RestoreJob")
		return ctrl.Result{}, err
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(restoreJob, restoreJobFinalizer) {
		controllerutil.AddFinalizer(restoreJob, restoreJobFinalizer)
		if updErr := r.Update(ctx, restoreJob); updErr != nil {
			return ctrl.Result{}, updErr
		}
	}

	// Handle deletion
	if !restoreJob.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, restoreJob)
	}

	// Update observed generation
	restoreJob.Status.ObservedGeneration = restoreJob.Generation

	// If job is completed or failed, don't reconcile
	if restoreJob.Status.Phase == "Completed" || restoreJob.Status.Phase == phaseFailed {
		return ctrl.Result{}, nil
	}

	// Initialize status if not set
	if restoreJob.Status.Phase == "" {
		restoreJob.Status.Phase = "Pending"
		restoreJob.Status.StartTime = &metav1.Time{Time: time.Now()}
		restoreJob.Status.Progress = 0
		restoreJob.Status.Attempts = 0

		r.updateCondition(restoreJob, "Initialized", metav1.ConditionTrue, "JobCreated", "Restore job initialized")

		if statusErr := r.Status().Update(ctx, restoreJob); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
	}

	// Validate restore job configuration
	if err = r.validateRestoreJob(restoreJob); err != nil {
		log.Error(err, "Invalid restore job configuration")
		restoreJob.Status.Phase = phaseFailed
		restoreJob.Status.LastError = err.Error()
		r.updateCondition(restoreJob, "Valid", metav1.ConditionFalse, "ValidationFailed", err.Error())

		if updateErr := r.Status().Update(ctx, restoreJob); updateErr != nil {
			log.Error(updateErr, "Failed to update status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil
	}

	r.updateCondition(restoreJob, "Valid", metav1.ConditionTrue, "ValidationSuccess", "Configuration is valid")

	// Create Kubernetes Job to perform restore
	job, err := r.createRestoreJob(ctx, restoreJob)
	if err != nil {
		log.Error(err, "Failed to create restore job")
		restoreJob.Status.Phase = phaseFailed
		restoreJob.Status.LastError = err.Error()
		restoreJob.Status.Attempts++

		// Check retry policy
		if r.shouldRetry(restoreJob) {
			log.Info("Retrying restore job", "attempt", restoreJob.Status.Attempts)
			r.updateCondition(restoreJob, "Retrying", metav1.ConditionTrue, "Retry", fmt.Sprintf("Retrying restore (attempt %d)", restoreJob.Status.Attempts))

			if err := r.Status().Update(ctx, restoreJob); err != nil {
				return ctrl.Result{}, err
			}

			// Calculate backoff delay
			backoffDelay := time.Duration(restoreJob.Spec.RetryPolicy.BackoffSeconds) * time.Second
			return ctrl.Result{RequeueAfter: backoffDelay}, nil
		}

		r.updateCondition(restoreJob, "Failed", metav1.ConditionTrue, "MaxRetriesExceeded", "Maximum retry attempts exceeded")
		if statusErr := r.Status().Update(ctx, restoreJob); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	// Update status with job information
	restoreJob.Status.Phase = "Restoring"
	restoreJob.Status.Progress = 50
	r.updateCondition(restoreJob, "Restoring", metav1.ConditionTrue, "JobRunning", "Restore job is running")

	// Monitor job progress
	if err := r.checkJobStatus(ctx, restoreJob, job); err != nil {
		log.Error(err, "Failed to check job status")
	}

	// Update status
	if err := r.Status().Update(ctx, restoreJob); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Requeue for monitoring
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// reconcileDelete handles restore job deletion.
func (r *RestoreJobReconciler) reconcileDelete(ctx context.Context, restoreJob *backupv1.RestoreJob) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(restoreJob, restoreJobFinalizer) {
		// Perform cleanup
		log.Info("Performing cleanup for RestoreJob", "name", restoreJob.Name)

		// Cancel running restore job if it exists
		jobName := fmt.Sprintf("restore-%s", restoreJob.Name)
		job := &batchv1.Job{}
		err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: restoreJob.Namespace}, job)
		if err == nil {
			// Job exists, delete it
			log.Info("Canceling running restore job", "job", jobName)
			propagationPolicy := metav1.DeletePropagationBackground
			if err = r.Delete(ctx, job, &client.DeleteOptions{
				PropagationPolicy: &propagationPolicy,
			}); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete restore job")
				return ctrl.Result{}, err
			}
		} else if !errors.IsNotFound(err) {
			log.Error(err, "Failed to get restore job for cleanup")
			return ctrl.Result{}, err
		}

		// Clean up temporary resources (ConfigMaps, Secrets created for this restore)
		log.Info("Cleaning up temporary resources for RestoreJob", "name", restoreJob.Name)

		// Delete any temporary ConfigMaps
		configMapList := &corev1.ConfigMapList{}
		if err := r.List(ctx, configMapList,
			client.InNamespace(restoreJob.Namespace),
			client.MatchingLabels{"restore-job": restoreJob.Name}); err == nil {
			for i := range configMapList.Items {
				cm := &configMapList.Items[i]
				log.Info("Deleting temporary ConfigMap", "name", cm.Name)
				if err := r.Delete(ctx, cm); err != nil && !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete ConfigMap", "name", cm.Name)
				}
			}
		}

		// Delete any temporary PVCs created for restore
		pvcList := &corev1.PersistentVolumeClaimList{}
		if err := r.List(ctx, pvcList,
			client.InNamespace(restoreJob.Namespace),
			client.MatchingLabels{"restore-job": restoreJob.Name}); err == nil {
			for i := range pvcList.Items {
				pvc := &pvcList.Items[i]
				log.Info("Deleting temporary PVC", "name", pvc.Name)
				if err := r.Delete(ctx, pvc); err != nil && !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete PVC", "name", pvc.Name)
				}
			}
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(restoreJob, restoreJobFinalizer)
		if updErr := r.Update(ctx, restoreJob); updErr != nil {
			return ctrl.Result{}, updErr
		}
	}

	return ctrl.Result{}, nil
}

// validateRestoreJob validates the restore job configuration.
func (r *RestoreJobReconciler) validateRestoreJob(restoreJob *backupv1.RestoreJob) error {
	// Validate backup ID
	if restoreJob.Spec.BackupID == "" {
		return fmt.Errorf("backup ID is required")
	}

	// Validate target database type
	validDatabaseTypes := map[string]bool{
		"postgres": true,
		"mysql":    true,
		"mongodb":  true,
		"sqlite":   true,
	}
	if !validDatabaseTypes[restoreJob.Spec.TargetDatabase.Type] {
		return fmt.Errorf("invalid database type: %s", restoreJob.Spec.TargetDatabase.Type)
	}

	// Validate target database secret
	if restoreJob.Spec.TargetDatabase.SecretName == "" {
		return fmt.Errorf("target database secret name is required")
	}

	// Validate backup location if specified
	if restoreJob.Spec.BackupLocation != nil {
		validProviders := map[string]bool{
			"s3":    true,
			"azure": true,
			"gcp":   true,
			"local": true,
		}
		if !validProviders[restoreJob.Spec.BackupLocation.Provider] {
			return fmt.Errorf("invalid storage provider: %s", restoreJob.Spec.BackupLocation.Provider)
		}

		if restoreJob.Spec.BackupLocation.Path == "" {
			return fmt.Errorf("backup location path is required")
		}
	}

	return nil
}

// createRestoreJob creates a Kubernetes Job to perform the restore.
func (r *RestoreJobReconciler) createRestoreJob(ctx context.Context, restoreJob *backupv1.RestoreJob) (*batchv1.Job, error) {
	log := log.FromContext(ctx)

	// Build container arguments
	args := []string{
		"restore",
		"--backup-id", restoreJob.Spec.BackupID,
		"--database-type", restoreJob.Spec.TargetDatabase.Type,
	}

	// Add backup location if specified
	if restoreJob.Spec.BackupLocation != nil {
		args = append(
			args,
			"--storage-provider", restoreJob.Spec.BackupLocation.Provider,
			"--storage-path", restoreJob.Spec.BackupLocation.Path,
		)
		if restoreJob.Spec.BackupLocation.Bucket != "" {
			args = append(args, "--storage-bucket", restoreJob.Spec.BackupLocation.Bucket)
		}
		if restoreJob.Spec.BackupLocation.Region != "" {
			args = append(args, "--storage-region", restoreJob.Spec.BackupLocation.Region)
		}
	}

	// Add point-in-time recovery timestamp if specified
	if restoreJob.Spec.PointInTime != nil {
		args = append(args, "--point-in-time", restoreJob.Spec.PointInTime.Format(time.RFC3339))
	}

	// Build environment variables from database secret
	envFrom := []corev1.EnvFromSource{
		{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: restoreJob.Spec.TargetDatabase.SecretName,
				},
			},
		},
	}

	// Add storage credentials secret if specified
	if restoreJob.Spec.BackupLocation != nil && restoreJob.Spec.BackupLocation.CredentialsSecret != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: restoreJob.Spec.BackupLocation.CredentialsSecret,
				},
			},
		})
	}

	// Determine container image
	image := "db-backup:latest"
	if restoreJob.Spec.Image != "" {
		image = restoreJob.Spec.Image
	}

	// Build pod labels
	labels := map[string]string{
		"app":                                 "db-backup",
		"component":                           "restore",
		"restore-job":                         restoreJob.Name,
		"backup.db.sanskarpan.com/managed-by": "restore-operator",
	}

	// Merge custom labels
	if restoreJob.Spec.Labels != nil {
		for k, v := range restoreJob.Spec.Labels {
			labels[k] = v
		}
	}

	// Build resource requirements
	resources := corev1.ResourceRequirements{}
	if restoreJob.Spec.Resources != nil {
		resources = *restoreJob.Spec.Resources
	}

	// Build container
	container := corev1.Container{
		Name:         "restore",
		Image:        image,
		Args:         args,
		EnvFrom:      envFrom,
		Resources:    resources,
		VolumeMounts: []corev1.VolumeMount{},
	}

	// Add volume mounts if specified
	volumes := []corev1.Volume{}
	if restoreJob.Spec.VolumeMounts != nil {
		container.VolumeMounts = restoreJob.Spec.VolumeMounts
		// Note: Volumes would need to be specified in the RestoreJob spec
	}

	// Build pod spec
	podSpec := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		Containers:         []corev1.Container{container},
		Volumes:            volumes,
		ServiceAccountName: "db-backup-restore",
	}

	// Add node selector if specified
	if restoreJob.Spec.NodeSelector != nil {
		podSpec.NodeSelector = restoreJob.Spec.NodeSelector
	}

	// Add tolerations if specified
	if restoreJob.Spec.Tolerations != nil {
		podSpec.Tolerations = restoreJob.Spec.Tolerations
	}

	// Add affinity if specified
	if restoreJob.Spec.Affinity != nil {
		podSpec.Affinity = restoreJob.Spec.Affinity
	}

	// Determine backoff limit
	backoffLimit := int32(0)
	if restoreJob.Spec.RetryPolicy != nil {
		maxAttempts := restoreJob.Spec.RetryPolicy.MaxAttempts
		if maxAttempts > 0 && maxAttempts <= math.MaxInt32 {
			backoffLimit = int32(maxAttempts)
		}
	}

	// Create job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("restore-%s", restoreJob.Name),
			Namespace: restoreJob.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podSpec,
			},
		},
	}

	// Set TTL for automatic cleanup
	if restoreJob.Spec.TTLSecondsAfterFinished != nil {
		job.Spec.TTLSecondsAfterFinished = restoreJob.Spec.TTLSecondsAfterFinished
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(restoreJob, job, r.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Create the job
	log.Info("Creating restore job", "job", job.Name, "backupID", restoreJob.Spec.BackupID)
	if err := r.Create(ctx, job); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create job: %w", err)
		}
		// Job already exists, get it
		existingJob := &batchv1.Job{}
		if err := r.Get(ctx, client.ObjectKey{Name: job.Name, Namespace: job.Namespace}, existingJob); err != nil {
			return nil, fmt.Errorf("failed to get existing job: %w", err)
		}
		log.Info("Restore job already exists", "job", existingJob.Name)
		return existingJob, nil
	}

	log.Info("Restore job created successfully", "job", job.Name)
	return job, nil
}

// checkJobStatus checks the status of the Kubernetes Job.
func (r *RestoreJobReconciler) checkJobStatus(ctx context.Context, restoreJob *backupv1.RestoreJob, job *batchv1.Job) error {
	// Get current job status
	currentJob := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{Name: job.Name, Namespace: job.Namespace}, currentJob); err != nil {
		return err
	}

	// Check if job completed successfully
	if currentJob.Status.Succeeded > 0 {
		restoreJob.Status.Phase = "Completed"
		restoreJob.Status.Progress = 100
		restoreJob.Status.CompletionTime = &metav1.Time{Time: time.Now()}

		if restoreJob.Status.StartTime != nil {
			duration := time.Since(restoreJob.Status.StartTime.Time).Seconds()
			restoreJob.Status.Duration = duration
		}

		r.updateCondition(restoreJob, "Completed", metav1.ConditionTrue, "RestoreSuccess", "Restore completed successfully")
		return nil
	}

	// Check if job failed
	if currentJob.Status.Failed > 0 {
		restoreJob.Status.Phase = phaseFailed
		restoreJob.Status.LastError = "Restore job failed"
		restoreJob.Status.Attempts++

		r.updateCondition(restoreJob, "Failed", metav1.ConditionTrue, "RestoreFailed", "Restore job failed")
		return nil
	}

	return nil
}

// shouldRetry determines if the restore should be retried.
func (r *RestoreJobReconciler) shouldRetry(restoreJob *backupv1.RestoreJob) bool {
	if restoreJob.Spec.RetryPolicy == nil {
		return false
	}

	return restoreJob.Status.Attempts < restoreJob.Spec.RetryPolicy.MaxAttempts
}

// updateCondition updates a condition in the restore job status.
func (r *RestoreJobReconciler) updateCondition(restoreJob *backupv1.RestoreJob, conditionType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: restoreJob.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing condition or append new one
	found := false
	for i, cond := range restoreJob.Status.Conditions {
		if cond.Type == conditionType {
			if cond.Status != status {
				restoreJob.Status.Conditions[i] = condition
			}
			found = true
			break
		}
	}

	if !found {
		restoreJob.Status.Conditions = append(restoreJob.Status.Conditions, condition)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestoreJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.RestoreJob{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
