package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to the kubeconfig file. If not specified, KUBECONFIG env variable is used.")
	duration := flag.Duration("duration", 1*time.Hour, "Duration for the operation")

	flag.Parse()

	if *duration <= 30*time.Second {
		panic("duration must be greater than 30 seconds")
	}
	if *kubeconfig == "" {
		*kubeconfig = os.Getenv("KUBECONFIG")
		if *kubeconfig == "" {
			panic("kubeconfig must be specified either via flag or KUBECONFIG env variable")
		}
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	ctx := context.Background()
	namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(fmt.Errorf("error listing namespaces: %w", err))
	}
	var errs []error
	for _, ns := range namespaceList.Items {
		fmt.Printf("Namespace: %s\n", ns.Name)
		if err := cleanupEvents(ctx, clientset, ns.Name, *duration); err != nil {
			errs = append(errs, fmt.Errorf("error cleaning up events in namespace %s: %w", ns.Name, err))
		}
	}
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		panic("errors occurred during cleanup")
	}
	fmt.Printf("Cleanup completed successfully.\n")
}

func cleanupEvents(ctx context.Context, clientset *kubernetes.Clientset, namespace string, duration time.Duration) error {
	eventsClient := clientset.CoreV1().Events(namespace)
	eventsList, err := eventsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing events: %w", err)
	}

	cutoffTime := time.Now().Add(-duration)
	var toDelete []string
	for _, event := range eventsList.Items {
		if event.LastTimestamp.Time.Before(cutoffTime) {
			toDelete = append(toDelete, event.Name)
		}
	}

	if len(toDelete) == 0 {
		return nil
	}
	fmt.Printf("Found %d events to delete in namespace %s\n", len(toDelete), namespace)
	for _, eventName := range toDelete {
		if err := eventsClient.Delete(ctx, eventName, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("error deleting event %s: %w", eventName, err)
		}
	}
	fmt.Printf("Deleted %d events in namespace %s\n", len(toDelete), namespace)
	return nil
}
