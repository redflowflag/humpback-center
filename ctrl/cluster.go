package ctrl

import "github.com/humpback/discovery"
import "github.com/humpback/gounits/http"
import "github.com/humpback/gounits/logger"
import "github.com/humpback/humpback-agent/models"
import "github.com/humpback/humpback-center/api/request"
import "github.com/humpback/humpback-center/cluster"
import "github.com/humpback/humpback-center/cluster/types"
import "github.com/humpback/humpback-center/etc"

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const (
	// humpback-api site request timeout value
	requestAPITimeout = 15 * time.Second
)

func createCluster(configuration *etc.Configuration) (*cluster.Cluster, error) {

	clusterOpts := configuration.Cluster
	heartbeat, err := time.ParseDuration(clusterOpts.Discovery.Heartbeat)
	if err != nil {
		return nil, fmt.Errorf("discovery heartbeat invalid.")
	}

	if heartbeat < 1*time.Second {
		return nil, fmt.Errorf("discovery heartbeat should be at least 1s.")
	}

	configopts := map[string]string{"kv.path": clusterOpts.Discovery.Cluster}
	discovery, err := discovery.New(clusterOpts.Discovery.URIs, heartbeat, 0, configopts)
	if err != nil {
		return nil, err
	}

	cluster, err := cluster.NewCluster(clusterOpts.DriverOpts, discovery)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *Controller) initCluster() {

	if groups := c.getClusterGroupStoreData(""); groups != nil {
		logger.INFO("[#ctrl#] init cluster groups:%d", len(groups))
		for _, group := range groups {
			c.Cluster.SetGroup(group.ID, group.Servers, group.Owners)
		}
	}
}

func (c *Controller) startCluster() error {

	c.initCluster()
	logger.INFO("[#ctrl#] start cluster.")
	return c.Cluster.Start()
}

func (c *Controller) stopCluster() {

	c.Cluster.Stop()
	logger.INFO("[#ctrl#] stop cluster.")
}

func (c *Controller) getClusterGroupStoreData(groupid string) []*cluster.Group {

	query := map[string][]string{}
	groupid = strings.TrimSpace(groupid)
	if groupid != "" {
		query["groupid"] = []string{groupid}
	}

	t := time.Now().UnixNano() / int64(time.Millisecond)
	value := fmt.Sprintf("HUMPBACK_CENTER%d", t)
	code := base64.StdEncoding.EncodeToString([]byte(value))
	header := map[string][]string{"x-get-cluster": []string{code}}
	respGroups, err := http.NewWithTimeout(requestAPITimeout).Get(c.Configuration.SiteAPI+"/groups/getclusters", query, header)
	if err != nil {
		logger.ERROR("[#ctrl#] get cluster group storedata error:%s", err.Error())
		return nil
	}

	defer respGroups.Close()
	if respGroups.StatusCode() != 200 {
		logger.ERROR("[#ctrl#] get cluster group storedata error:%d", respGroups.StatusCode())
		return nil
	}

	groups := []*cluster.Group{}
	if err := respGroups.JSON(&groups); err != nil {
		logger.ERROR("[#ctrl#] get cluster group storedata error:%s", err.Error())
		return nil
	}
	return groups
}

func (c *Controller) getEngineState(server string) string {

	state := cluster.GetStateText(cluster.StateDisconnected)
	if engine := c.Cluster.GetEngine(server); engine != nil {
		state = engine.State()
	}
	return state
}

func (c *Controller) SetCluster(cluster *cluster.Cluster) {

	if cluster != nil {
		logger.INFO("[#ctrl#] set cluster %p.", cluster)
		c.Cluster = cluster
	}
}

func (c *Controller) GetClusterGroupAllContainers(groupid string) *types.GroupContainers {

	return c.Cluster.GetGroupAllContainers(groupid)
}

func (c *Controller) GetClusterGroupContainers(metaid string) *types.GroupContainer {

	return c.Cluster.GetGroupContainers(metaid)
}

func (c *Controller) GetClusterGroupContainersMetaBase(metaid string) *cluster.MetaBase {

	return c.Cluster.GetMetaBase(metaid)
}

func (c *Controller) GetClusterGroupEngines(groupid string) []*cluster.Engine {

	return c.Cluster.GetGroupEngines(groupid)
}

func (c *Controller) GetClusterEngine(server string) *cluster.Engine {

	return c.Cluster.GetEngine(server)
}

func (c *Controller) SetClusterGroupEvent(groupid string, event string) {

	logger.INFO("[#ctrl#] set cluster groupevent %s.", event)
	switch event {
	case request.GROUP_CREATE_EVENT, request.GROUP_CHANGE_EVENT:
		{
			if groups := c.getClusterGroupStoreData(groupid); groups != nil {
				logger.INFO("[#ctrl#] get cluster groups:%d", len(groups))
				for _, group := range groups {
					c.Cluster.SetGroup(group.ID, group.Servers, group.Owners)
				}
			}
		}
	case request.GROUP_REMOVE_EVENT:
		{
			c.Cluster.RemoveGroup(groupid)
		}
	}
}

func (c *Controller) CreateClusterContainers(groupid string, instances int, webhooks types.WebHooks, config models.Container) (string, *types.CreatedContainers, error) {

	return c.Cluster.CreateContainers(groupid, instances, webhooks, config)
}

func (c *Controller) UpdateClusterContainers(metaid string, instances int, webhooks types.WebHooks) (*types.CreatedContainers, error) {

	return c.Cluster.UpdateContainers(metaid, instances, webhooks)
}

func (c *Controller) OperateContainers(metaid string, action string) (*types.OperatedContainers, error) {

	return c.Cluster.OperateContainers(metaid, "", action)
}

func (c *Controller) OperateContainer(containerid string, action string) (string, *types.OperatedContainers, error) {

	return c.Cluster.OperateContainer(containerid, action)
}

func (c *Controller) UpgradeContainers(metaid string, imagetag string) error {

	return c.Cluster.UpgradeContainers(metaid, imagetag)
}

func (c *Controller) RemoveContainers(metaid string) (*types.RemovedContainers, error) {

	return c.Cluster.RemoveContainers(metaid, "")
}

func (c *Controller) RemoveContainer(containerid string) (string, *types.RemovedContainers, error) {

	return c.Cluster.RemoveContainer(containerid)
}
