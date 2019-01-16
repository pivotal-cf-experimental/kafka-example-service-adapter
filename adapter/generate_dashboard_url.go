package adapter

import (
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

type DashboardUrlGenerator struct {
}

func (a *DashboardUrlGenerator) DashboardUrl(params serviceadapter.DashboardUrlParams) (serviceadapter.DashboardUrl, error) {
	return serviceadapter.DashboardUrl{DashboardUrl: "http://example_dashboard.com/" + params.InstanceID}, nil
}
