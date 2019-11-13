package resources

import (
	"context"

	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CopyBirdParams struct {
	Name        string
	Namespace   string
	Image       string
	Schedule    string
	Input       string
	Compression string
	Encryption  string
	Output      string
}

func NewCopyBirdParams(name, namespace, image, schedule, input, compression, encryption, output string) *CopyBirdParams {
	return &CopyBirdParams{
		Name:        name,
		Namespace:   namespace,
		Image:       image,
		Schedule:    schedule,
		Input:       input,
		Compression: compression,
		Encryption:  encryption,
		Output:      output,
	}
}

func (p *CopyBirdParams) MakeCronJob(ctx context.Context) *v1beta1.CronJob {
	return &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		Spec: v1beta1.CronJobSpec{
			Schedule: p.Schedule,
			JobTemplate: v1beta1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: p.Name,
				},
				Spec: v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: p.Name,
						},
						Spec: corev1.PodSpec{
							RestartPolicy: "OnFailure",
							Containers: []corev1.Container{
								corev1.Container{
									Name:  p.Name,
									Image: p.Image,
									Env: []corev1.EnvVar{
										{Name: "INPUT", Value: p.Input},
										{Name: "OUTPUT", Value: p.Output},
										{Name: "COMPRESSION", Value: p.Compression},
										{Name: "ENCRYPTION", Value: p.Encryption},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
