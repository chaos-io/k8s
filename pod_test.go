package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ctx = context.Background()
	tt  = struct {
		ns  string
		pod *corev1.Pod
	}{
		ns:  corev1.NamespaceDefault,
		pod: MockAutoharnessPod(),
	}
)

func TestStartPod(t *testing.T) {
	err := StartSpecPod(ctx, tt.pod)
	assert.NoError(t, err)
}

func TestPausePod(t *testing.T) {
	err := PausePod(ctx, tt.pod.Name, tt.ns)
	assert.NoError(t, err)
}

func TestResumePod(t *testing.T) {
	err := ResumePod(ctx, tt.pod.Name, tt.ns)
	assert.NoError(t, err)
}

func TestProcessIsInPod(t *testing.T) {
	found := ProcessIsInPod(ctx, tt.pod.Name, tt.ns, "top -b")
	assert.Equal(t, true, found)
}

func TestInspectPod(t *testing.T) {
	p, err := InspectPod(ctx, tt.pod.Name, tt.ns)
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestInspectPodIP(t *testing.T) {
	ip, err := InspectPodIP(ctx, tt.pod.Name, tt.ns)
	assert.NoError(t, err)
	assert.NotEmpty(t, ip)
}

func TestRemovePod(t *testing.T) {
	err := RemovePod(ctx, tt.pod.Name, tt.ns)
	assert.NoError(t, err)
}

func TestWaitPod(t *testing.T) {
	cb := func(status string, err error, duration time.Duration) {
		fmt.Printf("status=%v, err=%v, dur=%v\n", status, err, duration.String())
	}
	WaitPod(ctx, tt.pod.Name, tt.ns, cb)
	fmt.Printf("----wait pod done, name=%v\n", tt.pod.Name)
}

func MockAutoharnessPod() *corev1.Pod {
	// var sec int64 = 100
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "autoharness",
			Labels: map[string]string{
				"app": "autoharness",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			// TerminationGracePeriodSeconds: &sec,
			Containers: []corev1.Container{
				{
					Name:    "autoharness",
					Image:   "autoharness:latest",
					Command: []string{"/bin/sh", "-c", "for i in $(seq 1 10000); do sleep 1 && echo ${i}; done"},
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
							Server: "192.168.5.83",
							Path:   "/data/nfs/default/data",
						},
					},
				},
			},
		},
	}
}
