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

// MysqlBackup is the Schema for the mysqlbackups API
type MysqlBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MysqlBackupSpec   `json:"spec,omitempty"`
	Status MysqlBackupStatus `json:"status,omitempty"`
}

// MysqlBackupSpec defines the desired state of MysqlBackup
type MysqlBackupSpec struct {
	Schedule         string           `json:"schedule,omitempty"`
	StorePeriod      string           `json:"storeperiod,omitempty"`
	CompressionLevel int              `json:"compressionLevel,omitempty"`
	Database         MysqlDatabase    `json:"database,omitempty"`
	Storage          BackupStorage    `json:"storage,omitempty"`
	Encryption       BackupEncryption `json:"encryption,omitempty"`
}

// MysqlDatabase contains Mysql database host address, db name and access credentials
type MysqlDatabase struct {
	Host     string                `json:"host,omitempty"`
	Name     string                `json:"name,omitempty"`
	User     SecretValueFromSource `json:"user,omitempty"`
	Password SecretValueFromSource `json:"password,omitempty"`
}

// BackupStorage defines storage target where to store backup archives
type BackupStorage struct {
	Type      string                `json:"type,omitempty"`
	Region    string                `json:"region,omitempty"`
	AccessKey SecretValueFromSource `json:"accessKey,omitempty"`
	SecretKey SecretValueFromSource `json:"secretKey,omitempty"`
}

// BackupEncryption contains backup archive encryption parameters
type BackupEncryption struct {
	Algorithm string                `json:"algorithm,omitempty"`
	Key       SecretValueFromSource `json:"key,omitempty"`
}

// SecretValueFromSource is a reference to Core secret object with key selector
type SecretValueFromSource struct {
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// MysqlBackupStatus defines the observed state of MysqlBackup
type MysqlBackupStatus struct {
	LatestBackupHash      string `json:"latestBackupHash,omitempty"`
	LatestBackupTimestamp string `json:"latestBackupTimestamp,omitempty"`
}

// +kubebuilder:object:root=true

// MysqlBackupList contains a list of MysqlBackup
type MysqlBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MysqlBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MysqlBackup{}, &MysqlBackupList{})
}
