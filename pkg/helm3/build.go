package helm3

import (
	"fmt"
	"strings"

	"get.porter.sh/porter/pkg/exec/builder"
	yaml "gopkg.in/yaml.v2"
)

// These values may be referenced elsewhere (init.go), hence consts
var helmClientVersion string

// BuildInput represents stdin passed to the mixin for the build command.
type BuildInput struct {
	Config MixinConfig
}

// MixinConfig represents configuration that can be set on the helm mixin in porter.yaml
// mixins:
// - helm3:
//	  repositories:
//	    stable:
//		  url: "https://kubernetes-charts.storage.googleapis.com"
//		  cafile: "path/to/cafile"
//		  certfile: "path/to/certfile"
//		  keyfile: "path/to/keyfile"
//		  username: "username"
//		  password: "password"
type MixinConfig struct {
	ClientVersion string `yaml:"clientVersion,omitempty"`
	Repositories  map[string]Repository
}

type Repository struct {
	URL      string `yaml:"url,omitempty"`
	Cafile   string `yaml:"cafile,omitempty"`
	Certfile string `yaml:"certfile,omitempty"`
	Keyfile  string `yaml:"keyfile,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// Build will generate the necessary Dockerfile lines
// for an invocation image using this mixin
func (m *Mixin) Build() error {

	// Create new Builder.
	var input BuildInput
	err := builder.LoadAction(m.Context, "", func(contents []byte) (interface{}, error) {
		err := yaml.Unmarshal(contents, &input)
		return &input, err
	})
	if err != nil {
		return err
	}

	if input.Config.ClientVersion != "" {
		m.HelmClientVersion = input.Config.ClientVersion
	}

	// Install helm3
	fmt.Fprintf(m.Out, "RUN apt-get update && apt-get install -y curl")
	fmt.Fprintf(m.Out, "\nRUN curl https://get.helm.sh/helm-%s-linux-amd64.tar.gz --output helm3.tar.gz", m.HelmClientVersion)
	fmt.Fprintf(m.Out, "\nRUN tar -xvf helm3.tar.gz")
	fmt.Fprintf(m.Out, "\nRUN mv linux-amd64/helm /usr/local/bin/helm3")

	// Go through repositories
	for name, repo := range input.Config.Repositories {

		commandValue, err := GetAddRepositoryCommand(name, repo.URL, repo.Cafile, repo.Certfile, repo.Keyfile, repo.Username, repo.Password)
		if err != nil && m.Debug {
			fmt.Fprintf(m.Err, "DEBUG: addition of repository failed: %s\n", err.Error())
		} else {
			fmt.Fprintf(m.Out, strings.Join(commandValue, " "))
		}
	}
	return nil
}

func GetAddRepositoryCommand(name, url, cafile, certfile, keyfile, username, password string) (commandValue []string, err error) {

	var commandBuilder []string

	if url == "" {
		return commandBuilder, fmt.Errorf("repository url must be supplied")
	}

	commandBuilder = append(commandBuilder, "\nRUN", "helm3", "repo", "add", name, url)

	if certfile != "" && keyfile != "" {
		commandBuilder = append(commandBuilder, "--cert-file", certfile, "--key-file", keyfile)
	}
	if cafile != "" {
		commandBuilder = append(commandBuilder, "--ca-file", cafile)
	}
	if username != "" && password != "" {
		commandBuilder = append(commandBuilder, "--username", username, "--password", password)
	}

	return commandBuilder, nil
}
