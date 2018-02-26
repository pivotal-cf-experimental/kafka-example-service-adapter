package adapter

import (
	"errors"

	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

type SchemaGenerator struct{}

func (s SchemaGenerator) GeneratePlanSchema(plan serviceadapter.Plan) (serviceadapter.PlanSchema, error) {
	if plan.Properties["service_adapter_fails"] == true {
		return serviceadapter.PlanSchema{}, errors.New("Cannot generate the schema")
	}
	schemas := serviceadapter.JSONSchemas{
		Parameters: map[string]interface{}{
			"$schema":              "http://json-schema.org/draft-04/schema#",
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"auto_create_topics": map[string]interface{}{
					"description": "Auto create topics",
					"type":        "bool",
					"required":    false,
				},
				"default_replication_factor": map[string]interface{}{
					"description": "Replication factor",
					"type":        "integer",
					"required":    false,
				},
			},
		},
	}
	bindSchema := serviceadapter.JSONSchemas{
		Parameters: map[string]interface{}{
			"$schema":              "http://json-schema.org/draft-04/schema#",
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"topic": map[string]interface{}{
					"description": "The name of the topic",
					"type":        "string",
					"required":    false,
				},
			},
		},
	}
	return serviceadapter.PlanSchema{
		ServiceInstance: serviceadapter.ServiceInstanceSchema{
			Create: schemas,
			Update: schemas,
		},
		ServiceBinding: serviceadapter.ServiceBindingSchema{
			Create: bindSchema,
		},
	}, nil
}
