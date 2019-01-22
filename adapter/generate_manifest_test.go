package adapter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/kafka-example-service-adapter/adapter"
	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

var _ = Describe("generating manifests", func() {
	boolPointer := func(b bool) *bool {
		return &b
	}

	var (
		serviceRelease             serviceadapter.ServiceRelease
		serviceDeployment          serviceadapter.ServiceDeployment
		generatedInstanceGroups    []bosh.InstanceGroup
		planInstanceGroups         []serviceadapter.InstanceGroup
		previousPlanInstanceGroups []serviceadapter.InstanceGroup
		manifest                   bosh.BoshManifest
		generateErr                error
		plan                       serviceadapter.Plan
		actualInstanceGroups       []serviceadapter.InstanceGroup
		actualServiceReleases      serviceadapter.ServiceReleases
		actualStemcell             string
		output                     serviceadapter.GenerateManifestOutput
	)

	BeforeEach(func() {
		serviceRelease = serviceadapter.ServiceRelease{
			Name:    "wicked-release",
			Version: "0.16.0",
			Jobs:    []string{"kafka_server"},
		}
		serviceDeployment = serviceadapter.ServiceDeployment{
			DeploymentName: "a-great-deployment",
			Releases:       serviceadapter.ServiceReleases{serviceRelease},
			Stemcell:       serviceadapter.Stemcell{OS: "TempleOS", Version: "4.05"},
		}
		generatedInstanceGroups = []bosh.InstanceGroup{{
			Name:     "kafka_server",
			Networks: []bosh.Network{{Name: "an-etwork"}},
			Jobs: []bosh.Job{{
				Name: "kafka_server",
			}},
		}}
		planInstanceGroups = []serviceadapter.InstanceGroup{
			{
				Name: "kafka_server",
			},
			{
				Name: "zookeeper_server",
			},
		}
		previousPlanInstanceGroups = []serviceadapter.InstanceGroup{}
		plan = serviceadapter.Plan{InstanceGroups: planInstanceGroups}
		adapter.InstanceGroupMapper = func(instanceGroups []serviceadapter.InstanceGroup,
			serviceReleases serviceadapter.ServiceReleases,
			stemcell string,
			deploymentInstanceGroupsToJobs map[string][]string) ([]bosh.InstanceGroup, error) {
			actualInstanceGroups = instanceGroups
			actualServiceReleases = serviceReleases
			actualStemcell = stemcell
			return generatedInstanceGroups, nil
		}
		adapter.MinServiceReleaseVersion = "0.16.0"
	})

	Context("when the instance group mapper maps instance groups successfully", func() {
		var (
			requestParams map[string]interface{}
		)

		BeforeEach(func() {
			requestParams = map[string]interface{}{}
		})

		JustBeforeEach(func() {
			output, generateErr = manifestGenerator.GenerateManifest(serviceadapter.GenerateManifestParams{
				ServiceDeployment: serviceDeployment,
				Plan:              plan,
				RequestParams:     requestParams,
				PreviousPlan:      nil,
				PreviousManifest:  nil,
				PreviousSecrets:   nil,
				PreviousConfigs:   nil,
			})
			manifest = output.Manifest
		})

		It("returns no error", func() {
			Expect(generateErr).NotTo(HaveOccurred())
		})

		It("returns the basic deployment information", func() {
			Expect(manifest.Name).To(Equal(serviceDeployment.DeploymentName))
			Expect(manifest.Releases).To(ConsistOf(bosh.Release{Name: serviceRelease.Name, Version: serviceRelease.Version}))
			stemcellAlias := manifest.Stemcells[0].Alias
			Expect(manifest.Stemcells).To(ConsistOf(bosh.Stemcell{Alias: stemcellAlias, OS: serviceDeployment.Stemcell.OS, Version: serviceDeployment.Stemcell.Version}))
		})

		It("returns the instance groups produced by the mapper", func() {
			Expect(manifest.InstanceGroups).To(Equal(generatedInstanceGroups))
		})

		It("adds network name to properties", func() {
			Expect(manifest.InstanceGroups[0].Jobs[0].Properties).To(HaveKeyWithValue("network", "an-etwork"))
		})

		It("adds default values for auto_create_topics and default_replication_factor", func() {
			Expect(manifest.InstanceGroups[0].Jobs[0].Properties).To(HaveKeyWithValue("auto_create_topics", true))
			Expect(manifest.InstanceGroups[0].Jobs[0].Properties).To(HaveKeyWithValue("default_replication_factor", 3))
		})

		Context("the plan property auto_create_topics is specified", func() {
			BeforeEach(func() {
				plan.Properties = map[string]interface{}{"auto_create_topics": false}
			})
			It("overrides the value in the manifest", func() {
				Expect(manifest.InstanceGroups[0].Jobs[0].Properties).To(HaveKeyWithValue("auto_create_topics", false))
			})
		})

		Context("the plan property service_adapter_fails is specified", func() {
			BeforeEach(func() {
				plan.Properties = map[string]interface{}{"service_adapter_fails": true}
			})
			It("overrides the value in the manifest", func() {
				Expect(manifest.InstanceGroups[0].Jobs[0].Properties).To(HaveKeyWithValue("service_adapter_fails", true))
			})
		})

		Context("the plan property default_replication_factor is specified", func() {
			BeforeEach(func() {
				plan.Properties = map[string]interface{}{"default_replication_factor": 55.0} // JSON has no integers
			})
			It("overrides the value in the manifest", func() {
				Expect(manifest.InstanceGroups[0].Jobs[0].Properties).To(HaveKeyWithValue("default_replication_factor", 55))
			})
		})

		Context("the plan property default_replication_factor is specified and it is overridden by request params", func() {
			BeforeEach(func() {
				plan.Properties = map[string]interface{}{"default_replication_factor": 55.0} // JSON has no integers
				requestParams = map[string]interface{}{
					"parameters": map[string]interface{}{
						"default_replication_factor": 42.0,
					},
				}
			})

			It("overrides the value in the manifest", func() {
				Expect(manifest.InstanceGroups[0].Jobs[0].Properties).To(HaveKeyWithValue("default_replication_factor", 42))
			})
		})

		It("passes expected parameters to instance group mapper", func() {
			Expect(actualInstanceGroups).To(Equal(planInstanceGroups))
			Expect(actualServiceReleases).To(ConsistOf(serviceRelease))
			Expect(actualStemcell).To(Equal(manifest.Stemcells[0].Alias))
		})

		It("adds some reasonable defaults for update", func() {
			Expect(manifest.Update.Canaries).To(Equal(1))
			Expect(manifest.Update.MaxInFlight).To(Equal(10))
			Expect(manifest.Update.CanaryWatchTime).To(Equal("30000-240000"))
			Expect(manifest.Update.UpdateWatchTime).To(Equal("30000-240000"))
			Expect(manifest.Update.Serial).To(Equal(boolPointer(false)))
		})
	})

	Context("minimum service release version validation", func() {
		It("fails when service release < min required", func() {
			serviceRelease.Version = "0.15.0"
			serviceDeployment.Releases = serviceadapter.ServiceReleases{serviceRelease}
			output, generateErr = manifestGenerator.GenerateManifest(serviceadapter.GenerateManifestParams{
				ServiceDeployment: serviceDeployment,
				Plan:              plan,
				RequestParams:     map[string]interface{}{},
			})
			Expect(generateErr).To(HaveOccurred())
		})

		It("succeeds when service release == min required", func() {
			serviceRelease.Version = "0.16.0"
			serviceDeployment.Releases = serviceadapter.ServiceReleases{serviceRelease}
			output, generateErr = manifestGenerator.GenerateManifest(serviceadapter.GenerateManifestParams{
				ServiceDeployment: serviceDeployment,
				Plan:              plan,
				RequestParams:     map[string]interface{}{},
			})
			Expect(generateErr).ToNot(HaveOccurred())
		})

		It("succeeds when service release > min required", func() {
			serviceRelease.Version = "0.100.0"
			serviceDeployment.Releases = serviceadapter.ServiceReleases{serviceRelease}
			output, generateErr = manifestGenerator.GenerateManifest(serviceadapter.GenerateManifestParams{
				ServiceDeployment: serviceDeployment,
				Plan:              plan,
				RequestParams:     map[string]interface{}{},
			})
			Expect(generateErr).ToNot(HaveOccurred())
		})

		It("ignores version validation with invalid semver versions", func() {
			serviceRelease.Version = "0+dev.1"
			serviceDeployment.Releases = serviceadapter.ServiceReleases{serviceRelease}
			stderr.Reset()
			output, generateErr = manifestGenerator.GenerateManifest(serviceadapter.GenerateManifestParams{
				ServiceDeployment: serviceDeployment,
				Plan:              plan,
				RequestParams:     map[string]interface{}{},
			})
			Expect(generateErr).ToNot(HaveOccurred())
			Expect(stderr.String()).To(ContainSubstring("Skipping min service release version check"))
		})

		It("panics when MinServiceReleaseVersion is not a valid semver", func() {
			adapter.MinServiceReleaseVersion = "0+dev.2"
			serviceDeployment.Releases = serviceadapter.ServiceReleases{serviceRelease}
			genMani := func() {
				manifestGenerator.GenerateManifest(serviceadapter.GenerateManifestParams{
					ServiceDeployment: serviceDeployment,
					Plan:              plan,
					RequestParams:     map[string]interface{}{},
				})
			}
			Expect(genMani).To(Panic())
		})
	})

	Context("plan migrations", func() {
		var (
			previousPlan serviceadapter.Plan
		)
		BeforeEach(func() {
			planInstanceGroups = []serviceadapter.InstanceGroup{{Name: "kafka_server", Instances: 2}, {Name: "zookeeper_server", Instances: 2}}
			plan = serviceadapter.Plan{InstanceGroups: planInstanceGroups}
		})
		JustBeforeEach(func() {
			previousPlan = serviceadapter.Plan{InstanceGroups: previousPlanInstanceGroups}
			stderr.Reset()
			output, generateErr = manifestGenerator.GenerateManifest(serviceadapter.GenerateManifestParams{
				ServiceDeployment: serviceDeployment,
				Plan:              plan,
				RequestParams:     map[string]interface{}{"parameters": map[string]interface{}{}},
				PreviousPlan:      &previousPlan,
				PreviousManifest:  nil,
				PreviousSecrets:   nil,
				PreviousConfigs:   nil,
			})
		})

		Context("when the previous plan had more zookeepers", func() {
			BeforeEach(func() {
				previousPlanInstanceGroups = []serviceadapter.InstanceGroup{{Name: "kafka_server", Instances: 2}, {Name: "zookeeper_server", Instances: 3}}
			})
			It("fails", func() {
				Expect(generateErr).To(HaveOccurred())
			})
			It("logs details of the error for the operator", func() {
				Expect(stderr.String()).To(ContainSubstring("cannot migrate to a smaller plan"))
			})
			It("returns an empty error to the cli user", func() {
				Expect(generateErr.Error()).To(Equal(""))
			})
		})
		Context("when the previous plan had more kafkas", func() {
			BeforeEach(func() {
				previousPlanInstanceGroups = []serviceadapter.InstanceGroup{{Name: "kafka_server", Instances: 3}, {Name: "zookeeper_server", Instances: 2}}
			})
			It("fails", func() {
				Expect(generateErr).To(HaveOccurred())
			})
			It("logs details of the error for the operator", func() {
				Expect(stderr.String()).To(ContainSubstring("cannot migrate to a smaller plan"))
			})
			It("returns an empty error to the cli user", func() {
				Expect(generateErr.Error()).To(Equal(""))
			})
		})
		Context("when the previous plan had same number of zookeepers", func() {
			BeforeEach(func() {
				previousPlanInstanceGroups = []serviceadapter.InstanceGroup{{Name: "kafka_server", Instances: 2}, {Name: "zookeeper_server", Instances: 2}}
			})
			It("succeeds", func() {
				Expect(generateErr).NotTo(HaveOccurred())
			})
			It("does not log an error for the operator", func() {
				Expect(stderr.String()).To(Equal(""))
			})
		})
		Context("when the previous plan had same number of kafkas", func() {
			BeforeEach(func() {
				previousPlanInstanceGroups = []serviceadapter.InstanceGroup{{Name: "kafka_server", Instances: 2}, {Name: "zookeeper_server", Instances: 2}}
			})
			It("succeeds", func() {
				Expect(generateErr).NotTo(HaveOccurred())
			})
			It("does not log an error for the operator", func() {
				Expect(stderr.String()).To(Equal(""))
			})
		})
		Context("when the previous plan had fewer zookeepers", func() {
			BeforeEach(func() {
				previousPlanInstanceGroups = []serviceadapter.InstanceGroup{{Name: "kafka_server", Instances: 2}, {Name: "zookeeper_server", Instances: 1}}
			})
			It("succeeds", func() {
				Expect(generateErr).NotTo(HaveOccurred())
			})
			It("does not log an error for the operator", func() {
				Expect(stderr.String()).To(Equal(""))
			})
		})
		Context("when the previous plan had fewer kafkas", func() {
			BeforeEach(func() {
				previousPlanInstanceGroups = []serviceadapter.InstanceGroup{{Name: "kafka_server", Instances: 1}, {Name: "zookeeper_server", Instances: 2}}
			})
			It("succeeds", func() {
				Expect(generateErr).NotTo(HaveOccurred())
			})
			It("does not log an error for the operator", func() {
				Expect(stderr.String()).To(Equal(""))
			})
		})

	})
})
