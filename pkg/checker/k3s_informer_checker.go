package checker

import "k8s.io/client-go/kubernetes"

type HealthChecker struct {
	client *kubernetes.Clientset
}

type HealthCheckerMetadata struct {
}
