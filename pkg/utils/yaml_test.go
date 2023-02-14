package utils

import (
	"testing"
)

func TestLoadTemplateYAMLs(t *testing.T) {
	testFile := "/home/michael/go/src/github.com/xuyun-io/kubestar-cli/yamls.tar"
	yamls, err := LoadTemplateYAMLs(testFile)
	if err != nil {
		t.Fatal(err)
	}

	tmplArgs := &YAMLTmplArguments{
		Values: &map[string]interface{}{
			"deployKubeStar": true,
		},
		Release: &map[string]interface{}{
			"Namespace": "kubestar2",
		},
	}

	_, err = ExecuteTemplatedYAMLs(yamls, tmplArgs)
	if err != nil {
		t.Fatal(err)
	}
}
