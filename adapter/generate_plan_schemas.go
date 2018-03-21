package adapter

import (
	"encoding/json"
	"errors"

	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

type SchemaGenerator struct{}

func (s SchemaGenerator) GeneratePlanSchema(plan serviceadapter.Plan) (serviceadapter.PlanSchema, error) {

	// For system testing only...
	if plan.Properties["service_adapter_fails"] == true {
		return serviceadapter.PlanSchema{}, errors.New("Cannot generate the schema")
	}
	if theSchema, ok := plan.Properties["schema_to_return"]; ok {
		var planSchema serviceadapter.PlanSchema
		var schema map[string]interface{}
		schemaStr, ok := theSchema.(string)
		if !ok {
			return planSchema, errors.New("'schema_to_return' must be a JSON string")
		}
		err := json.Unmarshal([]byte(schemaStr), &schema)
		if err != nil {
			return planSchema, errors.New("Invalid 'schema_to_return' JSON")
		}
		planSchema = serviceadapter.PlanSchema{
			ServiceInstance: serviceadapter.ServiceInstanceSchema{
				Create: serviceadapter.JSONSchemas{Parameters: schema},
				Update: serviceadapter.JSONSchemas{Parameters: schema},
			},
			ServiceBinding: serviceadapter.ServiceBindingSchema{
				Create: serviceadapter.JSONSchemas{Parameters: schema},
			},
		}

		return planSchema, nil
	}
	// ... end

	schemas := serviceadapter.JSONSchemas{
		Parameters: map[string]interface{}{
			"$schema":              "http://json-schema.org/draft-04/schema#",
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"auto_create_topics": map[string]interface{}{
					"description": "Auto create topics",
					"type":        "boolean",
				},
				"default_replication_factor": map[string]interface{}{
					"description": "Replication factor",
					"type":        "integer",
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
