package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/xuyun-io/kubestar-cli/pkg/cmd"
	"github.com/xuyun-io/kubestar-cli/pkg/utils"
	"os"
)

func main() {
	log.SetOutput(os.Stderr)
	utils.Info("KubeStar CLI")
	cmd.Execute()
}
