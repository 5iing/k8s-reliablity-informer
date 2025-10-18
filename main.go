package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/5iing/k3s-reliablity-informer/pkg/checker"
	"github.com/5iing/k3s-reliablity-informer/pkg/config"
)

func main() {
	configFile := flag.String("config", "pkg/config/config.yaml", "path to config file")
	flag.Parse()

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	appConfig, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	hc := checker.NewHealthChecker(ctx, client, *appConfig, nil)

	fmt.Println(" Starting K8s Health Checker")
	fmt.Printf("   Pods: %v\n", appConfig.Checker.CheckPods)
	fmt.Printf("   Nodes: %v\n", appConfig.Checker.CheckNodes)
	fmt.Printf("   Deployments: %v\n", appConfig.Checker.CheckDeployments)

	if err := hc.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting health checker: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Health checker started successfully")

	<-ctx.Done()
	fmt.Println("\nShutting down...")
}