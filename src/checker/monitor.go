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
	handler := &ContainerMonitor{container, eclient, uint64(container.healthcheckPeriod*2 + 1), nil, nil}
	return handler, nil
}

//StartMonitoring wrapped container
func (m *ContainerMonitor) StartMonitoring() {
	logger.Info("Start monitoring container %s", m)
	m.ticker = time.NewTicker(time.Duration(m.healthcheckPeriod) * time.Second)
	m.stop = make(chan bool, 1)
	for {
		select {
		case <-m.ticker.C:
			err := m.Check()
			if err != nil {
				switch err.(type) {
				case *docker.NoSuchContainer:
					logger.Warning("%s", err)
					m.StopMonitoring()
				case *NotSupportCheckError:
					logger.Warning("%s", err)
					m.StopMonitoring()
				default:
					logger.Info("Got error while check container %s: %s", m, err)
				}

			} else {
				err := m.Register()
				if err != nil {
					logger.Error("Cannot update info about container in etcd: %s", err)
				}
			}
		case <-m.stop:
			logger.Debug("Got stop signal %s", m)
			return
		}
	}
}

//StopMonitoring wrapped container
func (m *ContainerMonitor) StopMonitoring() {
	logger.Info("Stop monitoring container %s", m)
	if m.ticker != nil {
		m.ticker.Stop()
		m.stop <- true
	}
}

//Register service provided by container in etcd
func (m *ContainerMonitor) Register() error {
	path := strings.Split(m.Container.Config.Image, "/")
	service := path[len(path)-1]
	key := fmt.Sprintf("/service/%s/%s", service, m.Container.ID)

	ip := m.Container.NetworkSettings.IPAddress
	name := m.Container.Name
	ports := m.Container.NetworkSettings.PortMappingAPI()

	logger.Debug("Register container %s with ip = %s (ttl = %s)", m, ip, m.etcdTTL)

	val, err := json.Marshal(ServiceInfo{ip, name, ports})

	if err != nil {
		return err
	}

	_, err = m.Set(key, string(val), m.etcdTTL)

	if err != nil {
		return err
	}

	_, err = m.UpdateDir(fmt.Sprintf("/service/%s/", service), m.etcdTTL)

	if err != nil {
		return err
	}

	return nil
}
