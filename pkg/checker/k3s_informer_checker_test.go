package checker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/5iing/k3s-reliablity-informer/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type MockNotifier struct {
	alerts []Alert
}

func (m *MockNotifier) Notifiy(message string) error {
	alert := Alert{
		Level:    "test",
		Resource: "test",
		Name:     "test",
		Message:  message,
	}
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *MockNotifier) GetAlerts() []Alert {
	return m.alerts
}

func (m *MockNotifier) ClearAlerts() {
	m.alerts = []Alert{}
}

func TestNewHealthChecker(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	config := config.AppConfig{
		Checker: struct {
			CheckPods        bool `yaml:"check_pods"`
			CheckNodes       bool `yaml:"check_nodes"`
			CheckDeployments bool `yaml:"check_deployments"`
		}{
			CheckPods:        true,
			CheckNodes:       true,
			CheckDeployments: true,
		},
	}
	notifier := &MockNotifier{}

	hc := NewHealthChecker(ctx, client, config, notifier)

	if hc == nil {
		t.Fatal("Expected HealthChecker to be created")
	}
	if hc.client != client {
		t.Error("Expected client to be set")
	}
	if hc.config != config {
		t.Error("Expected config to be set")
	}
	if hc.notifier != notifier {
		t.Error("Expected notifier to be set")
	}
	if hc.alertHistory == nil {
		t.Error("Expected alertHistory to be initialized")
	}
}

func TestHealthChecker_Start(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := fake.NewSimpleClientset()
	config := config.AppConfig{
		Checker: struct {
			CheckPods        bool `yaml:"check_pods"`
			CheckNodes       bool `yaml:"check_nodes"`
			CheckDeployments bool `yaml:"check_deployments"`
		}{
			CheckPods:        true,
			CheckNodes:       true,
			CheckDeployments: true,
		},
	}
	notifier := &MockNotifier{}

	hc := NewHealthChecker(ctx, client, config, notifier)

	err := hc.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

func TestHealthChecker_checkPod(t *testing.T) {
	notifier := &MockNotifier{}
	hc := &HealthChecker{
		notifier:     notifier,
		alertHistory: make(map[string]time.Time),
	}

	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected int
	}{
		{
			name: "CrashLoopBackOff pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "High restart count pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount: 10,
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "ImagePullBackOff pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "image-pull-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "ImagePullBackOff",
								},
							},
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Failed pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failed-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: "ContainerCannotRun",
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount: 1,
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Healthy pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount: 0,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier.ClearAlerts()
			hc.checkPod(tt.pod)
			
			if len(notifier.GetAlerts()) != tt.expected {
				t.Errorf("Expected %d alerts, got %d", tt.expected, len(notifier.GetAlerts()))
				for i, alert := range notifier.GetAlerts() {
					t.Logf("Alert %d: %+v", i, alert)
				}
			}
		})
	}
}

func TestHealthChecker_checkNode(t *testing.T) {
	notifier := &MockNotifier{}
	hc := &HealthChecker{
		notifier:     notifier,
		alertHistory: make(map[string]time.Time),
	}

	tests := []struct {
		name     string
		node     *corev1.Node
		expected int
	}{
		{
			name: "Not ready node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
							Reason: "KubeletNotReady",
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Memory pressure node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "memory-pressure-node",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeMemoryPressure,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Disk pressure node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "disk-pressure-node",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeDiskPressure,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Healthy node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "healthy-node",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier.ClearAlerts()
			hc.checkNode(tt.node)
			
			alerts := notifier.GetAlerts()
			if len(alerts) != tt.expected {
				t.Errorf("Expected %d alerts, got %d", tt.expected, len(alerts))
				for i, alert := range alerts {
					t.Logf("Alert %d: %+v", i, alert)
				}
				if tt.name == "Disk pressure node" {
					t.Logf("Node conditions: %+v", tt.node.Status.Conditions)
				}
			}
		})
	}
}

func TestHealthChecker_checkDeployment(t *testing.T) {
	notifier := &MockNotifier{}
	hc := &HealthChecker{
		notifier:     notifier,
		alertHistory: make(map[string]time.Time),
	}

	replicas := int32(3)
	available := int32(1)

	tests := []struct {
		name     string
		deploy   *appsv1.Deployment
		expected int
	}{
		{
			name: "Deployment with insufficient replicas",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: available,
				},
			},
			expected: 1,
		},
		{
			name: "Deployment with sufficient replicas",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: replicas,
				},
			},
			expected: 0,
		},
		{
			name: "Deployment with nil replicas",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: available,
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier.ClearAlerts()
			hc.checkDeployment(tt.deploy)
			
			if len(notifier.GetAlerts()) != tt.expected {
				t.Errorf("Expected %d alerts, got %d", tt.expected, len(notifier.GetAlerts()))
				for i, alert := range notifier.GetAlerts() {
					t.Logf("Alert %d: %+v", i, alert)
				}
			}
		})
	}
}

func TestHealthChecker_sendAlert_DuplicatePrevention(t *testing.T) {
	notifier := &MockNotifier{}
	hc := &HealthChecker{
		notifier:     notifier,
		alertHistory: make(map[string]time.Time),
	}

	alert := Alert{
		Level:    "error",
		Resource: "pod",
		Name:     "test-pod",
		Message:  "Test message",
	}

	// Send first alert
	hc.sendAlert(alert)
	if len(notifier.GetAlerts()) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(notifier.GetAlerts()))
	}

	// Send duplicate alert immediately (should be prevented)
	hc.sendAlert(alert)
	if len(notifier.GetAlerts()) != 1 {
		t.Errorf("Expected 1 alert after duplicate, got %d", len(notifier.GetAlerts()))
	}

	// Wait for cooldown period and send again
	hc.alertHistory[fmt.Sprintf("%s:%s:%s", alert.Level, alert.Resource, alert.Name)] = time.Now().Add(-6 * time.Minute)
	hc.sendAlert(alert)
	if len(notifier.GetAlerts()) != 2 {
		t.Errorf("Expected 2 alerts after cooldown, got %d", len(notifier.GetAlerts()))
	}
}

func TestHealthChecker_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test objects
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionFalse,
					Reason: "KubeletNotReady",
				},
			},
		},
	}

	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}

	// Create fake client with test objects
	client := fake.NewSimpleClientset(pod, node, deployment)
	config := config.AppConfig{
		Checker: struct {
			CheckPods        bool `yaml:"check_pods"`
			CheckNodes       bool `yaml:"check_nodes"`
			CheckDeployments bool `yaml:"check_deployments"`
		}{
			CheckPods:        true,
			CheckNodes:       true,
			CheckDeployments: true,
		},
	}
	notifier := &MockNotifier{}

	hc := NewHealthChecker(ctx, client, config, notifier)

	err := hc.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	updatedPod := pod.DeepCopy()
	updatedPod.Status.ContainerStatuses[0].State.Waiting.Reason = "ImagePullBackOff"
	_, err = client.CoreV1().Pods("default").Update(ctx, updatedPod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update pod: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	alerts := notifier.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Expected alerts to be generated")
	}

	t.Logf("Generated %d alerts", len(alerts))
	for i, alert := range alerts {
		t.Logf("Alert %d: %+v", i, alert)
	}
}
