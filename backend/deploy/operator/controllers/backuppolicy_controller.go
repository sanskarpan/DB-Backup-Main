package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
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

const backupPolicyFinalizer = "backup.db.sanskarpan.com/finalizer"

// BackupPolicyReconciler reconciles a BackupPolicy object.
type BackupPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backuppolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backuppolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backuppolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=backup.db.sanskarpan.com,resources=backupschedules,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is the main reconciliation loop.
func (r *BackupPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the BackupPolicy instance
	policy := &backupv1.BackupPolicy{}
	err := r.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("BackupPolicy resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get BackupPolicy")
		return ctrl.Result{}, err
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(policy, backupPolicyFinalizer) {
		controllerutil.AddFinalizer(policy, backupPolicyFinalizer)
		if err := r.Update(ctx, policy); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle deletion
	if !policy.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, policy)
	}

	// Update observed generation
	policy.Status.ObservedGeneration = policy.Generation

	// Validate policy configuration
	if err := r.validatePolicy(policy); err != nil {
		log.Error(err, "Invalid policy configuration")
		r.updateCondition(policy, "Valid", metav1.ConditionFalse, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, policy); updateErr != nil {
			log.Error(updateErr, "Failed to update policy status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	r.updateCondition(policy, "Valid", metav1.ConditionTrue, "ValidationSuccess", "Policy configuration is valid")

	// Check if policy is suspended
	if policy.Spec.Suspended {
		policy.Status.Phase = "Suspended"
		r.updateCondition(policy, "Active", metav1.ConditionFalse, "Suspended", "Policy is suspended")
		if err := r.Status().Update(ctx, policy); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Set policy to active
	policy.Status.Phase = "Active"
	r.updateCondition(policy, "Active", metav1.ConditionTrue, "Running", "Policy is active")

	// Calculate next backup time if schedule is set
	if policy.Spec.Schedule != "" {
		nextTime, err := r.calculateNextBackupTime(policy.Spec.Schedule)
		if err != nil {
			log.Error(err, "Failed to calculate next backup time")
			r.updateCondition(policy, "Scheduled", metav1.ConditionFalse, "ScheduleParseError", err.Error())
		} else {
			policy.Status.NextBackupTime = &metav1.Time{Time: nextTime}
			r.updateCondition(policy, "Scheduled", metav1.ConditionTrue, "ScheduleActive", "Backup scheduled successfully")
		}
	}

	// Update status
	if err := r.Status().Update(ctx, policy); err != nil {
		log.Error(err, "Failed to update policy status")
		return ctrl.Result{}, err
	}

	// Requeue for next reconciliation
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// reconcileDelete handles policy deletion.
func (r *BackupPolicyReconciler) reconcileDelete(ctx context.Context, policy *backupv1.BackupPolicy) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(policy, backupPolicyFinalizer) {
		return ctrl.Result{}, nil
	}

	// Perform cleanup
	log.Info("Performing cleanup for BackupPolicy", "name", policy.Name)
	r.cancelInProgressBackups(ctx, policy)

	// Optionally delete associated backups based on policy.
	if policy.Spec.DeletionPolicy == "Delete" {
		r.deleteAssociatedBackups(ctx, policy)
	} else {
		log.Info("Retaining associated backups (DeletionPolicy=Retain)", "policy", policy.Name)
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(policy, backupPolicyFinalizer)
	if err := r.Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// cancelInProgressBackups cancels active backup jobs for schedules that
// reference the policy being deleted.
func (r *BackupPolicyReconciler) cancelInProgressBackups(ctx context.Context, policy *backupv1.BackupPolicy) {
	log := log.FromContext(ctx)
	log.Info("Canceling in-progress backups for policy", "policy", policy.Name)

	scheduleList := &backupv1.BackupScheduleList{}
	if err := r.List(ctx, scheduleList, client.InNamespace(policy.Namespace)); err != nil {
		log.Error(err, "Failed to list backup schedules")
		return
	}

	for i := range scheduleList.Items {
		schedule := &scheduleList.Items[i]
		if schedule.Spec.BackupPolicyRef.Name != policy.Name {
			continue
		}
		for _, activeJob := range schedule.Status.ActiveJobs {
			r.cancelActiveJob(ctx, activeJob)
		}
	}
}

// cancelActiveJob deletes a single active backup job if it still exists.
func (r *BackupPolicyReconciler) cancelActiveJob(ctx context.Context, activeJob backupv1.ActiveJob) {
	log := log.FromContext(ctx)
	log.Info("Canceling active backup job", "job", activeJob.Name)

	job := &batchv1.Job{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      activeJob.Name,
		Namespace: activeJob.Namespace,
	}, job); err != nil {
		return
	}

	propagationPolicy := metav1.DeletePropagationBackground
	if err := r.Delete(ctx, job, &client.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete active job", "job", activeJob.Name)
	}
}

// deleteAssociatedBackups deletes backup jobs created by the policy when the
// deletion policy requests cascade deletion.
func (r *BackupPolicyReconciler) deleteAssociatedBackups(ctx context.Context, policy *backupv1.BackupPolicy) {
	log := log.FromContext(ctx)
	log.Info("Deleting associated backups for policy", "policy", policy.Name)

	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList,
		client.InNamespace(policy.Namespace),
		client.MatchingLabels{"backup-policy": policy.Name}); err != nil {
		log.Error(err, "Failed to list backup jobs")
		return
	}

	for i := range jobList.Items {
		job := &jobList.Items[i]
		log.Info("Deleting backup job", "job", job.Name)
		propagationPolicy := metav1.DeletePropagationBackground
		if err := r.Delete(ctx, job, &client.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		}); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete backup job", "job", job.Name)
		}
	}

	// Delete backup artifacts from storage (if implemented).
	// This would require integration with storage providers.
	log.Info("Backup artifacts cleanup would be performed here", "policy", policy.Name)
}

// validatePolicy validates the policy configuration.
func (r *BackupPolicyReconciler) validatePolicy(policy *backupv1.BackupPolicy) error {
	// Validate database type
	validDatabaseTypes := map[string]bool{
		"postgres": true,
		"mysql":    true,
		"mongodb":  true,
		"sqlite":   true,
	}
	if !validDatabaseTypes[policy.Spec.DatabaseType] {
		return fmt.Errorf("invalid database type: %s", policy.Spec.DatabaseType)
	}

	// Validate backup type
	if policy.Spec.BackupType != "" {
		validBackupTypes := map[string]bool{
			"full":         true,
			"incremental":  true,
			"differential": true,
		}
		if !validBackupTypes[policy.Spec.BackupType] {
			return fmt.Errorf("invalid backup type: %s", policy.Spec.BackupType)
		}
	}

	// Validate schedule if set
	if policy.Spec.Schedule != "" {
		if _, err := cron.ParseStandard(policy.Spec.Schedule); err != nil {
			return fmt.Errorf("invalid cron schedule: %w", err)
		}
	}

	// Validate retention policy
	if policy.Spec.Retention.Daily < 1 {
		return fmt.Errorf("retention.daily must be at least 1")
	}

	// Validate storage configuration
	if policy.Spec.Storage != nil {
		validProviders := map[string]bool{
			"s3":    true,
			"azure": true,
			"gcp":   true,
			"local": true,
		}
		if !validProviders[policy.Spec.Storage.Provider] {
			return fmt.Errorf("invalid storage provider: %s", policy.Spec.Storage.Provider)
		}
	}

	return nil
}

// calculateNextBackupTime calculates the next backup time based on schedule.
func (r *BackupPolicyReconciler) calculateNextBackupTime(schedule string) (time.Time, error) {
	cronSchedule, err := cron.ParseStandard(schedule)
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now()
	nextTime := cronSchedule.Next(now)

	return nextTime, nil
}

// updateCondition updates a condition in the policy status.
func (r *BackupPolicyReconciler) updateCondition(policy *backupv1.BackupPolicy, conditionType string, status metav1.ConditionStatus, reason, message string) {
	policy.Status.Conditions = upsertCondition(policy.Status.Conditions, policy.Generation, conditionType, status, reason, message)
}

// upsertCondition updates the matching condition in conditions (only when its
// status changed) or appends a new one, returning the resulting slice. The
// condition's ObservedGeneration is set to generation.
func upsertCondition(conditions []metav1.Condition, generation int64, conditionType string, status metav1.ConditionStatus, reason, message string) []metav1.Condition {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing condition or append new one
	for i, cond := range conditions {
		if cond.Type == conditionType {
			// Only update if status changed
			if cond.Status != status {
				conditions[i] = condition
			}
			return conditions
		}
	}

	return append(conditions, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.BackupPolicy{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
