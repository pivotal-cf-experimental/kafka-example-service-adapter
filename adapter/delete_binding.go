package adapter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

func (b *Binder) DeleteBinding(params serviceadapter.DeleteBindingParams) error {

	zookeeperServers := params.DeploymentTopology["zookeeper_server"]
	if len(zookeeperServers) == 0 {
		b.StderrLogger.Println("no VMs for job zookeeper_server")
		return errors.New("")
	}

	if _, errorStream, err := b.Run(b.TopicDeleterCommand, strings.Join(zookeeperServers, ","), params.BindingID); err != nil {
		if strings.Contains(string(errorStream), fmt.Sprintf("Topic %s does not exist on ZK path", params.BindingID)) {
			b.StderrLogger.Println(fmt.Sprintf("topic '%s' not found", params.BindingID))
			return serviceadapter.NewBindingNotFoundError(nil)
		}
		b.StderrLogger.Println("Error deleting topic: " + err.Error())
		return errors.New("")
	}

	return nil
}
