package k8s

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/chaos-io/chaos/core"
	"github.com/chaos-io/chaos/logs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/remotecommand"
)

func StartSpecPod(ctx context.Context, pod *corev1.Pod) error {
	logger := logs.With("pod", pod.Name)

	podsClient := clientSet.CoreV1().Pods(k8sCfg.Namespace)
	result, err := podsClient.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	logger.Debugw("created pod", "status", result.Status.Phase)

	defer func() {
		if err = podsClient.Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			logger.Debugw("failed tp delete pod")
			return
		}
		logger.Debugw("deleted pod")
	}()

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
		p, err := podsClient.Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, c := range p.Status.ContainerStatuses {
			if c.Name == pod.Name {
				if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
					return true, nil
				}
				logger.Debugw("get pod status", "status", p.Status.Phase)
				if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
					return true, nil
				}
			}
		}

		return false, nil
	})
	if err != nil {
		return err
	}
	logger.Debugw("pod execution completed")

	return nil
}

func StartPod(ctx context.Context, namespace, podName string, cmd []string, options core.Options, bindPaths ...string) (string, error) {
	logger := logs.With("pod", podName)

	var envVars []corev1.EnvVar
	if env, ok := options[OptionEnv].([]string); ok {
		for _, v := range env {
			_v := strings.Split(v, "=")
			if len(_v) != 2 {
				continue
			}
			envVars = append(envVars, corev1.EnvVar{
				Name:  _v[0],
				Value: _v[1],
			})
		}
	}

	var resources corev1.ResourceRequirements
	if memoryLimit, ok := options[OptionMemoryLimit].(string); ok {
		resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				// corev1.ResourceMemory: resource.MustParse(memoryLimit),
				// corev1.ResourceCPU: resource.MustParse("1000m"), // 1 CPU core
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse(memoryLimit),
				corev1.ResourceCPU:    resource.MustParse("2000m"),
			},
		}
	}

	volumeMounts, volume := podVolumes(bindPaths...)
	callbackPod := CallbackPod(podName, envVars, cmd, volumeMounts, volume, resources)

	start := time.Now()
	result, err := clientSet.CoreV1().Pods(namespace).Create(ctx, callbackPod, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	logger.Debugw("created pod")

	if err := WaitPodRunning(ctx, namespace, podName); err != nil {
		logger.Warnw("failed to run pod while wait for running", "status", result.Status.Phase)
		return "", err
	}

	go WaitPod(ctx, podName, namespace, func(status string, err error, duration time.Duration) {
		logger.Infow("pod execute completed", "status", status, "duration", duration.String())
		// TODO: add something
	})

	logger.Debugw("start pod successfully", "duration", time.Since(start).String())
	return result.Name, nil
}

func podVolumes(bindPaths ...string) ([]corev1.VolumeMount, []corev1.Volume) {
	hostPath := ""
	volumeMounts := make([]corev1.VolumeMount, 0)
	volumes := make([]corev1.Volume, 0)
	for i, bindPath := range bindPaths {
		if i%2 == 0 {
			hostPath = bindPath
		} else {
			volumeName := fmt.Sprintf("%v-volume", GetEncodeString(bindPath))
			volumeMounts = append(
				volumeMounts, corev1.VolumeMount{
					Name:      volumeName,
					ReadOnly:  false,
					MountPath: bindPath,
				},
			)

			volumes = append(
				volumes, corev1.Volume{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						NFS: &corev1.NFSVolumeSource{
							Server: k8sCfg.NFSServer,
							Path:   path.Join(k8sCfg.NFSPath, hostPath),
						},
					},
				},
			)
		}
	}

	return volumeMounts, volumes
}

func PausePod(ctx context.Context, namespace, podName string) error {
	cmd := []string{"/bin/sh", "-c", "kill -STOP -1"}
	_, err := RunCMDInPod(ctx, namespace, podName, cmd...)
	if err != nil {
		return err
	}
	logs.Infow("pause pod successfully", "pod", podName, "command", cmd)
	return nil
}

func ResumePod(ctx context.Context, namespace, podName string) error {
	cmd := []string{"/bin/sh", "-c", "kill -CONT -1"}
	_, err := RunCMDInPod(ctx, namespace, podName, cmd...)
	if err != nil {
		return err
	}
	logs.Infow("resume pod successfully", "pod", podName, "command", cmd)
	return nil
}

func RunCMDInPod(ctx context.Context, namespace, podName string, cmd ...string) (string, error) {
	req := clientSet.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "false")

	for _, c := range cmd {
		req.Param("command", c)
	}

	executor, err := remotecommand.NewSPDYExecutor(kubeConf, "POST", req.URL())
	if err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer
	streamOptions := remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	}

	err = executor.StreamWithContext(ctx, streamOptions)
	if err != nil {
		return "", err
	}

	if len(strings.TrimSpace(stderr.String())) > 0 {
		return "", fmt.Errorf("error: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func ProcessIsInPod(ctx context.Context, namespace, podName, processName string) bool {
	cmd := []string{"sh", "-c", fmt.Sprintf("ps -ef | grep %q | grep -v grep", processName)}
	output, err := RunCMDInPod(ctx, namespace, podName, cmd...)
	if err != nil {
		return false
	}
	logs.Debugw("get process in pod", "pod", podName, "command", cmd, "output", output)
	return true
}

func InspectPod(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	podsClient := clientSet.CoreV1().Pods(namespace)
	p, err := podsClient.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// pod, _ := jsoniter.MarshalToString(p)
	logs.Debugw("inspect pod", "pod", podName, "status", p.Status)
	return p, nil
}

func InspectPodIP(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := InspectPod(ctx, namespace, podName)
	if err != nil {
		return "", err
	}
	logs.Infow("inspect pod ip", "pod", podName, "ip", pod.Status.PodIP)
	return pod.Status.PodIP, nil
}

func RemovePod(ctx context.Context, namespace, podName string) error {
	podsClient := clientSet.CoreV1().Pods(namespace)
	// deletePolicy := metav1.DeletePropagationForeground
	deletePolicy := metav1.DeletePropagationBackground
	err := podsClient.Delete(ctx, podName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		return err
	}
	logs.Infow("deleted pod", "pod", podName)
	return nil
}

func WaitPod(ctx context.Context, namespace, podName string, callback func(status string, err error, duration time.Duration)) {
	if callback == nil {
		return
	}

	start := time.Now()
	status := ""
	podsClient := clientSet.CoreV1().Pods(namespace)
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 300*time.Second, true, func(ctx context.Context) (done bool, err error) {
		p, err := podsClient.Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, c := range p.Status.ContainerStatuses {
			if c.Name == podName {
				status = string(p.Status.Phase)
				// Check if the container is in CrashLoopBackOff state
				if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
					return true, nil
				}
				// Check if the pod is in Completed state
				logs.Debugw("get pod status", "pod", podName, "status", p.Status.Phase)
				if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
					return true, nil
				}
			}
		}

		return false, nil
	})

	callback(status, err, time.Since(start))
}

func WaitPodRunning(ctx context.Context, namespace, podName string) error {
	return wait.PollUntilContextTimeout(ctx, 1*time.Second, 60*time.Second, true, func(ctx context.Context) (done bool, err error) {
		pod, err := clientSet.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			logs.Warnw("failed to get pod when wait pod running", "pod", podName, "error", err)
			return false, err
		}

		if pod.Status.Phase == corev1.PodRunning {
			logs.Infow("pod is running", "pod", podName)
			return true, nil
		}

		return false, nil
	})
}
