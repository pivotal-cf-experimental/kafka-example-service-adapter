package adapter

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

func (b *Binder) CreateBinding(params serviceadapter.CreateBindingParams) (serviceadapter.Binding, error) {

	arbitraryParams := params.RequestParams.ArbitraryParams()

	bindResource := params.RequestParams.BindResource()
	b.StderrLogger.Printf("Bind Resource with app GUID: %s, credential client ID: %s, route: %s\n", bindResource.AppGuid, bindResource.CredentialClientID, bindResource.Route)

	var invalidParams []string
	for paramKey, _ := range arbitraryParams {
		if paramKey != "topic" {
			invalidParams = append(invalidParams, paramKey)
		}
	}

	if len(invalidParams) > 0 {
		sort.Strings(invalidParams)
		errorMessage := fmt.Sprintf("unsupported parameter(s) for this service: %s", strings.Join(invalidParams, ", "))
		b.StderrLogger.Println(errorMessage)
		return serviceadapter.Binding{}, errors.New(errorMessage)
	}

	kafkaHosts := params.DeploymentTopology["kafka_server"]
	if len(kafkaHosts) == 0 {
		b.StderrLogger.Println("no VMs for instance group kafka_server")
		return serviceadapter.Binding{}, errors.New("")
	}

	var kafkaAddresses []interface{}
	for _, kafkaHost := range kafkaHosts {
		kafkaAddresses = append(kafkaAddresses, fmt.Sprintf("%s:9092", kafkaHost))
	}

	zookeeperServers := params.DeploymentTopology["zookeeper_server"]
	if len(zookeeperServers) == 0 {
		b.StderrLogger.Println("no VMs for job zookeeper_server")
		return serviceadapter.Binding{}, errors.New("")
	}

	if _, errorStream, err := b.Run(b.TopicCreatorCommand, strings.Join(zookeeperServers, ","), params.BindingID); err != nil {
		if strings.Contains(string(errorStream), "kafka.common.TopicExistsException") {
			b.StderrLogger.Println(fmt.Sprintf("topic '%s' already exists", params.BindingID))
			return serviceadapter.Binding{}, serviceadapter.NewBindingAlreadyExistsError(nil)
		}
		b.StderrLogger.Println("Error creating topic: " + err.Error())
		return serviceadapter.Binding{}, errors.New("")
	}

	if arbitraryParams["topic"] != nil {
		if _, _, err := b.Run(b.TopicCreatorCommand, strings.Join(zookeeperServers, ","), arbitraryParams["topic"].(string)); err != nil {
			b.StderrLogger.Println("Error creating topic: " + err.Error())
			return serviceadapter.Binding{}, errors.New("")
		}
	}

	return serviceadapter.Binding{
		Credentials: map[string]interface{}{
			"bootstrap_servers": kafkaAddresses,
		},
	}, nil
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fake_command_runner/fake_command_runner.go . CommandRunner
type CommandRunner interface {
	Run(name string, arg ...string) ([]byte, []byte, error)
}

type ExternalCommandRunner struct{}

func (c ExternalCommandRunner) Run(name string, arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(name, arg...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.Output()
	return stdout, stderr.Bytes(), err
}
