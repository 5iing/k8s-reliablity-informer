package checker

import (
	"context"
	"fmt"
	"time"

	"github.com/5iing/k3s-reliablity-informer/pkg/config"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type HealthChecker struct {
	client kubernetes.Interface
	factory informers.SharedInformerFactory
	config config.AppConfig
	notifier Notifier
	alertHistory map[string]time.Time
}

type Alert struct {
	Level string 
	Resource string 
	Name string 
	Message string
}

const (
	Synced    = "Synced"
	ErrExited = "ErrResourceExists"
)

type Notifier interface {
	Notifiy(message string) error
}

func NewHealthChecker(
	ctx context.Context,
	client kubernetes.Interface,
	config config.AppConfig,
	notifier Notifier,
) *HealthChecker {

	return &HealthChecker{
		client: client,
		factory: informers.NewSharedInformerFactory(client, 30*time.Second),
		config: config,
		notifier: notifier,
		alertHistory: make(map[string]time.Time),
	}

}

func (hc *HealthChecker) Start(ctx context.Context) error {
	if hc.config.Checker.CheckPods {
		_, err := hc.factory.Core().V1().Pods().Informer().AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(old, new interface{}) {
					hc.checkPod(new.(*corev1.Pod))
				},	
			},
		)
		if err != nil {
			return fmt.Errorf("failed to add pod event handler: %w", err)
		}
	}

	if hc.config.Checker.CheckNodes {
		_, err := hc.factory.Core().V1().Nodes().Informer().AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(old, new interface{}) {
					hc.checkNode(new.(*corev1.Node))
				},
			})
		if err != nil {
			return fmt.Errorf("failed to add node event handler: %w", err)
		}
	}

	if hc.config.Checker.CheckDeployments {
		_, err := hc.factory.Apps().V1().Deployments().Informer().AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(old, new interface{}) {
					hc.checkDeployment(new.(*appsv1.Deployment))
				},
			})
		if err != nil {
			return fmt.Errorf("failed to add deployment event handler: %w", err)
		}
	}

	hc.factory.Start(ctx.Done())
	hc.factory.WaitForCacheSync(ctx.Done())

	fmt.Println("Health checker succesfully enabled")
	return nil
}

func (hc *HealthChecker) checkPod(pod *corev1.Pod) {
	//pod failed 
	if pod.Status.Phase == corev1.PodFailed {
		hc.sendAlert(Alert{
			Level:    "error",
			Resource: "pod",
			Name:     fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
			Message:  fmt.Sprintf("Pod failed: %s", pod.Status.Reason),
		})
	}

	for _, cs := range pod.Status.ContainerStatuses {
		// crashloopfallback 
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			hc.sendAlert(Alert{
				Level: "error",
				Resource: "pod",
				Name: fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Message:  "Container is in CrashLoopBackOff",
			})
		}

		// restart 
		if cs.RestartCount > 5 {
			hc.sendAlert(Alert{
				Level:    "warning",
				Resource: "pod",
				Name:     fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Message:  fmt.Sprintf("High restart count: %d", cs.RestartCount),
			})
		}

		// pod waiting or image pull back of ffff
		if cs.State.Waiting != nil && 
		(cs.State.Waiting.Reason == "ImagePullBackOff" || cs.State.Waiting.Reason == "ErrImagePull") {
		 hc.sendAlert(Alert{
			 Level:    "error",
			 Resource: "pod",
			 Name:     fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
			 Message:  fmt.Sprintf("Image pull failed: %s", cs.State.Waiting.Reason),
		 })
		}
	}
}

func (hc *HealthChecker) checkNode(node *corev1.Node) {
	for _, cond := range node.Status.Conditions {
		// not ready 
		if cond.Type == corev1.NodeReady && cond.Status != corev1.ConditionTrue {
			hc.sendAlert(Alert{
				Level:    "critical",
				Resource: "node",
				Name:     node.Name,
				Message:  fmt.Sprintf("Node is not ready: %s", cond.Reason),
			})
		}

		// mem pressure
		if cond.Type == corev1.NodeMemoryPressure && cond.Status == corev1.ConditionTrue {
			hc.sendAlert(Alert{
				Level:    "warning",
				Resource: "node",
				Name:     node.Name,
				Message:  "Node has memory pressure",
			})
		}

		// disk pressure
		if cond.Type == corev1.NodeDiskPressure && cond.Status == corev1.ConditionTrue {
			hc.sendAlert(Alert{
				Level:    "warning",
				Resource: "node",
				Name:     node.Name,
				Message:  "Node has disk pressure",
			})
		}
	}
}

func (hc *HealthChecker) checkDeployment(deploy *appsv1.Deployment) {
	if deploy.Spec.Replicas == nil {
		return
	}

	desired := *deploy.Spec.Replicas
	available := deploy.Status.AvailableReplicas

	if available < desired {
		hc.sendAlert(Alert{
			Level:    "warning",
			Resource: "deployment",
			Name:     fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name),
			Message:  fmt.Sprintf("Replicas not ready: %d/%d available", available, desired),
		})
	}
}

func (hc *HealthChecker) sendAlert(alert Alert) {
	alertKey := fmt.Sprintf("%s:%s:%s", alert.Level, alert.Resource, alert.Name)
	now := time.Now()
	
	if lastAlert, exists := hc.alertHistory[alertKey]; exists {
		if now.Sub(lastAlert) < 5*time.Minute {
			return
		}
	}
	
	hc.alertHistory[alertKey] = now
	
	emoji := map[string]string{
		"warning":  "⚠️",
		"error":    "❌",
		"critical": "🚨",
	}

	msg := fmt.Sprintf("%s [%s] %s: %s",
		emoji[alert.Level],
		alert.Resource,
		alert.Name,
		alert.Message,
	)

	fmt.Println(msg)

	if hc.notifier != nil {
		if err := hc.notifier.Notifiy(msg); err != nil {
			fmt.Printf("Failed to send notification: %v\n", err)
		}
	}
}