package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	Kubeconfig string
	Duration   time.Duration
	QPS        float64
	Burst      int
	Retries    int
	DryRun     bool
	Statistics *Statistics
}

type Statistics struct {
	TotalEvents       int
	DeletedEvents     int
	NamespacesScanned int
}

func main() {
	cfg := &Config{
		Statistics: &Statistics{},
	}
	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file. If not specified, KUBECONFIG env variable is used. Use 'in-cluster' for in-cluster configuration.")
	flag.DurationVar(&cfg.Duration, "duration", 1*time.Hour, "Duration for the operation")
	flag.Float64Var(&cfg.QPS, "qps", 200, "Kubernetes client QPS")
	flag.IntVar(&cfg.Burst, "burst", 50, "Kubernetes client Burst")
	flag.IntVar(&cfg.Retries, "retries", 2, "Number of retries for Kubernetes client operations")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "If true, no changes will be made")
	flag.Parse()

	if cfg.Duration < 30*time.Second {
		panic("duration must be greater or equal than 30 seconds")
	}
	fmt.Printf("Starting cleanup of events older than %s\n", cfg.Duration.String())
	if cfg.DryRun {
		fmt.Printf("Dry run mode enabled, no events will be deleted.\n")
	}

	clientset, err := createClientSet(cfg)
	if err != nil {
		panic(err.Error())
	}

	ctx := context.Background()
	if err := cleanupAllEvents(ctx, clientset, cfg); err != nil {
		panic(err.Error())
	}
}

func cleanupAllEvents(ctx context.Context, clientset *kubernetes.Clientset, cfg *Config) error {
	namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing namespaces: %w", err)
	}
	for _, ns := range namespaceList.Items {
		fmt.Printf("Namespace: %s\n", ns.Name)
		if err := cleanupEvents(ctx, clientset, ns.Name, cfg); err != nil {
			fmt.Printf("error cleaning up events in namespace %s: %w", ns.Name, err)
		}
		cfg.Statistics.NamespacesScanned++
	}
	mode := "Deleted"
	msg := "Cleanup completed successfully.\n"
	if cfg.DryRun {
		mode = "To be deleted"
		msg = "Dry run completed successfully.\n"
	}
	fmt.Printf(msg)
	fmt.Printf("Statistics:\n")
	fmt.Printf("  Namespaces scanned: %d\n", cfg.Statistics.NamespacesScanned)
	fmt.Printf("  Total events: %d\n", cfg.Statistics.TotalEvents)
	fmt.Printf("  %s events: %d\n", mode, cfg.Statistics.DeletedEvents)
	fmt.Printf("  Retained events: %d\n", cfg.Statistics.TotalEvents-cfg.Statistics.DeletedEvents)

	return nil
}

func createClientSet(cfg *Config) (*kubernetes.Clientset, error) {
	kubeconfig := cfg.Kubeconfig
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	var config *rest.Config
	var err error
	if kubeconfig == "in-cluster" {
		fmt.Printf("Using in-cluster configuration\n")
		config, err = rest.InClusterConfig()
	} else if kubeconfig == "" {
		fmt.Printf("KUBECONFIG not specified, trying in-cluster configuration\n")
		config, err = rest.InClusterConfig()
	} else {
		fmt.Printf("Using kubeconfig: %s\n", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		panic(err.Error())
	}

	// Increase QPS and Burst to handle large number of requests
	config.QPS = float32(cfg.QPS)
	config.Burst = cfg.Burst

	return kubernetes.NewForConfig(config)
}

func cleanupEvents(ctx context.Context, clientset *kubernetes.Clientset, namespace string, cfg *Config) error {
	eventsClient := clientset.CoreV1().Events(namespace)
	var eventsList *corev1.EventList
	if err := opWithRetries(func() error {
		var listErr error
		eventsList, listErr = eventsClient.List(ctx, metav1.ListOptions{})
		return listErr
	}, cfg.Retries); err != nil {
		return fmt.Errorf("error listing events: %w", err)
	}

	cutoffTime := time.Now().Add(-cfg.Duration)
	var toDelete []string
	for _, event := range eventsList.Items {
		if event.CreationTimestamp.Time.Before(cutoffTime) && event.LastTimestamp.Time.IsZero() {
			toDelete = append(toDelete, event.Name)
		} else if event.LastTimestamp.Time.Before(cutoffTime) {
			toDelete = append(toDelete, event.Name)
		}
	}

	cfg.Statistics.TotalEvents += len(eventsList.Items)
	cfg.Statistics.DeletedEvents += len(toDelete)
	if len(toDelete) == 0 {
		fmt.Printf("No events to delete in namespace %s (total: %d events)\n", namespace, len(eventsList.Items))
		return nil
	}
	fmt.Printf("Found %d events to delete in namespace %s (total: %d events)\n", len(toDelete), namespace, len(eventsList.Items))
	if cfg.DryRun {
		return nil
	}
	for i, eventName := range toDelete {
		if err := opWithRetries(func() error {
			err := eventsClient.Delete(ctx, eventName, metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
			return nil
		}, cfg.Retries); err != nil {
			return fmt.Errorf("error deleting event %s: %w", eventName, err)
		}
		if (i+1)%500 == 0 {
			fmt.Printf("  Deleted %d/%d events in namespace %s\n", i+1, len(toDelete), namespace)
		}
	}
	fmt.Printf("Deleted %d events in namespace %s\n", len(toDelete), namespace)
	return nil
}

func opWithRetries(op func() error, retries int) error {
	var err error
	for i := 0; i <= retries; i++ {
		err = op()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i+1) * 50 * time.Millisecond)
	}
	return err
}
