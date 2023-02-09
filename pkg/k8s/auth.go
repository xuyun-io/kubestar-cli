package k8s

import (
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"os"
	"path/filepath"
	"strings"
)

var kubeconfig *string

func init() {
	defaultKubeConfig := ""
	optionalStr := "(optional) "
	if k := os.Getenv("KUBECONFIG"); k != "" {
		for _, config := range strings.Split(k, ":") {
			if fileExists(config) {
				defaultKubeConfig = config
				break
			}
		}
		if defaultKubeConfig == "" {
			// Don't use log.Fatal, because it will send an error to Sentry when invoked from the CLI.
			fmt.Println("Failed to find valid config in KUBECONFIG env. Is it formatted correctly?")
			os.Exit(1)
		}
	} else if home := homeDir(); home != "" {
		defaultKubeConfig = filepath.Join(home, ".kube", "config")
	} else {
		optionalStr = ""
	}

	kubeconfig = pflag.String("kubeconfig", defaultKubeConfig, fmt.Sprintf("%sabsolute path to the kubeconfig file", optionalStr))
}

// GetConfig gets the kubernetes rest config.
func GetConfig() *rest.Config {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		// Don't use log.Fatal, because it will send an error to Sentry when invoked from the CLI.
		fmt.Printf("Could not build kubeconfig: %s\n", err.Error())
		os.Exit(1)
	}

	return config
}

// GetClientAPIConfig gets the config used for reading the current kube contexts.
func GetClientAPIConfig() *clientcmdapi.Config {
	return clientcmd.GetConfigFromFileOrDie(*kubeconfig)
}

// GetClientset gets the clientset for the current kubernetes cluster.
func GetClientset(config *rest.Config) *kubernetes.Clientset {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		// Don't use log.Fatal, because it will send an error to Sentry when invoked from the CLI.
		fmt.Printf("Could not create k8s clientset: %s\n", err.Error())
		os.Exit(1)
	}
	return clientset
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetDiscoveryClient gets the discovery client for the current kubernetes cluster.
func GetDiscoveryClient(config *rest.Config) *discovery.DiscoveryClient {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		// Don't use log.Fatal, because it will send an error to Sentry when invoked from the CLI.
		fmt.Printf("Could not create k8s discovery client: %s\n", err.Error())
		os.Exit(1)
	}

	return discoveryClient
}
