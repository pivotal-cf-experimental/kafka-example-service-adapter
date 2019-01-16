package adapter_test

import (
	"github.com/pivotal-cf-experimental/kafka-example-service-adapter/adapter"
	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Adapter/GenerateDashboardUrl", func() {
	It("generates a arbitrary dashboard url", func() {
		generator := &adapter.DashboardUrlGenerator{}
		params := serviceadapter.DashboardUrlParams{
			InstanceID: "instanceID",
			Plan:       serviceadapter.Plan{},
			Manifest:   bosh.BoshManifest{},
		}

		Expect(generator.DashboardUrl(params)).To(Equal(serviceadapter.DashboardUrl{DashboardUrl: "http://example_dashboard.com/instanceID"}))
	})
})
