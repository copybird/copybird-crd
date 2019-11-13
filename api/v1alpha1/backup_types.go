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
	Schedule         string           `json:"schedule,omitempty"`
	StorePeriod      string           `json:"storeperiod,omitempty"`
	CompressionLevel int              `json:"compressionlevel,omitempty"`
	Database         Database         `json:"database,omitempty"`
	Storage          BackupStorage    `json:"storage,omitempty"`
	Encryption       BackupEncryption `json:"encryption,omitempty"`
}

// Database contains  database host address, db name and access credentials
type Database struct {
	Type     string                `json:"type,omitempty"`
	Host     string                `json:"host,omitempty"`
	Name     string                `json:"name,omitempty"`
	User     SecretValueFromSource `json:"user,omitempty"`
	Password SecretValueFromSource `json:"password,omitempty"`
}

// BackupStorage defines storage target where to store backup archives
type BackupStorage struct {
	Type      string                `json:"type,omitempty"`
	Region    string                `json:"region,omitempty"`
	Bucket    string                `json:"bucket,omitempty"`
	AccessKey SecretValueFromSource `json:"accesskey,omitempty"`
	SecretKey SecretValueFromSource `json:"secretkey,omitempty"`
}

// BackupEncryption contains backup archive encryption parameters
type BackupEncryption struct {
	Type string                `json:"type,omitempty"`
	Key  SecretValueFromSource `json:"key,omitempty"`
}

// SecretValueFromSource is a reference to Core secret object with key selector
type SecretValueFromSource struct {
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	LatestBackupHash         string       `json:"latestBackupHash,omitempty"`
	LatestBackupTimestamp    string       `json:"latestBackupTimestamp,omitempty"`
	EncryptionSecretProvided bool         `json:"encryptionSecretProvided"`
	Input                    InputStatus  `json:"input"`
	Output                   OutputStatus `json:"output"`
}

// InputStatus represents statuses of backup input component
type InputStatus struct {
	SecretProvided bool `json:"inputSecretProvided"`
}

// OutputStatus represents statuses of backup output component
type OutputStatus struct {
	SecretProvided bool `json:"OutputSecretProvided"`
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
