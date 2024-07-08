package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	imageAutoharness = "autoharness"
	imageCallback    = "callback-A"
)

func CallbackPod(podName string, env []corev1.EnvVar, cmd []string, volumeMounts []corev1.VolumeMount, volume []corev1.Volume, resources corev1.ResourceRequirements) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: k8sCfg.Namespace,
			Labels: map[string]string{
				"app": "callback-",
			},
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: k8sCfg.ImagePullSecret}},
			Containers: []corev1.Container{
				{
					Name:            podName,
					Image:           kubeImage(imageCallback),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env:             env,
					Command:         cmd,
					VolumeMounts:    volumeMounts,
					Resources:       resources,
				},
			},
			Volumes: volume,
		},
	}
}
