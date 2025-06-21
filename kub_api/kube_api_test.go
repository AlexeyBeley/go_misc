package kub_api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
)

const serviceNameConst string = "test-service"

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
	Namespace           *string           `json:"Namespace"`
	Image               *string           `json:"Image"`
	ImagePullSecretName *string           `json:"ImagePullSecretName"`
	IngressClassName    *string           `json:"IngressClassName"`
	RuleHost            *string           `json:"RuleHost"`
	TLSSecretName       *string           `json:"TLSSecretName"`
	Annotations         map[string]string `json:"Annotations"`
}

func loadRealConfig() *TestConfig {
	configFilePath := "/opt/kube_api_test.json"
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		panic(err)
	}
	config := TestConfig{}
	err = json.Unmarshal(data, &config)

	if err != nil {
		panic(err)
	}

	return &config
}

func TestGetNamespaces(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}

		namespaces, err := api.GetNamespaces()

		if err != nil {
			t.Errorf("%v", err)
		}
		for _, namespace := range namespaces {
			fmt.Printf("Namespaces: %s\n", namespace.Name)
		}
	})
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

		job := JobFlat{}

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

		job := JobFlat{}

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

		job := JobFlat{}

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

		selectors := map[string]string{
			"app": "test", // Select pods with the label "app: my-app"
		}
		labels := map[string]string{
			"app": "test", // Select pods with the label "app: my-app"
		}

		serviceF := &ServiceFlat{Name: strPtr(serviceNameConst),
			Port:     int32Ptr(8080),
			Selector: selectors,
			Labels:   labels,
		}
		api.CreateService(serviceF)

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

			for _, ser := range ret {
				if ser.Name != "core-broker-discovery" {
					continue
				}

				marshaled, err := ser.Marshal()
				log.Printf("%s: %s", namespace.Name, string(marshaled))
				_ = err
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
					fmt.Printf("Public ingres name: %s, class: %s\n", ingress.Name, *ingress.Spec.IngressClassName)
				}
			}

		}
	})
}

func TestListIngressClasses(t *testing.T) {
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
		ret, err := api.ListIngressClasses()

		if err != nil {
			t.Errorf("%v", err)
		}

		for _, ingressClass := range ret {
			fmt.Printf("IngressClassName: %s: %v\n ", ingressClass.GetName(), ingressClass.Spec)
		}

	})
}

func TestCreateDeployment(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		depF := &DeploymentFlat{AppName: strPtr("test"), Ports: []int32{8080}, Image: realConfig.Image, ImagePullSecretName: realConfig.ImagePullSecretName}
		err = api.CreateDeployment(depF)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestUpdateDeployment(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		depF := &DeploymentFlat{AppName: strPtr("test"), Ports: []int32{8080}, Image: realConfig.Image, ImagePullSecretName: realConfig.ImagePullSecretName}
		err = api.UpdateDeployment(depF)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestCreateIngress(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		strServiceNameConst := string(serviceNameConst)

		ingressF := &IngressFlat{Name: strPtr("test-ingress"),
			BackendServicePort: int32Ptr(8080),
			IngressClassName:   realConfig.IngressClassName,
			TLSSecretName:      realConfig.TLSSecretName,
			RuleHost:           realConfig.RuleHost,
			BackendServiceName: &strServiceNameConst,
			Annotations:        realConfig.Annotations,
		}

		err = api.CreateIngress(ingressF)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}



func TestUpdateIngress(t *testing.T) {
	t.Run("Valid run", func(t *testing.T) {
		realConfig := loadRealConfig()
		_ = realConfig

		api, err := KubAPINew()
		if err != nil {
			t.Errorf("%v", err)
		}
		api.Namespace = realConfig.Namespace

		strServiceNameConst := string(serviceNameConst)

		ingressF := &IngressFlat{Name: strPtr("test-ingress"),
			BackendServicePort: int32Ptr(8080),
			IngressClassName:   realConfig.IngressClassName,
			TLSSecretName:      realConfig.TLSSecretName,
			RuleHost:           realConfig.RuleHost,
			BackendServiceName: &strServiceNameConst,
			Annotations:        realConfig.Annotations,
		}

		err = api.UpdateIngress(ingressF)

		if err != nil {
			t.Errorf("%v", err)
		}
	})
}