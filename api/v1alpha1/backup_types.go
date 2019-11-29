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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// Backup is the Schema for the backups API
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec,omitempty"`
	Status BackupStatus `json:"status,omitempty"`
}

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	Schedule string `json:"schedule,omitempty"`
	Input    Module `json:"input,omitempty"`
	Output   Module `json:"output,omitempty"`
	Encrypt  Module `json:"encrypt,omitempty"`
	Compress Module `json:"compress,omitempty"`
}

// Module is a Copybird module representation
type Module struct {
	Type    string         `json:"type,omitempty"`
	Params  []ModuleParam  `json:"params,omitempty"`
	Secrets []ModuleSecret `json:"secrets,omitempty"`
}

// ModuleParam contains key-value module parameter
type ModuleParam struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// ModuleSecret contains a secret used by module
type ModuleSecret struct {
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	CronjobName           string       `json:"cronjobName,omitempty"`
	LatestBackupTimestamp string       `json:"latestBackupTimestamp,omitempty"`
	Input                 ModuleStatus `json:"input,omitempty"`
	Output                ModuleStatus `json:"output,omitempty"`
	Compress              ModuleStatus `json:"compress,omitempty"`
	Encrypt               ModuleStatus `json:"encrypt,omitempty"`
	Jobs                  []JobStatus  `json:"jobs,omitempty"`
}

type JobStatus struct {
	Name       string       `json:"name,omitempty"`
	Success    bool         `json:"success"`
	StartTime  *metav1.Time `json:"startTime,omitempty"`
	FinishTime *metav1.Time `json:"finishTime,omitempty"`
}

// ModuleStatus is a list of module statuses
type ModuleStatus struct {
	// SecretsProvided bool `json:"secretsProvided"`
}

// +kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
