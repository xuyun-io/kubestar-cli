package cmd

import (
	"context"
	"github.com/xuyun-io/kubestar-cli/pkg/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"sort"

	v1 "k8s.io/api/core/v1"
	sv1 "k8s.io/api/storage/v1"
)

func getOKNodes(clientset *kubernetes.Clientset) ([]string, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return []string{}, err
	}

	displayNodes(nodes.Items)

	var nodeNames []string
	for _, n := range nodes.Items {
		if pemCanScheduleWithTaint(n) {
			nodeNames = append(nodeNames, n.GetName())
		}
	}
	sort.Strings(nodeNames)
	return nodeNames, nil
}

func pemCanScheduleWithTaint(n v1.Node) bool {
	for _, t := range n.Spec.Taints {
		if t.Effect == "NoSchedule" {
			return false
		}
	}
	// For now an effect of NoSchedule should be sufficient, we don't have tolerations in the Daemonset spec.
	return true
}

func ListStorageClass(clientset *kubernetes.Clientset) ([]string, error) {
	scList, err := clientset.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return []string{}, err
	}

	displayStorageClass(scList.Items)
	var scNames []string
	for _, sc := range scList.Items {
		scNames = append(scNames, sc.GetName())
	}

	sort.Strings(scNames)
	return scNames, nil
}

// nodeName,cpu, memory, pods
func displayNodes(l []v1.Node) {
	w := components.CreateStreamWriter("Nodes ", os.Stdout)
	w.SetHeader("Node List", []string{"NAME", "CPU", "MEM", "ephemeral-STORAGE", "PODS"})

	for _, item := range l {
		w.Write([]interface{}{
			item.GetName(),
			item.Status.Allocatable.Cpu(),
			item.Status.Allocatable.Memory(),
			item.Status.Allocatable.StorageEphemeral(),
			item.Status.Allocatable.Pods(),
		})
	}

	w.Finish()
}

func displayStorageClass(l []sv1.StorageClass) {
	w := components.CreateStreamWriter("StorageClass ", os.Stdout)
	w.SetHeader("StorageClass List", []string{"NAME", "PROVISIONER", "RECLAIMPOLICY", "VOLUMEBINDINGMODE"})

	for _, item := range l {
		w.Write([]interface{}{
			item.GetName(),
			item.Provisioner,
			*item.ReclaimPolicy,
			*item.VolumeBindingMode,
		})
	}
	w.Finish()
}
