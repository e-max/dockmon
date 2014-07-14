package checker

import (
	"flag"
	"fmt"
	"os"
)

func _checkLinked() error {
	fmt.Println("==============================================")
	names, err := getLinkedContainers()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		return fmt.Errorf("No linked containers")
	}

	for _, name := range names {
		cont, err := ContainerByName(name)
		if err != nil {
			return err
		}
		err = cont.Check()
		if err != nil {
			return err
		}

	}

	return nil

}

func checkByName(cname string) error {
	cont, err := ContainerByName(cname)
	if err != nil {
		return err
	}
	err = cont.Check()
	if err != nil {
		return err
	}
	return nil
}

func checkAndRegister(cname string, etcdHost string) error {
	cont, err := ContainerByName(cname)
	if err != nil {
		return err
	}
	handler, err := GetMonitor(cont, etcdHost)
	if err != nil {
		return err
	}
	err = handler.Check()
	if err != nil {
		return err
	}
	err = handler.Register()

	return err

}

func startMonitoring(cname string, etcdHost string) error {
	cont, err := ContainerByName(cname)
	if err != nil {
		return err
	}
	handler, err := GetMonitor(cont, etcdHost)
	if err != nil {
		return err
	}
	handler.StartMonitoring()
	return nil
}

func main1() {
	InitLogger("DEBUG", true, true)
	logger.Debug("START ")
	var etcdHost string
	flag.StringVar(&etcdHost, "etcd-host", "localhost", "host where etcd is listenting")
	flag.Parse()
	//err := _checkLinked()
	cname := flag.Arg(0)
	fmt.Printf("cname %+v\n", cname)
	fmt.Printf("os.Args %+v\n", os.Args)
	//err := checkByName(cname)
	//err := checkAndRegister(cname, etcdHost)
	//if err != nil {
	//fmt.Println("Error ", err)
	//}
	err := startMonitoring(cname, etcdHost)
	fmt.Printf("err %#v\n", err)

}
