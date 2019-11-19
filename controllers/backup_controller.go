/*
Copyright 2019 Mad Devs.

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
	"os"

	backupv1alpha1 "github.com/copybird/copybird-crd/api/v1alpha1"
	"github.com/copybird/copybird-crd/controllers/resources"
	"github.com/go-logr/logr"
	"k8s.io/api/batch/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizerName        = "copybird-backup-controller"
	copybirdImageEnvVar  = "COPYBIRD_IMAGE"
	copybirdDefaultImage = "copybird/copybird:latest"
	// use hardcoded "gzip" compression for now
	compressionType = "gzip"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=copybird.org,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=copybird.org,resources=backups/status,verbs=get;update;patch

// Reconcile implements controllbackup.Nameer reconcilation logic
func (r *BackupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("backup", req.NamespacedName)

	backup := &backupv1alpha1.Backup{}
	result := ctrl.Result{
		Requeue: false,
	}

	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Object not in the queue", "object", req.NamespacedName)
		} else {
			log.Error(err, "Failed to get runtime object from request")
		}
		return result, nil
	}

	accessor, err := meta.Accessor(backup)
	if err != nil {
		log.Error(err, "Failed to get metadata accessor")
		return result, err
	}

	var reconcileErr error
	if accessor.GetDeletionTimestamp() == nil {
		reconcileErr = r.reconcile(ctx, backup)
	} else {
		reconcileErr = r.finalize(ctx, backup)
	}

	if reconcileErr != nil {
		result.Requeue = true
		log.Error(reconcileErr, "reconcilation error")
		// return result, err
	}

	return result, nil
}

func (r *BackupReconciler) reconcile(ctx context.Context, backup *backupv1alpha1.Backup) error {
	log := r.Log.WithName("reconciler")
	if len(backup.GetFinalizers()) == 0 {
		backup.Finalizers = []string{finalizerName}
	}

	copybirdImage, defined := os.LookupEnv(copybirdImageEnvVar)
	if !defined {
		log.Info("environment variable \"" + copybirdImageEnvVar + "\" not defined, using default value: \"" + copybirdDefaultImage)
		copybirdImage = copybirdDefaultImage
	}

	backup.Status.Input.SecretsProvided = true
	backup.Status.Output.SecretsProvided = true
	backup.Status.Encrypt.SecretsProvided = true
	backup.Status.Compress.SecretsProvided = true
	backup.Status.LatestBackupHash = "unknown"

	copybird := resources.NewCopyBirdParams(copybirdImage, backup)

	cronjob := &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
	}

	err := r.Get(ctx, client.ObjectKey{Namespace: cronjob.Namespace, Name: cronjob.Name}, cronjob)
	if apierrors.IsNotFound(err) {
		cronjob = copybird.MakeCronJob(ctx)
	} else if err != nil {
		return err
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, cronjob, func() error {
		if cronjob.ObjectMeta.CreationTimestamp.IsZero() {
			if err = controllerutil.SetControllerReference(backup, cronjob, r.Scheme); err != nil {
				return err
			}
		} else {
			newCronjob := copybird.MakeCronJob(ctx)
			cronjob.Spec = newCronjob.Spec
		}
		return nil
	})
	if err != nil {
		return err
	}
	log.Info("Cronjob successfully reconciled", "operation", op)
	return nil
}

func (r *BackupReconciler) finalize(ctx context.Context, backup *backupv1alpha1.Backup) error {
	log := r.Log.WithName("finalizer")
	_ = log
	finalizers := sets.NewString(backup.Finalizers...)
	finalizers.Delete(finalizerName)
	backup.Finalizers = finalizers.List()
	return nil
}

func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.Backup{}).
		Complete(r)
}
