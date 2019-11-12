package resources

import (
	backupv1alpha1 "github.com/tzununbekov/copybird-crd/api/v1alpha1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeCronJob(backup *backupv1alpha1.MysqlBackup, copybirdImage string, copybirdEnv []corev1.EnvVar) *v1beta1.CronJob {
	return &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(backup,
					backupv1alpha1.SchemeBuilder.GroupVersion.WithKind("MysqlBackup")),
			},
		},
		Spec: v1beta1.CronJobSpec{
			Schedule: backup.Spec.Schedule,
			JobTemplate: v1beta1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: backup.Name,
				},
				Spec: v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: backup.Name,
						},
						Spec: corev1.PodSpec{
							RestartPolicy: "OnFailure",
							Containers: []corev1.Container{
								corev1.Container{
									Name:  backup.Name,
									Image: copybirdImage,
									Env:   copybirdEnv,
								},
							},
						},
					},
				},
			},
		},
	}
}
