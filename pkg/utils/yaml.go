package utils

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	goyaml "gopkg.in/yaml.v2"
)

// YAMLFile is a YAML associated with a name.
type YAMLFile struct {
	Dir  string
	Name string
	YAML string
}

// YAMLTmplArguments is a wrapper around YAMLTmplValues.
type YAMLTmplArguments struct {
	Values *map[string]interface{}
	// Release values represent special fields that are filled out by Helm.
	Release *map[string]interface{}
}

func LoadTemplateYAMLs(tar string) ([]*YAMLFile, error) {
	f, err := os.Open(tar)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	yamlMap, err := ReadTarFileFromReader(f)

	yamlNames := make([]string, len(yamlMap))
	i := 0
	for k := range yamlMap {
		yamlNames[i] = k
		i++
	}
	sort.Strings(yamlNames)

	// Write to YAMLFile slice.
	var yamlFiles []*YAMLFile
	re := regexp.MustCompile(`((?:[0-9]+_)|(?:kubestar-monitor)|(?:kubestar))(.*)(?:\.yaml)`)
	for _, fName := range yamlNames {
		// The filename looks like "./pixie_yamls/00_namespace.yaml" or "./pixie_yamls/crds/vizier_crd.yaml", we want to extract the "namespace".
		ms := re.FindStringSubmatch(fName)
		if ms == nil || len(ms) != 3 {
			continue
		}
		yamlFiles = append(yamlFiles, &YAMLFile{
			Dir:  ms[1],
			Name: ms[2],
			YAML: yamlMap[fName],
		})
	}

	return yamlFiles, nil
}

// ExecuteTemplatedYAMLs takes a template YAML and applies the given template values to it.
func ExecuteTemplatedYAMLs(yamls []*YAMLFile, tmplValues *YAMLTmplArguments) ([]*YAMLFile, error) {
	// Execute the template on each of the YAMLs.
	executedYAMLs := make([]*YAMLFile, len(yamls))
	for i, y := range yamls {
		yamlFile := &YAMLFile{
			Dir:  y.Dir,
			Name: y.Name,
		}

		if tmplValues == nil {
			yamlFile.YAML = y.YAML
		} else {
			executedYAML, err := executeTemplate(tmplValues, y.YAML)
			if err != nil {
				return nil, err
			}
			yamlFile.YAML = executedYAML
		}
		executedYAMLs[i] = yamlFile
	}

	return executedYAMLs, nil
}

func executeTemplate(tmplValues *YAMLTmplArguments, tmplStr string) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["required"] = required
	funcMap["toYaml"] = toYAML

	tmpl, err := template.New("yaml").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, tmplValues)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func required(str string, value string) (string, error) {
	if value != "" {
		return value, nil
	}
	return "", errors.New("Value is required")
}

func toYAML(v interface{}) string {
	data, err := goyaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}
