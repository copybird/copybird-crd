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
	"fmt"

	"github.com/go-logr/logr"
	backupv1alpha1 "github.com/tzununbekov/copybird-crd/api/v1alpha1"
	"github.com/tzununbekov/copybird-crd/controllers/resources"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
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
	controllerAgentName = "copybird-backup-controller"
	finalizerName       = controllerAgentName
)

// MysqlBackupReconciler reconciles a MysqlBackup object
type MysqlBackupReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	CopyBirdImage string
}

// +kubebuilder:rbac:groups=copybird.org,resources=mysqlbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=copybird.org,resources=mysqlbackups/status,verbs=get;update;patch

// Reconcile implements controllbackup.Nameer reconcilation logic
func (r *MysqlBackupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("mysqlbackup", req.NamespacedName)

	backup := &backupv1alpha1.MysqlBackup{}
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
		result.Requeue = true
	}

	if reconcileErr != nil {
		log.Error(err, "Reconcilation failed")
		result.Requeue = true
		return result, err
	}

	if err := r.Update(context.Background(), backup); err != nil {
		log.Error(err, "Failed to update object")
		return result, err
	}

	return result, nil
}

func (r *MysqlBackupReconciler) reconcile(ctx context.Context, backup *backupv1alpha1.MysqlBackup) error {
	log := r.Log.WithName("reconciler")
	_ = log
	if len(backup.GetFinalizers()) == 0 {
		backup.Finalizers = []string{finalizerName}
	}

	cronjob, err := r.getOwnedCronjob(ctx, backup)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cronjob = resources.MakeCronJob(backup, r.CopyBirdImage)
			if err = controllerutil.SetControllerReference(backup, cronjob, r.Scheme); err != nil {
				return err
			}
			if err = r.Create(ctx, cronjob); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func (r *MysqlBackupReconciler) getOwnedCronjob(ctx context.Context, backup *backupv1alpha1.MysqlBackup) (*v1beta1.CronJob, error) {
	cronjob := &v1beta1.CronJob{}
	err := r.Get(ctx, client.ObjectKey{Namespace: backup.Namespace, Name: backup.Name}, cronjob)
	if err != nil {
		return nil, err
	}
	if metav1.IsControlledBy(cronjob, backup) {
		return cronjob, nil
	}
	return nil, apierrors.NewNotFound(v1beta1.Resource("cronjob"), "")
}

func (r *MysqlBackupReconciler) finalize(ctx context.Context, backup *backupv1alpha1.MysqlBackup) error {
	log := r.Log.WithName("finalizer")
	_ = log
	finalizers := sets.NewString(backup.Finalizers...)
	finalizers.Delete(finalizerName)
	backup.Finalizers = finalizers.List()
	return nil
}

func (r *MysqlBackupReconciler) secretFrom(ctx context.Context, namespace string, secretKeySelector *corev1.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretKeySelector.Name}, secret)
	if err != nil {
		return "", err
	}
	secretVal, ok := secret.Data[secretKeySelector.Key]
	if !ok {
		return "", fmt.Errorf(`key "%s" not found in secret "%s"`, secretKeySelector.Key, secretKeySelector.Name)
	}
	return string(secretVal), nil
}

func (r *MysqlBackupReconciler) ConstructArguments(ctx context.Context, backup *backupv1alpha1.MysqlBackup) error {
	return nil
}

func (r *MysqlBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.MysqlBackup{}).
		Complete(r)
}
