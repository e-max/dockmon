package checker

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
)

//ServiceInfo represent information about service which we share by etcd
type ServiceInfo struct {
	IP    string
	Name  string
	Ports []docker.APIPort
}

//ContainerMonitor periodically check if container is alive and work properly
type ContainerMonitor struct {
	*Container
	*etcd.Client
	etcdTTL uint64
	ticker  *time.Ticker
	stop    chan bool
}

//GetMonitor return new monitor for container
func GetMonitor(container *Container, etcdHost string) (*ContainerMonitor, error) {
	machines := []string{etcdHost}
	logger.Debug("Create etcd client connected to %+v", machines)
	eclient := etcd.NewClient(machines)
	handler := &ContainerMonitor{container, eclient, uint64(container.healthcheckttl*2 + 1), nil, nil}
	return handler, nil
}

//StartMonitoring wrapped container
func (h *ContainerMonitor) StartMonitoring() {
	logger.Info("Start monitoring container %s", h)
	h.ticker = time.NewTicker(h.healthcheckttl)
	h.stop = make(chan bool, 1)
	for {
		select {
		case <-h.ticker.C:
			err := h.Check()
			if err != nil {
				switch err.(type) {
				case *docker.NoSuchContainer:
					logger.Warning("%s", err)
					h.StopMonitoring()
				case *NotSupportCheckError:
					logger.Warning("%s", err)
					h.StopMonitoring()
				default:
					logger.Info("Got error while check container %s: %s", h, err)
				}

			} else {
				h.Register()
			}
		case <-h.stop:
			logger.Debug("Got stop signal %s", h)
			return
		}
	}
}

//StopMonitoring wrapped container
func (h *ContainerMonitor) StopMonitoring() {
	logger.Info("Stop monitoring container %s", h)
	if h.ticker != nil {
		h.ticker.Stop()
		h.stop <- true
	}
}

//Register service provided by container in etcd
func (h *ContainerMonitor) Register() error {
	path := strings.Split(h.Container.Config.Image, "/")
	service := path[len(path)-1]
	key := fmt.Sprintf("/service/%s/%s", service, h.Container.ID)

	ip := h.Container.NetworkSettings.IPAddress
	name := h.Container.Name
	ports := h.Container.NetworkSettings.PortMappingAPI()

	logger.Debug("Register container %s with ip = %s (ttl = %s)", h, ip, h.etcdTTL)

	val, err := json.Marshal(ServiceInfo{ip, name, ports})

	if err != nil {
		return err
	}

	_, err = h.Set(key, string(val), h.etcdTTL)

	if err != nil {
		return err
	}

	_, err = h.UpdateDir(fmt.Sprintf("/service/%s/", service), h.etcdTTL)

	if err != nil {
		return err
	}

	return nil
}
