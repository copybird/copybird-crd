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

	backupv1alpha1 "github.com/copybird/copybird-crd/api/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numberOfJobsToShow = 5
)

type JobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile implements controller reconcilation logic
func (r *JobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("Job", req.NamespacedName)

	job := &v1.Job{}
	result := ctrl.Result{
		Requeue: false,
	}

	if err := r.Get(ctx, req.NamespacedName, job); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Object not in the queue", "object", req.NamespacedName)
		} else {
			log.Error(err, "Failed to get runtime object from request")
		}
		return result, nil
	}

	accessor, err := meta.Accessor(job)
	if err != nil {
		log.Error(err, "Failed to get metadata accessor")
		return result, err
	}

	jobOwners := accessor.GetOwnerReferences()
	for _, jobOwner := range jobOwners {
		if !*jobOwner.Controller {
			continue
		}
		ownerCronJob := &v1beta1.CronJob{}
		err := r.Get(ctx, types.NamespacedName{Name: jobOwner.Name, Namespace: job.Namespace}, ownerCronJob)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			log.Info("can't get job owner", "reason", err)
			continue
		}

		cronJobOwners := ownerCronJob.GetOwnerReferences()
		for _, cronJobOwner := range cronJobOwners {
			if !*cronJobOwner.Controller {
				continue
			}
			if cronJobOwner.APIVersion != backupv1alpha1.GroupVersion.String() {
				continue
			}
			if cronJobOwner.Kind != "Backup" {
				continue
			}
			backup := &backupv1alpha1.Backup{}
			err := r.Get(ctx, types.NamespacedName{Name: cronJobOwner.Name, Namespace: job.Namespace}, backup)
			if apierrors.IsNotFound(err) {
				continue
			} else if err != nil {
				log.Info("can't get cronjob owner", "reason", err)
				continue
			}
			currentStatus := backupv1alpha1.JobStatus{
				Name:       job.Name,
				Success:    false,
				StartTime:  job.Status.StartTime,
				FinishTime: job.Status.CompletionTime,
			}

			for _, cond := range job.Status.Conditions {
				if cond.Type == v1.JobComplete {
					if cond.Status == corev1.ConditionTrue {
						currentStatus.Success = true
					}
				}
			}
			update := false
			for i, statusJob := range backup.Status.Jobs {
				if statusJob.Name == currentStatus.Name {
					backup.Status.Jobs[i] = currentStatus
					update = true
					break
				}
			}
			if !update {
				backup.Status.Jobs = append([]backupv1alpha1.JobStatus{currentStatus}, backup.Status.Jobs...)
			}
			if len(backup.Status.Jobs) > numberOfJobsToShow {
				backup.Status.Jobs = backup.Status.Jobs[:numberOfJobsToShow]
			}
			err = r.Update(ctx, backup)
			if err != nil {
				log.Info("can't update backup status", "reason", err)
				result.Requeue = true
			}
		}
	}

	return result, nil
}

func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Job{}).
		Complete(r)
}
