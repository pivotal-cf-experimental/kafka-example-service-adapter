package adapter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

const OnlyStemcellAlias = "only-stemcell"

func defaultDeploymentInstanceGroupsToJobs() map[string][]string {
	return map[string][]string{
		"kafka_server":     []string{"kafka_server"},
		"zookeeper_server": []string{"zookeeper_server"},
		"smoke_tests":      []string{"smoke_tests"},
	}
}

func (a *ManifestGenerator) GenerateManifest(params serviceadapter.GenerateManifestParams) (serviceadapter.GenerateManifestOutput, error) {

	if params.PreviousPlan != nil {
		prev := instanceCounts(*params.PreviousPlan)
		current := instanceCounts(params.Plan)
		if (prev["kafka_server"] > current["kafka_server"]) || (prev["zookeeper_server"] > current["zookeeper_server"]) {
			a.StderrLogger.Println("cannot migrate to a smaller plan")
			return serviceadapter.GenerateManifestOutput{}, errors.New("")
		}
	}

	var releases []bosh.Release

	loggingRaw, ok := params.Plan.Properties["logging"]
	includeMetron := false
	if ok {
		includeMetron = true
	}

	for _, serviceRelease := range params.ServiceDeployment.Releases {
		releases = append(releases, bosh.Release{
			Name:    serviceRelease.Name,
			Version: serviceRelease.Version,
		})
	}

	deploymentInstanceGroupsToJobs := defaultDeploymentInstanceGroupsToJobs()
	if includeMetron {
		for instanceGroup, jobs := range deploymentInstanceGroupsToJobs {
			deploymentInstanceGroupsToJobs[instanceGroup] = append(jobs, "metron_agent")
		}
	}

	err := checkInstanceGroupsPresent([]string{"kafka_server", "zookeeper_server"}, params.Plan.InstanceGroups)
	if err != nil {
		a.StderrLogger.Println(err.Error())
		return serviceadapter.GenerateManifestOutput{}, errors.New("Contact your operator, service configuration issue occurred")
	}

	instanceGroups, err := InstanceGroupMapper(params.Plan.InstanceGroups, params.ServiceDeployment.Releases, OnlyStemcellAlias, deploymentInstanceGroupsToJobs)
	if err != nil {
		a.StderrLogger.Println(err.Error())
		return serviceadapter.GenerateManifestOutput{}, errors.New("")
	}

	kafkaServerRelease, err := serviceadapter.FindReleaseForJob("kafka_server", params.ServiceDeployment.Releases)

	if err != nil {
		a.StderrLogger.Println("Cannot determine kafka_server release", err.Error())
		return serviceadapter.GenerateManifestOutput{}, errors.New("")
	}

	minVersion := semver.New(MinServiceReleaseVersion)
	actualVersion, err := semver.NewVersion(kafkaServerRelease.Version)
	if err != nil {
		a.StderrLogger.Printf("Skipping min service release version check. Release version '%s' is not valid semver\n", kafkaServerRelease.Version)
	} else {
		if actualVersion.LessThan(*minVersion) {
			err = fmt.Errorf("minimum release version not met: >= kafka-service-release %s required", MinServiceReleaseVersion)
			a.StderrLogger.Printf(err.Error())
			return serviceadapter.GenerateManifestOutput{}, err
		}
	}

	kafkaBrokerInstanceGroup := &instanceGroups[0]

	if len(kafkaBrokerInstanceGroup.Networks) != 1 {
		a.StderrLogger.Println(fmt.Sprintf("expected 1 network for %s, got %d", kafkaBrokerInstanceGroup.Name, len(kafkaBrokerInstanceGroup.Networks)))
		return serviceadapter.GenerateManifestOutput{}, errors.New("")
	}

	autoCreateTopics := true
	arbitraryParameters := params.RequestParams.ArbitraryParams()

	if arbitraryVal, ok := arbitraryParameters["auto_create_topics"]; ok {
		autoCreateTopics = arbitraryVal.(bool)
	} else if previousVal, previousOk := getPreviousManifestProperty("auto_create_topics", params.PreviousManifest); previousOk {
		autoCreateTopics = previousVal.(bool)
	} else if planVal, ok := params.Plan.Properties["auto_create_topics"]; ok {
		autoCreateTopics = planVal.(bool)
	}

	defaultReplicationFactor := 3
	if arbitraryVal, ok := arbitraryParameters["default_replication_factor"]; ok {
		defaultReplicationFactor = int(arbitraryVal.(float64))
	} else if val, ok := params.Plan.Properties["default_replication_factor"]; ok {
		defaultReplicationFactor = int(val.(float64))
	}

	serviceAdapterFails := false
	if val, ok := params.Plan.Properties["service_adapter_fails"]; ok {
		serviceAdapterFails = val.(bool)
	}
	if kafkaBrokerJob, ok := getJobFromInstanceGroup("kafka_server", kafkaBrokerInstanceGroup); ok {
		kafkaBrokerJob.Properties = map[string]interface{}{
			"default_replication_factor": defaultReplicationFactor,
			"auto_create_topics":         autoCreateTopics,
			"network":                    kafkaBrokerInstanceGroup.Networks[0].Name,
			"service_adapter_fails":      serviceAdapterFails,
		}
	}

	manifestProperties := map[string]interface{}{}

	if includeMetron {
		logging := loggingRaw.(map[string]interface{})
		manifestProperties["syslog_daemon_config"] = map[interface{}]interface{}{
			"address": logging["syslog_address"],
			"port":    logging["syslog_port"],
		}
		manifestProperties["metron_agent"] = map[interface{}]interface{}{
			"zone":       "",
			"deployment": params.ServiceDeployment.DeploymentName,
		}
		manifestProperties["loggregator"] = map[interface{}]interface{}{
			"tls": map[interface{}]interface{}{
				"metron": map[interface{}]interface{}{
					"cert": logging["loggregator_tls_metron_cert"],
					"key":  logging["loggregator_tls_metron_key"],
				},
				"ca_cert": logging["loggregator_tls_ca_cert"],
			},
			"loggregator_endpoint": map[interface{}]interface{}{
				"shared_secret": logging["loggregator_shared_secret"],
			},
			"etcd": map[interface{}]interface{}{
				"ca_cert":  logging["loggregator_etcd_ca_cert"],
				"machines": logging["loggregator_etcd_addresses"].([]interface{}),
			},
		}
		manifestProperties["metron_endpoint"] = map[interface{}]interface{}{
			"shared_secret": logging["loggregator_shared_secret"],
		}
	}

	updateBlock := &bosh.Update{
		Canaries:        1,
		MaxInFlight:     10,
		CanaryWatchTime: "30000-240000",
		UpdateWatchTime: "30000-240000",
		Serial:          boolPointer(false),
	}

	if params.Plan.Update != nil {
		updateBlock = &bosh.Update{
			Canaries:        params.Plan.Update.Canaries,
			MaxInFlight:     params.Plan.Update.MaxInFlight,
			CanaryWatchTime: params.Plan.Update.CanaryWatchTime,
			UpdateWatchTime: params.Plan.Update.UpdateWatchTime,
			Serial:          params.Plan.Update.Serial,
		}
	}

	manifest := bosh.BoshManifest{
		Name:     params.ServiceDeployment.DeploymentName,
		Releases: releases,
		Stemcells: []bosh.Stemcell{{
			Alias:   OnlyStemcellAlias,
			OS:      params.ServiceDeployment.Stemcell.OS,
			Version: params.ServiceDeployment.Stemcell.Version,
		}},
		InstanceGroups: instanceGroups,
		Properties:     manifestProperties,
		Update:         updateBlock,
	}

	return serviceadapter.GenerateManifestOutput{
		Manifest:          manifest,
		ODBManagedSecrets: serviceadapter.ODBManagedSecrets{},
	}, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getPreviousManifestProperty(name string, manifest *bosh.BoshManifest) (interface{}, bool) {
	if manifest != nil {
		if val, ok := manifest.Properties["auto_create_topics"]; ok {
			return val, true
		}
	}
	return nil, false
}

func getJobFromInstanceGroup(name string, instanceGroup *bosh.InstanceGroup) (*bosh.Job, bool) {
	for index, job := range instanceGroup.Jobs {
		if job.Name == name {
			return &instanceGroup.Jobs[index], true
		}
	}
	return &bosh.Job{}, false
}

func instanceCounts(plan serviceadapter.Plan) map[string]int {
	val := map[string]int{}
	for _, instanceGroup := range plan.InstanceGroups {
		val[instanceGroup.Name] = instanceGroup.Instances
	}
	return val
}

func boolPointer(b bool) *bool {
	return &b
}

func checkInstanceGroupsPresent(names []string, instanceGroups []serviceadapter.InstanceGroup) error {
	var missingNames []string

	for _, name := range names {
		if !containsInstanceGroup(name, instanceGroups) {
			missingNames = append(missingNames, name)
		}
	}

	if len(missingNames) > 0 {
		return fmt.Errorf("Invalid instance group configuration: expected to find: '%s' in list: '%s'",
			strings.Join(missingNames, ", "),
			strings.Join(getInstanceGroupNames(instanceGroups), ", "))
	}
	return nil
}

func getInstanceGroupNames(instanceGroups []serviceadapter.InstanceGroup) []string {
	var instanceGroupNames []string
	for _, instanceGroup := range instanceGroups {
		instanceGroupNames = append(instanceGroupNames, instanceGroup.Name)
	}
	return instanceGroupNames
}

func containsInstanceGroup(name string, instanceGroups []serviceadapter.InstanceGroup) bool {
	for _, instanceGroup := range instanceGroups {
		if instanceGroup.Name == name {
			return true
		}
	}

	return false
}
