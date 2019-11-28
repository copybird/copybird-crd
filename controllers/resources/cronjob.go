package resources

import (
	"context"
	"fmt"
	"strings"

	backupv1alpha1 "github.com/copybird/copybird-crd/api/v1alpha1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	inputEnv    = "COPYBIRD_INPUT"
	outputEnv   = "COPYBIRD_OUTPUT"
	compressEnv = "COPYBIRD_COMPRESS"
	encryptEnv  = "COPYBIRD_ENCRYPT"
)

type CopyBirdParams struct {
	Image  string
	Backup *backupv1alpha1.Backup
}

func NewCopyBirdParams(image string, backup *backupv1alpha1.Backup) *CopyBirdParams {
	return &CopyBirdParams{
		Image:  image,
		Backup: backup,
	}
}

func (p *CopyBirdParams) MakeCronJob(ctx context.Context) *v1beta1.CronJob {
	env := []corev1.EnvVar{
		{
			Name:  inputEnv,
			Value: p.Backup.Spec.Input.Type,
		}, {
			Name:  outputEnv,
			Value: p.Backup.Spec.Output.Type,
		}, {
			Name:  encryptEnv,
			Value: p.Backup.Spec.Encrypt.Type,
		}, {
			Name:  compressEnv,
			Value: p.Backup.Spec.Compress.Type,
		},
	}
	env = append(env, parseParams(p.Backup.Spec.Input.Params, inputEnv)...)
	env = append(env, parseSecrets(p.Backup.Spec.Input.Secrets, inputEnv)...)
	env = append(env, parseParams(p.Backup.Spec.Output.Params, outputEnv)...)
	env = append(env, parseSecrets(p.Backup.Spec.Output.Secrets, outputEnv)...)
	env = append(env, parseParams(p.Backup.Spec.Compress.Params, compressEnv)...)
	env = append(env, parseSecrets(p.Backup.Spec.Compress.Secrets, compressEnv)...)
	env = append(env, parseParams(p.Backup.Spec.Encrypt.Params, encryptEnv)...)
	env = append(env, parseSecrets(p.Backup.Spec.Encrypt.Secrets, encryptEnv)...)
	return &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Backup.Name,
			Namespace: p.Backup.Namespace,
		},
		Spec: v1beta1.CronJobSpec{
			Schedule: p.Backup.Spec.Schedule,
			JobTemplate: v1beta1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: p.Backup.Name,
				},
				Spec: v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: p.Backup.Name,
						},
						Spec: corev1.PodSpec{
							RestartPolicy: "OnFailure",
							Containers: []corev1.Container{
								corev1.Container{
									Name:  p.Backup.Name,
									Image: p.Image,
									// docker entrypoint should work,
									// but Args being ignored without Command for some reason
									Command: []string{"/copybird"},
									Args:    []string{"backup"},
									Env:     env,
								},
							},
						},
					},
				},
			},
		},
	}
}

func parseParams(params []backupv1alpha1.ModuleParam, prefix string) []corev1.EnvVar {
	var env []corev1.EnvVar
	for _, v := range params {
		env = append(env, corev1.EnvVar{
			Name:  fmt.Sprintf("%s_%s", prefix, strings.ToUpper(v.Key)),
			Value: v.Value,
		})
	}
	return env
}

func parseSecrets(secrets []backupv1alpha1.ModuleSecret, prefix string) []corev1.EnvVar {
	var env []corev1.EnvVar
	for _, v := range secrets {
		env = append(env, corev1.EnvVar{
			Name: fmt.Sprintf("%s_%s", prefix, strings.ToUpper(v.SecretKeyRef.Key)),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: v.SecretKeyRef,
			},
		})
	}
	return env
}
