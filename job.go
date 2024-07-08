package k8s

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/chaos-io/chaos/logs"
	"github.com/segmentio/ksuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func StartJob(ctx context.Context, job *batchv1.Job) error {
	jobsClient := clientSet.BatchV1().Jobs(k8sCfg.Namespace)
	_, err := jobsClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		logs.Warnw("failed to create job", "job", job.Name, "error", err)
		return err
	}

	logs.Debugw("created job", "name", job.Name)

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 300*time.Second, true, func(ctx context.Context) (done bool, err error) {
		p, err := jobsClient.Get(ctx, job.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, c := range p.Status.Conditions {
			logs.Debugw("get job status", "job", job.Name, "status", c.Type)
			if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		return err
	}

	logs.Debugw("job execution completed", "job", job.Name)
	return nil
}

func AutoharnessJob(relPath string) *batchv1.Job {
	logs.Debugw("pod mount path", "path", path.Join(k8sCfg.NFSPath, relPath))

	var ttlSeconds int32 = 10 // auto remove job after ttlSeconds
	name := "autoharness-" + strings.ToLower(ksuid.New().String()[:8])
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "autoharness",
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttlSeconds,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: k8sCfg.ImagePullSecret}},
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           kubeImage(imageAutoharness),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c", "python3 /opt/631/python/main.py /src/prototype.json"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "prototype-volume",
									MountPath: "/src",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "prototype-volume",
							VolumeSource: corev1.VolumeSource{
								NFS: &corev1.NFSVolumeSource{
									Server: k8sCfg.NFSServer,
									Path:   path.Join(k8sCfg.NFSPath, relPath),
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}
