package adapter_test

import (
	"github.com/pivotal-cf-experimental/kafka-example-service-adapter/adapter"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Adapter/GeneratePlanSchemas", func() {
	It("generates schemas", func() {
		plan := serviceadapter.Plan{
			Properties: serviceadapter.Properties{
				"name": "plan-with-schema",
			},
		}
		schemas := serviceadapter.JSONSchemas{
			Parameters: map[string]interface{}{
				"$schema":              "http://json-schema.org/draft-04/schema#",
				"type":                 "object",
				"additionalProperties": true,
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
		expectedSchema := serviceadapter.PlanSchema{
			ServiceInstance: serviceadapter.ServiceInstanceSchema{
				Create: schemas,
				Update: schemas,
			},
			ServiceBinding: serviceadapter.ServiceBindingSchema{
				Create: bindSchema,
			},
		}

		generator := &adapter.SchemaGenerator{}
		Expect(generator.GeneratePlanSchema(plan)).To(Equal(expectedSchema))
	})

	It("fails with an error if the plan is unknown", func() {
		plan := serviceadapter.Plan{
			Properties: serviceadapter.Properties{
				"service_adapter_fails": true,
			},
		}
		generator := &adapter.SchemaGenerator{}
		_, err := generator.GeneratePlanSchema(plan)
		Expect(err).To(HaveOccurred())
	})

	Describe("when schema_to_return is set", func() {
		It("returns the supplied schema", func() {
			plan := serviceadapter.Plan{
				Properties: serviceadapter.Properties{
					"schema_to_return": `{"foo": "bar"}`,
				},
			}
			generator := &adapter.SchemaGenerator{}
			schemas, err := generator.GeneratePlanSchema(plan)
			Expect(err).NotTo(HaveOccurred())
			expectedSchema := map[string]interface{}{
				"foo": "bar",
			}
			Expect(schemas.ServiceInstance.Create.Parameters).To(Equal(expectedSchema))
			Expect(schemas.ServiceInstance.Update.Parameters).To(Equal(expectedSchema))
			Expect(schemas.ServiceBinding.Create.Parameters).To(Equal(expectedSchema))
		})

		It("fails if it's not a valid json", func() {
			plan := serviceadapter.Plan{
				Properties: serviceadapter.Properties{
					"schema_to_return": `nop`,
				},
			}
			generator := &adapter.SchemaGenerator{}
			_, err := generator.GeneratePlanSchema(plan)
			Expect(err).To(MatchError("Invalid 'schema_to_return' JSON"))
		})

		It("fails if it's not a string", func() {
			plan := serviceadapter.Plan{
				Properties: serviceadapter.Properties{
					"schema_to_return": false,
				},
			}
			generator := &adapter.SchemaGenerator{}
			_, err := generator.GeneratePlanSchema(plan)
			Expect(err).To(MatchError("'schema_to_return' must be a JSON string"))
		})
	})
})
