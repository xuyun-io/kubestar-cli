package cmd

import (
	"fmt"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xuyun-io/kubestar-cli/pkg/components"
	"github.com/xuyun-io/kubestar-cli/pkg/k8s"
	"github.com/xuyun-io/kubestar-cli/pkg/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
)

const (
	DeploySuccess = "successfulDeploy"
)

func init() {
	DeployCmd.Flags().BoolP("check", "c", true, "Check whether the cluster can run KubeStar")
	DeployCmd.Flags().BoolP("check_only", "", false, "Only run check and exit.")
	DeployCmd.Flags().StringP("namespace", "n", "kubestar", "The namespace to deploy KubeStar to")
	DeployCmd.Flags().StringP("yamls", "y", "yamls.tar", "The k8s resources yaml to install")
	DeployCmd.Flags().StringP("domain", "", "", "The kubestar domain used to access")
	DeployCmd.Flags().StringP("kubestar_image", "", "michaelpan/kubestar:20230220", "The kubestar image to deploy")
	DeployCmd.Flags().BoolP("monitor_only", "", false, "Only deploy monitor components")

}

// DeployCmd is the "deploy" command.
var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys KubeStar on the current K8s cluster",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("check", cmd.Flags().Lookup("check"))
		viper.BindPFlag("check_only", cmd.Flags().Lookup("check_only"))
		viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))
		viper.BindPFlag("yamls", cmd.Flags().Lookup("yamls"))
		viper.BindPFlag("domain", cmd.Flags().Lookup("domain"))
		viper.BindPFlag("kubestar_image", cmd.Flags().Lookup("kubestar_image"))
		viper.BindPFlag("monitor_only", cmd.Flags().Lookup("monitor_only"))
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
	namespace, _ := cmd.Flags().GetString("namespace")
	yamls, _ := cmd.Flags().GetString("yamls")
	domain, _ := cmd.Flags().GetString("domain")
	kubestarImage, _ := cmd.Flags().GetString("kubestar_image")
	monitorOnly, _ := cmd.Flags().GetBool("monitor_only")

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

	currentCluster := kubeAPIConfig.CurrentContext
	utils.Infof("Deploying KubeStar to the following cluster: %s", getClusterShortName(currentCluster))
	clusterOk := components.YNPrompt("Is the cluster correct?", true)
	if !clusterOk {
		utils.Error("Cluster is not correct. Aborting.")
		return
	}

	okNodes, err := getOKNodes(clientset)
	if err != nil {
		utils.Error(err.Error())
	}
	if len(okNodes) == 0 {
		utils.Error("Cluster has no nodes. Try deploying KubeStar to a cluster with at least one node.")
		return
	}
	utils.Infof("Found %v nodes", len(okNodes))

	storageClasses, err := ListStorageClass(clientset)
	if err != nil {
		utils.Error(err.Error())
		return
	}

	deployOpt := deployOptions{MonitorOnly: monitorOnly}
	if len(storageClasses) > 0 {
		utils.Infof("Found %d StorageClasses", len(storageClasses))
		deployOpt.ChooseStorageClassName = components.ChooseOne("Choose one StorageClass or press enter to ignore to install ", storageClasses)
	}

	if len(deployOpt.ChooseStorageClassName) == 0 && !monitorOnly {
		c := components.YNPrompt("As no StorageClass are supplied, KubeStar are using a local PV, Are you ok to continue?", true)
		if !c {
			utils.Error("No StorageClass exist. Aborting.")
			return
		}

		utils.Info("Please choose one node to supply the Local Persistent Volume(/data/kubestar/mysql): ")
		if deployOpt.ChooseNodeName = components.ChooseOne("Choose one Node to install ", okNodes); len(deployOpt.ChooseNodeName) == 0 {
			utils.Fatalf("You must choose one node.")
			return
		}
		utils.Infof("Node %s choosed.\n", deployOpt.ChooseNodeName)
	}

	// Fill in template values.
	tmplArgs := &utils.YAMLTmplArguments{
		Values: &map[string]interface{}{
			"deployKubeStar":   !monitorOnly,
			"deployMonitor":    monitorOnly,
			"KubeStarImage":    kubestarImage,
			"StorageClassName": deployOpt.ChooseStorageClassName,
		},
		Release: &map[string]interface{}{
			"Namespace":        namespace,
			"Domain":           domain,
			"SelectedNodeName": deployOpt.ChooseNodeName,
			"Cluster":          getClusterShortName(currentCluster),
		},
	}

	utils.Infof("Starting to load %s yaml resources", yamls)
	yamlData, err := utils.LoadTemplateYAMLs(yamls)
	if err != nil {
		utils.Errorf("Load %s yaml resources failed %w", yamls, err)
		return
	}

	utils.Infof("Starting to prepare %s yaml resources", yamls)
	okYAMLs, err := utils.ExecuteTemplatedYAMLs(yamlData, tmplArgs)
	if err != nil {
		utils.Errorf("Prepare %s yaml resources failed %w", yamls, err)
		return
	}

	if len(okYAMLs) == 0 {
		utils.Info("Found 0 yamls to install")
		return
	}

	if err := deploy(okYAMLs, monitorOnly, clientset, kubeConfig); err != nil {
		utils.Errorf("Deploy kubeStar resource failed with error %w", yamls, err)
		os.Exit(1)
	}
	utils.Infof("Deploy success on current cluster %s\n", currentCluster)
}

type deployOptions struct {
	MonitorOnly            bool
	useNodeStorage         bool
	ChooseStorageClassName string

	ChooseNodeName string
}

func deploy(yamls []*utils.YAMLFile, monitorOnly bool, clientset *kubernetes.Clientset, kubeConfig *rest.Config) error {
	var targets []*utils.YAMLFile
	for _, item := range yamls {
		if monitorOnly {
			if item.Dir == "kubestar-monitor" {
				targets = append(targets, item)
			}
		} else {
			if item.Dir == "kubestar" {
				targets = append(targets, item)
			}
		}
	}

	for _, item := range targets {
		utils.Infof("Deploying %s\n", item.Name)
		if err := retryDeploy(clientset, kubeConfig, item.YAML); err != nil {
			utils.Errorf("Deploy %s failed with error %w\n", err)
			return err
		}
	}

	return nil
}

func retryDeploy(clientset *kubernetes.Clientset, config *rest.Config, yamlContents string) error {
	fmt.Println(yamlContents)
	tries := 12
	var err error
	for tries > 0 {
		err = k8s.ApplyYAML(clientset, config, "", strings.NewReader(yamlContents), false)
		if err == nil {
			return nil
		}

		if err != nil && k8serrors.IsAlreadyExists(err) {
			return nil
		}
		time.Sleep(5 * time.Second)
		tries--
	}
	if tries == 0 {
		return err
	}
	return nil
}

func getClusterShortName(fullName string) string {
	i := strings.LastIndex(fullName, "/")
	if i == len(fullName)-1 {
		return fullName
	}

	return fullName[i+1:]

}
