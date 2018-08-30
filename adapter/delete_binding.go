package adapter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

func (b *Binder) DeleteBinding(bindingId string, boshVMs bosh.BoshVMs, manifest bosh.BoshManifest, requestParameters serviceadapter.RequestParameters,
	secrets serviceadapter.ManifestSecrets) error {

	zookeeperServers := boshVMs["zookeeper_server"]
	if len(zookeeperServers) == 0 {
		b.StderrLogger.Println("no VMs for job zookeeper_server")
		return errors.New("")
	}

	if _, errorStream, err := b.Run(b.TopicDeleterCommand, strings.Join(zookeeperServers, ","), bindingId); err != nil {
		if strings.Contains(string(errorStream), fmt.Sprintf("Topic %s does not exist on ZK path", bindingId)) {
			b.StderrLogger.Println(fmt.Sprintf("topic '%s' not found", bindingId))
			return serviceadapter.NewBindingNotFoundError(nil)
		}
		b.StderrLogger.Println("Error deleting topic: " + err.Error())
		return errors.New("")
	}

	return nil
}
