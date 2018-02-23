package integration_tests

import (
	"bytes"
	"io"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("generate-plan-schemas subcommand", func() {
	var (
		stdout   bytes.Buffer
		exitCode int
		plan     = `{
			"instance_groups": [
				{
					"name": "kafka_server",
					"vm_type": "small",
					"persistent_disk_type": "ten",
					"networks": [
						"example-network"
					],
					"azs": [
						"example-az"
					],
					"instances": 1
				},
				{
					"name": "zookeeper_server",
					"vm_type": "medium",
					"persistent_disk_type": "twenty",
					"networks": [
						"example-network"
					],
					"azs": [
						"example-az"
					],
					"instances": 1
				}
			],
			"properties": {
				"auto_create_topics": false
			}
		}`
		expectedSchema = `{
        "service_instance": {
          "create": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "properties": {
                "auto_create_topics": {
                  "description": "Auto create topics",
                  "required": false,
                  "type": "bool"
                },
                "default_replication_factor": {
                  "description": "Replication factor",
                  "required": false,
                  "type": "integer"
                }
              },
              "type": "object"
            }
          },
          "update": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "properties": {
                "auto_create_topics": {
                  "description": "Auto create topics",
                  "required": false,
                  "type": "bool"
                },
                "default_replication_factor": {
                  "description": "Replication factor",
                  "required": false,
                  "type": "integer"
                }
              },
              "type": "object"
            }
          }
        },
        "service_binding": {
          "create": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "properties": {
                "topic": {
                  "description": "The name of the topic",
                  "required": false,
                  "type": "string"
                }
              },
              "type": "object"
            }
          }
        }
      }`
	)

	BeforeEach(func() {
		stdout = bytes.Buffer{}
		cmd := exec.Command(serviceAdapterBinPath, "generate-plan-schemas", "-plan-json", plan)
		runningBin, err := gexec.Start(cmd, io.MultiWriter(GinkgoWriter, &stdout), GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(runningBin).Should(gexec.Exit())
		exitCode = runningBin.ExitCode()
	})

	It("should succeed", func() {
		Expect(exitCode).To(BeZero())
	})

	It("generates schemas", func() {
		Expect(stdout.String()).To(MatchJSON(expectedSchema))
	})
})
