package controllers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1 "github.com/sanskarpan/db-backup/deploy/operator/api/v1"
)

const backupScheduleFinalizer = "backup.db.sanskarpan.com/finalizer"

// phaseFailed is the status phase used when a resource enters a failed state.
const phaseFailed = "Failed"

// BackupScheduleReconciler reconciles a BackupSchedule object.
type BackupScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backupschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backupschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backupschedules/finalizers,verbs=update
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backuppolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is the main reconciliation loop.
func (r *BackupScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the BackupSchedule instance
	schedule := &backupv1.BackupSchedule{}
	err := r.Get(ctx, req.NamespacedName, schedule)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("BackupSchedule resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get BackupSchedule")
		return ctrl.Result{}, err
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(schedule, backupScheduleFinalizer) {
		controllerutil.AddFinalizer(schedule, backupScheduleFinalizer)
		if err := r.Update(ctx, schedule); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle deletion
	if !schedule.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, schedule)
	}

	// Update observed generation
	schedule.Status.ObservedGeneration = schedule.Generation

	// Check if schedule is suspended
	if schedule.Spec.Suspended {
		schedule.Status.Phase = "Suspended"
		r.updateCondition(schedule, "Active", metav1.ConditionFalse, "Suspended", "Schedule is suspended")
		if err := r.Status().Update(ctx, schedule); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Validate schedule configuration
	if err = r.validateSchedule(schedule); err != nil {
		log.Error(err, "Invalid schedule configuration")
		schedule.Status.Phase = phaseFailed
		r.updateCondition(schedule, "Valid", metav1.ConditionFalse, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, schedule); updateErr != nil {
			log.Error(updateErr, "Failed to update status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	r.updateCondition(schedule, "Valid", metav1.ConditionTrue, "ValidationSuccess", "Configuration is valid")

	// Verify BackupPolicy exists
	policy, err := r.getBackupPolicy(ctx, schedule)
	if err != nil {
		log.Error(err, "Failed to get BackupPolicy")
		schedule.Status.Phase = phaseFailed
		r.updateCondition(schedule, "PolicyFound", metav1.ConditionFalse, "PolicyNotFound", err.Error())
		if updateErr := r.Status().Update(ctx, schedule); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	r.updateCondition(schedule, "PolicyFound", metav1.ConditionTrue, "PolicyExists", fmt.Sprintf("BackupPolicy %s found", policy.Name))

	// Set schedule to active
	schedule.Status.Phase = "Active"
	r.updateCondition(schedule, "Active", metav1.ConditionTrue, "Running", "Schedule is active")

	// Calculate next scheduled time
	nextTime, err := r.calculateNextScheduleTime(schedule)
	if err != nil {
		log.Error(err, "Failed to calculate next schedule time")
		r.updateCondition(schedule, "Scheduled", metav1.ConditionFalse, "ScheduleParseError", err.Error())
	} else {
		schedule.Status.NextScheduleTime = &metav1.Time{Time: nextTime}
		r.updateCondition(schedule, "Scheduled", metav1.ConditionTrue, "ScheduleActive", "Backup scheduled successfully")
		r.maybeRunScheduledBackup(ctx, schedule, policy, nextTime)
	}

	// Clean up old jobs based on history limits
	if err = r.cleanupOldJobs(ctx, schedule); err != nil {
		log.Error(err, "Failed to cleanup old jobs")
	}

	// Update status
	if err = r.Status().Update(ctx, schedule); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Calculate time until next backup
	var requeueAfter time.Duration
	if schedule.Status.NextScheduleTime != nil {
		requeueAfter = time.Until(schedule.Status.NextScheduleTime.Time)
		if requeueAfter < 0 {
			requeueAfter = 30 * time.Second
		}
	} else {
		requeueAfter = 30 * time.Second
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// maybeRunScheduledBackup triggers a backup if it is due, within the
// maintenance window, and permitted by the concurrency policy.
func (r *BackupScheduleReconciler) maybeRunScheduledBackup(ctx context.Context, schedule *backupv1.BackupSchedule, policy *backupv1.BackupPolicy, nextTime time.Time) {
	log := log.FromContext(ctx)

	now := time.Now()
	if schedule.Status.LastScheduleTime != nil && !now.After(nextTime) {
		return
	}

	if !r.isInMaintenanceWindow(schedule, now) {
		log.Info("Backup not triggered - outside maintenance window")
		schedule.Status.MissedRuns++
		r.updateCondition(schedule, "MaintenanceWindow", metav1.ConditionFalse, "OutsideWindow", "Current time is outside maintenance window")
		return
	}

	if !r.canRunBackup(schedule) {
		log.Info("Backup not triggered due to concurrency policy")
		schedule.Status.MissedRuns++
		r.updateCondition(schedule, "ConcurrencyBlocked", metav1.ConditionTrue, "ConcurrentBackupRunning", "Another backup is already running")
		return
	}

	if err := r.triggerBackup(ctx, schedule, policy); err != nil {
		log.Error(err, "Failed to trigger backup")
		schedule.Status.FailedRuns++
		schedule.Status.MissedRuns++
		return
	}

	schedule.Status.LastScheduleTime = &metav1.Time{Time: now}
	schedule.Status.TotalRuns++
	schedule.Status.SuccessfulRuns++
	schedule.Status.LastSuccessfulTime = &metav1.Time{Time: now}
}

// reconcileDelete handles schedule deletion.
func (r *BackupScheduleReconciler) reconcileDelete(ctx context.Context, schedule *backupv1.BackupSchedule) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(schedule, backupScheduleFinalizer) {
		// Perform cleanup
		log.Info("Performing cleanup for BackupSchedule", "name", schedule.Name)

		// Cancel any active backup jobs
		log.Info("Canceling active backup jobs", "schedule", schedule.Name, "activeJobs", len(schedule.Status.ActiveJobs))
		for _, activeJob := range schedule.Status.ActiveJobs {
			job := &batchv1.Job{}
			err := r.Get(ctx, client.ObjectKey{
				Name:      activeJob.Name,
				Namespace: activeJob.Namespace,
			}, job)
			if err == nil {
				log.Info("Canceling active backup job", "job", activeJob.Name)
				propagationPolicy := metav1.DeletePropagationBackground
				if err = r.Delete(ctx, job, &client.DeleteOptions{
					PropagationPolicy: &propagationPolicy,
				}); err != nil && !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete active job", "job", activeJob.Name)
				}
			} else if !errors.IsNotFound(err) {
				log.Error(err, "Failed to get active job", "job", activeJob.Name)
			}
		}

		// Optionally clean up job history
		if schedule.Spec.CleanupOnDelete {
			log.Info("Cleaning up job history for schedule", "schedule", schedule.Name)
			jobList := &batchv1.JobList{}
			if err := r.List(ctx, jobList,
				client.InNamespace(schedule.Namespace),
				client.MatchingLabels{"backup-schedule": schedule.Name}); err != nil {
				log.Error(err, "Failed to list jobs for cleanup")
			} else {
				for i := range jobList.Items {
					job := &jobList.Items[i]
					log.Info("Deleting backup job", "job", job.Name)
					propagationPolicy := metav1.DeletePropagationBackground
					if err := r.Delete(ctx, job, &client.DeleteOptions{
						PropagationPolicy: &propagationPolicy,
					}); err != nil && !errors.IsNotFound(err) {
						log.Error(err, "Failed to delete job", "job", job.Name)
					}
				}
			}
		} else {
			log.Info("Retaining job history (CleanupOnDelete=false)", "schedule", schedule.Name)
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(schedule, backupScheduleFinalizer)
		if err := r.Update(ctx, schedule); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// validateSchedule validates the schedule configuration.
func (r *BackupScheduleReconciler) validateSchedule(schedule *backupv1.BackupSchedule) error {
	// Validate cron schedule
	if _, err := cron.ParseStandard(schedule.Spec.Schedule); err != nil {
		return fmt.Errorf("invalid cron schedule: %w", err)
	}

	// Validate BackupPolicy reference
	if schedule.Spec.BackupPolicyRef.Name == "" {
		return fmt.Errorf("backup policy reference name is required")
	}

	// Validate concurrency policy
	validConcurrencyPolicies := map[string]bool{
		"Allow":   true,
		"Forbid":  true,
		"Replace": true,
	}
	if schedule.Spec.ConcurrencyPolicy != "" && !validConcurrencyPolicies[schedule.Spec.ConcurrencyPolicy] {
		return fmt.Errorf("invalid concurrency policy: %s", schedule.Spec.ConcurrencyPolicy)
	}

	return nil
}

// getBackupPolicy retrieves the referenced BackupPolicy.
func (r *BackupScheduleReconciler) getBackupPolicy(ctx context.Context, schedule *backupv1.BackupSchedule) (*backupv1.BackupPolicy, error) {
	policy := &backupv1.BackupPolicy{}
	namespace := schedule.Spec.BackupPolicyRef.Namespace
	if namespace == "" {
		namespace = schedule.Namespace
	}

	err := r.Get(ctx, types.NamespacedName{
		Name:      schedule.Spec.BackupPolicyRef.Name,
		Namespace: namespace,
	}, policy)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

// calculateNextScheduleTime calculates the next scheduled time.
func (r *BackupScheduleReconciler) calculateNextScheduleTime(schedule *backupv1.BackupSchedule) (time.Time, error) {
	cronSchedule, err := cron.ParseStandard(schedule.Spec.Schedule)
	if err != nil {
		return time.Time{}, err
	}

	// Get base time (last schedule time or now)
	baseTime := time.Now()
	if schedule.Status.LastScheduleTime != nil {
		baseTime = schedule.Status.LastScheduleTime.Time
	}

	// Calculate next time
	nextTime := cronSchedule.Next(baseTime)

	return nextTime, nil
}

// isInMaintenanceWindow checks if current time is within maintenance window.
func (r *BackupScheduleReconciler) isInMaintenanceWindow(schedule *backupv1.BackupSchedule, now time.Time) bool {
	if schedule.Spec.MaintenanceWindow == nil {
		return true // No maintenance window defined, always allowed
	}

	// Check day of week
	if len(schedule.Spec.MaintenanceWindow.Days) > 0 {
		dayAllowed := false
		currentDay := int(now.Weekday())
		for _, allowedDay := range schedule.Spec.MaintenanceWindow.Days {
			if currentDay == allowedDay {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			return false
		}
	}

	// Check time window if specified
	if schedule.Spec.MaintenanceWindow.StartTime != "" && schedule.Spec.MaintenanceWindow.EndTime != "" {
		currentTime := time.Now().Format("15:04")
		startTime := schedule.Spec.MaintenanceWindow.StartTime
		endTime := schedule.Spec.MaintenanceWindow.EndTime

		// Handle time window that crosses midnight (e.g., 22:00 - 02:00)
		if endTime < startTime {
			return currentTime >= startTime || currentTime <= endTime
		}

		// Normal time window (e.g., 02:00 - 06:00)
		return currentTime >= startTime && currentTime <= endTime
	}

	return true
}

// canRunBackup checks if a backup can run based on concurrency policy.
func (r *BackupScheduleReconciler) canRunBackup(schedule *backupv1.BackupSchedule) bool {
	concurrencyPolicy := schedule.Spec.ConcurrencyPolicy
	if concurrencyPolicy == "" {
		concurrencyPolicy = "Forbid" // Default
	}

	// Check if there are active jobs
	hasActiveJobs := len(schedule.Status.ActiveJobs) > 0

	switch concurrencyPolicy {
	case "Allow":
		return true
	case "Forbid":
		return !hasActiveJobs
	case "Replace":
		// Cancel existing active jobs before allowing new one
		if hasActiveJobs {
			log := log.FromContext(context.Background())
			log.Info("Replace policy: canceling existing jobs", "schedule", schedule.Name)

			// Create a background context for job deletion
			ctx := context.Background()

			for _, activeJob := range schedule.Status.ActiveJobs {
				job := &batchv1.Job{}
				err := r.Get(ctx, client.ObjectKey{
					Name:      activeJob.Name,
					Namespace: activeJob.Namespace,
				}, job)
				if err == nil {
					log.Info("Canceling existing job due to Replace policy", "job", activeJob.Name)
					propagationPolicy := metav1.DeletePropagationBackground
					if err = r.Delete(ctx, job, &client.DeleteOptions{
						PropagationPolicy: &propagationPolicy,
					}); err != nil && !errors.IsNotFound(err) {
						log.Error(err, "Failed to delete existing job", "job", activeJob.Name)
					}
				} else if !errors.IsNotFound(err) {
					log.Error(err, "Failed to get existing job", "job", activeJob.Name)
				}
			}

			// Clear active jobs list
			schedule.Status.ActiveJobs = []backupv1.ActiveJob{}
		}
		return true
	}

	return false
}

// triggerBackup triggers a new backup job.
func (r *BackupScheduleReconciler) triggerBackup(ctx context.Context, schedule *backupv1.BackupSchedule, policy *backupv1.BackupPolicy) error {
	log := log.FromContext(ctx)
	log.Info("Triggering backup", "schedule", schedule.Name, "policy", policy.Name)

	// Generate unique job name with timestamp
	jobName := fmt.Sprintf("backup-%s-%d", schedule.Name, time.Now().Unix())

	// Build container arguments
	args := []string{
		"backup",
		"--database-type", policy.Spec.DatabaseType,
	}

	// Add backup type
	if policy.Spec.BackupType != "" {
		args = append(args, "--backup-type", policy.Spec.BackupType)
	}

	// Add storage configuration if specified
	if policy.Spec.Storage != nil {
		args = append(
			args,
			"--storage-provider", policy.Spec.Storage.Provider,
			"--storage-path", policy.Spec.Storage.Path,
		)
		if policy.Spec.Storage.Bucket != "" {
			args = append(args, "--storage-bucket", policy.Spec.Storage.Bucket)
		}
		if policy.Spec.Storage.Region != "" {
			args = append(args, "--storage-region", policy.Spec.Storage.Region)
		}
	}

	// Add compression if enabled
	if policy.Spec.Compression != nil && policy.Spec.Compression.Enabled {
		args = append(args, "--compression", policy.Spec.Compression.Algorithm)
	}

	// Add encryption if enabled
	if policy.Spec.Encryption != nil && policy.Spec.Encryption.Enabled {
		args = append(args, "--encryption", policy.Spec.Encryption.Algorithm)
	}

	// Build environment variables from database secret
	envFrom := []corev1.EnvFromSource{}
	if policy.Spec.DatabaseSecret != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: policy.Spec.DatabaseSecret,
				},
			},
		})
	}

	// Add storage credentials secret if specified
	if policy.Spec.Storage != nil && policy.Spec.Storage.CredentialsSecret != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: policy.Spec.Storage.CredentialsSecret,
				},
			},
		})
	}

	// Determine container image
	image := "db-backup:latest"
	if policy.Spec.Image != "" {
		image = policy.Spec.Image
	}

	// Build pod labels
	labels := map[string]string{
		"app":                                 "db-backup",
		"component":                           "backup",
		"backup-schedule":                     schedule.Name,
		"backup-policy":                       policy.Name,
		"backup.db.sanskarpan.com/managed-by": "backup-operator",
	}

	// Merge custom labels from policy
	if policy.Spec.Labels != nil {
		for k, v := range policy.Spec.Labels {
			labels[k] = v
		}
	}

	// Build resource requirements
	resources := corev1.ResourceRequirements{}
	if policy.Spec.Resources != nil {
		resources = *policy.Spec.Resources
	}

	// Build container
	container := corev1.Container{
		Name:      "backup",
		Image:     image,
		Args:      args,
		EnvFrom:   envFrom,
		Resources: resources,
	}

	// Build pod spec
	podSpec := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		Containers:         []corev1.Container{container},
		ServiceAccountName: "db-backup",
	}

	// Add node selector if specified
	if policy.Spec.NodeSelector != nil {
		podSpec.NodeSelector = policy.Spec.NodeSelector
	}

	// Add tolerations if specified
	if policy.Spec.Tolerations != nil {
		podSpec.Tolerations = policy.Spec.Tolerations
	}

	// Add affinity if specified
	if policy.Spec.Affinity != nil {
		podSpec.Affinity = policy.Spec.Affinity
	}

	// Create backup job
	backoffLimit := int32(3)   // Default retry limit
	ttlSeconds := int32(86400) // 24 hours TTL

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: schedule.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"backup.db.sanskarpan.com/schedule":    schedule.Name,
				"backup.db.sanskarpan.com/policy":      policy.Name,
				"backup.db.sanskarpan.com/backup-type": policy.Spec.BackupType,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podSpec,
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(schedule, job, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Create the job
	log.Info("Creating backup job", "job", jobName, "policy", policy.Name)
	if err := r.Create(ctx, job); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Backup job already exists", "job", jobName)
			return nil
		}
		return fmt.Errorf("failed to create backup job: %w", err)
	}

	// Add to active jobs list
	activeJob := backupv1.ActiveJob{
		Name:      jobName,
		Namespace: schedule.Namespace,
		UID:       string(job.UID),
		StartTime: &metav1.Time{Time: time.Now()},
	}
	schedule.Status.ActiveJobs = append(schedule.Status.ActiveJobs, activeJob)

	log.Info("Backup job created successfully", "job", jobName)
	return nil
}

// cleanupOldJobs removes old job history based on limits.
func (r *BackupScheduleReconciler) cleanupOldJobs(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	logger := log.FromContext(ctx)

	// Get all jobs associated with this schedule
	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, client.InNamespace(schedule.Namespace), client.MatchingLabels{
		"backup-schedule": schedule.Name,
	}); err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	if len(jobList.Items) == 0 {
		return nil
	}

	// Separate successful and failed jobs
	var successfulJobs, failedJobs []batchv1.Job
	for i := range jobList.Items {
		job := jobList.Items[i]
		if isJobSuccessful(&job) {
			successfulJobs = append(successfulJobs, job)
		} else if isJobFailed(&job) {
			failedJobs = append(failedJobs, job)
		}
	}

	// Sort by completion time (oldest first)
	sortJobsByCompletionTime(successfulJobs)
	sortJobsByCompletionTime(failedJobs)

	// Clean up old successful jobs
	successLimit := 3 // Default
	if schedule.Spec.SuccessfulJobsHistoryLimit != nil {
		successLimit = *schedule.Spec.SuccessfulJobsHistoryLimit
	}
	if err := r.cleanupJobsOverLimit(ctx, successfulJobs, successLimit); err != nil {
		logger.Error(err, "Failed to cleanup old successful jobs")
	}

	// Clean up old failed jobs
	failedLimit := 1 // Default
	if schedule.Spec.FailedJobsHistoryLimit != nil {
		failedLimit = *schedule.Spec.FailedJobsHistoryLimit
	}
	if err := r.cleanupJobsOverLimit(ctx, failedJobs, failedLimit); err != nil {
		logger.Error(err, "Failed to cleanup old failed jobs")
	}

	return nil
}

func (r *BackupScheduleReconciler) cleanupJobsOverLimit(ctx context.Context, jobs []batchv1.Job, limit int) error {
	if len(jobs) <= limit {
		return nil
	}

	// Delete oldest jobs beyond the limit
	toDelete := jobs[:len(jobs)-limit]
	for i := range toDelete {
		job := &toDelete[i]
		if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete job %s: %w", job.Name, err)
			}
		}
	}

	return nil
}

func isJobSuccessful(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isJobFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func sortJobsByCompletionTime(jobs []batchv1.Job) {
	sort.Slice(jobs, func(i, j int) bool {
		timeI := getJobCompletionTime(&jobs[i])
		timeJ := getJobCompletionTime(&jobs[j])
		if timeI == nil {
			return false
		}
		if timeJ == nil {
			return true
		}
		return timeI.Before(timeJ)
	})
}

func getJobCompletionTime(job *batchv1.Job) *metav1.Time {
	if job.Status.CompletionTime != nil {
		return job.Status.CompletionTime
	}
	// Fall back to start time if not completed
	return job.Status.StartTime
}

// updateCondition updates a condition in the schedule status.
func (r *BackupScheduleReconciler) updateCondition(schedule *backupv1.BackupSchedule, conditionType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: schedule.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing condition or append new one
	found := false
	for i, cond := range schedule.Status.Conditions {
		if cond.Type == conditionType {
			if cond.Status != status {
				schedule.Status.Conditions[i] = condition
			}
			found = true
			break
		}
	}

	if !found {
		schedule.Status.Conditions = append(schedule.Status.Conditions, condition)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.BackupSchedule{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
