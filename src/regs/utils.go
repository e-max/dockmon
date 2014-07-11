package main

import (
	"fmt"
	"os"
	"strings"
)

func getLinkedContainers() ([]string, error) {
	names := []string{}

	for _, e := range os.Environ() {
		vals := strings.Split(e, "=")
		if len(vals) != 2 {
			return nil, fmt.Errorf("Wrong options %s", e)
		}
		k := strings.ToUpper(vals[0])
		v := vals[1]
		if strings.HasSuffix(k, "_NAME") {
			//names = append(names, strings.TrimSuffix(k, "_NAME"))
			names = append(names, v)
		}
	}
	return names, nil
}

func getEnvVariable(name string) (string, error) {
	for _, e := range os.Environ() {
		vals := strings.Split(e, "=")
		if len(vals) != 2 {
			return "", fmt.Errorf("Wrong options %s", e)
		}
		if strings.ToUpper(vals[0]) == strings.ToUpper(name) {
			return vals[1], nil
		}
	}
	return "", fmt.Errorf("Variable %s doesn't exist in enviroment")
}
