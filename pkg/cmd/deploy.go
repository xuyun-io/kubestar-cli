package cmd

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xuyun-io/kubestar-cli/pkg/components"
	"github.com/xuyun-io/kubestar-cli/pkg/k8s"
	"github.com/xuyun-io/kubestar-cli/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	v1 "k8s.io/api/core/v1"
	"os"
)

const (
	DeploySuccess = "successfulDeploy"
)

func init() {
	DeployCmd.Flags().BoolP("check", "c", true, "Check whether the cluster can run KubeStar")
	DeployCmd.Flags().BoolP("check_only", "", false, "Only run check and exit.")
	DeployCmd.Flags().StringP("namespace", "n", "kubestar", "The namespace to deploy KubeStar to")
}

// DeployCmd is the "deploy" command.

var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys KubeStar on the current K8s cluster",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("check", cmd.Flags().Lookup("check"))
		viper.BindPFlag("check_only", cmd.Flags().Lookup("check_only"))
		viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		if cmd.Annotations["status"] != DeploySuccess {
			return
		}

		p := func(s string, a ...interface{}) {
			fmt.Fprintf(os.Stderr, s, a...)
		}
		b := color.New(color.Bold).Sprintf
		g := color.GreenString

		fmt.Fprint(os.Stderr, "\n")
		p(color.CyanString("==> ") + b("Next Steps:\n"))
		p("\nRun some scripts using the %s cli. For example: \n", g("ks"))
		p("- %s : to show pre-installed scripts.\n", g("ks script list"))
		p("- %s : to run service info for sock-shop demo application (service selection coming soon!).\n",
			g("ks run %s", "installation"))
	},
	Run: runDeployCmd,
}

func runDeployCmd(cmd *cobra.Command, args []string) {
	check, _ := cmd.Flags().GetBool("check")
	checkOnly, _ := cmd.Flags().GetBool("check_only")
	//namespace, _ := cmd.Flags().GetString("namespace")

	if check || checkOnly {
		err := utils.RunDefaultClusterChecks()
		if err != nil {
			utils.WithError(err).Fatal("Check pre-check has failed. To bypass pass in --check=false.")
		}
	}
	if checkOnly {
		log.Info("All Required Checks Passed!")
		os.Exit(0)
	}

	kubeConfig := k8s.GetConfig()
	kubeAPIConfig := k8s.GetClientAPIConfig()
	clientset := k8s.GetClientset(kubeConfig)

	clusterName, _ := cmd.Flags().GetString("cluster_name")
	if clusterName == "" {
		clusterName = kubeAPIConfig.CurrentContext

	}

	currentCluster := kubeAPIConfig.CurrentContext
	utils.Infof("Deploying KubeStar to the following cluster: %s", currentCluster)
	clusterOk := components.YNPrompt("Is the cluster correct?", true)
	if !clusterOk {
		utils.Error("Cluster is not correct. Aborting.")
		return
	}

	numNodes, err := getNumNodes(clientset)
	if err != nil {
		utils.Error(err.Error())
	}
	if numNodes == 0 {
		utils.Error("Cluster has no nodes. Try deploying KubeStar to a cluster with at least one node.")
		return
	}

	utils.Infof("Found %v nodes", numNodes)

}

func getNumNodes(clientset *kubernetes.Clientset) (int, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	unscheduleableNodes := 0
	for _, n := range nodes.Items {
		for _, t := range n.Spec.Taints {
			if !pemCanScheduleWithTaint(&t) {
				unscheduleableNodes++
				break
			}
		}
	}
	return len(nodes.Items) - unscheduleableNodes, nil
}

func pemCanScheduleWithTaint(t *v1.Taint) bool {
	// For now an effect of NoSchedule should be sufficient, we don't have tolerations in the Daemonset spec.
	return t.Effect != "NoSchedule"
}
