package kub_api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
)

func LoadDynamicConfig() (config any, err error) {
	configFilePath := "/opt/kube_api_test.json"
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

type TestConfig struct {
	Namespace *string
}

func (testConfig *TestConfig) InitFromM(source any) error {
	mapValues, sucess := source.(map[string]any)
	if !sucess {
		panic(source)
	}

	namespace, sucess := mapValues["Namespace"].(string)
	if !sucess {
		panic(source)
	}

	(*testConfig).Namespace = &namespace
	return nil
}

func loadRealConfig() *TestConfig {
	config, err := LoadDynamicConfig()
	if err != nil {
		log.Fatalf("%v", err)
	}

	testConfig := TestConfig{}
	err = testConfig.InitFromM(config)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return &testConfig
}

func TestCreateJob(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		job := Job{}

		name := "test"
		containerImage := "busybox:1.28"
		containerCommand := []string{
			"/bin/sh",
			"-c",
			"echo Hello from Kubernetes Job! && sleep 5", // Simple command
		}

		job.JobName = &name
		job.ContainerName = &name
		job.ContainerImage = &containerImage
		job.ContainerCommand = &containerCommand
		tempZero := int32(0)
		job.TTLSecondsAfterFinished = &tempZero

		err = api.CreateJob(&job)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestDeleteJob(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		job := Job{}

		name := "test"
		containerImage := "busybox:1.28"
		containerCommand := []string{
			"/bin/sh",
			"-c",
			"echo Hello from Kubernetes Job! && sleep 5", // Simple command
		}

		job.JobName = &name
		job.ContainerName = &name
		job.ContainerImage = &containerImage
		job.ContainerCommand = &containerCommand

		err = api.DeleteJob(&job)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestCreatePod(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		job := Job{}

		name := "test"
		containerImage := "busybox:1.28"
		containerCommand := []string{
			"/bin/sh",
			"-c",
			"echo Hello from Kubernetes Job! && sleep 5", // Simple command
		}

		job.JobName = &name
		job.ContainerName = &name
		job.ContainerImage = &containerImage
		job.ContainerCommand = &containerCommand

		for podId := range 10 {
			err = api.CreatePod(&job, strconv.Itoa(podId))

		}

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestListPods(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace
		ret, err := api.GetPods()
		if err != nil {
			t.Errorf("%v", err)
		}
		for _, obj := range ret {
			fmt.Printf("Pod: %v\n", obj.Spec)

		}
	})
}

func TestPrunePods(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace
		jobName := "test"
		api.PrunePods(&jobName)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestProvisionNamespace(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		api.ProvisionNamespace(realConfig.Namespace)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestCreateService(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace
		port := 80
		serviceName := "test"
		selector := map[string]string{
			"app": "my-app", // Select pods with the label "app: my-app"
		}
		api.CreateService(&serviceName, int32(port), selector)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestGetServices(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		ret, err := api.GetServices()

		if err != nil {
			t.Errorf("%v", err)
		}
		if len(ret) == 0 {
			t.Errorf("No services found %v", ret)
		}
	})
}

func TestGetAllNamespacesServices(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace
		namespaces, err := api.GetNamespaces()
		if err != nil {
			t.Errorf("%v", err)
		}
		_ = namespaces
		for _, namespace := range namespaces {

			api.Namespace = &namespace.Name
			ret, err := api.GetServices()

			if err != nil {
				t.Errorf("%v", err)
			}

			log.Printf("%s: %d", namespace.Name, len(ret))

		}
	})
}

func TestGetAllIngresses(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace
		namespaces, err := api.GetNamespaces()
		if err != nil {
			t.Errorf("%v", err)
		}
		_ = namespaces
		for _, namespace := range namespaces {

			api.Namespace = &namespace.Name
			ret, err := api.GetIngresses()

			if err != nil {
				t.Errorf("%v", err)
			}

			if len(ret) > 0 {
				fmt.Printf("Ingresses in Namespace %s: %d\n", namespace.Name, len(ret))
			}

			for _, ingress := range ret {
				fmt.Printf("Ingress: %s, IngressClassName: %s\n ", *&ingress.Name, *ingress.Spec.IngressClassName)

				if *ingress.Spec.IngressClassName == "nginx-public" {
					fmt.Print(ingress.String() + "\n")
				}
			}

		}
	})
}
