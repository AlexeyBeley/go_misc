package kub_api

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubAPI struct {
	Kubeconfig   *string
	clientset    *kubernetes.Clientset
	Namespace    *string
	FieldManager *string
}

type JobFlat struct {
	JobName                 *string
	ContainerName           *string
	ContainerImage          *string
	ContainerCommand        *[]string
	TTLSecondsAfterFinished *int32
	UID                     *types.UID
}

type DeploymentFlat struct {
	AppName             *string
	Namespace           *string
	Image               *string
	ImagePullSecretName *string
	Ports               []int32
	ServiceAccount      *string
}

func (depf *DeploymentFlat) GenerateRequest() (*appsv1.Deployment, error) {
	containerPorts := []corev1.ContainerPort{}
	for _, port := range depf.Ports {
		var name string
		switch port {
		case 80:
			name = "http"
		case 443:
			name = "https"
		default:
			name = "tcp" + strconv.Itoa(int(port))
		}

		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: port,
			Name:          name, // Named port for service/ingress target
		})
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",    // <-- Ensure this is correct
			Kind:       "Deployment", // <-- Ensure this is correct
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      *depf.AppName + "-deployment",
			Namespace: *depf.Namespace,
			Labels:    map[string]string{"app": *depf.AppName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1), // One replica for simplicity
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": *depf.AppName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": *depf.AppName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  *depf.AppName + "-container",
							Image: *depf.Image,
							Ports: containerPorts,
						},
					},
					ImagePullSecrets:   []corev1.LocalObjectReference{corev1.LocalObjectReference{Name: *depf.ImagePullSecretName}},
					ServiceAccountName: *depf.ServiceAccount,
				},
			},
		},
	}
	return deployment, nil
}

type ServiceAccountFlat struct {
	Name      *string
	Namespace *string
}

func (serviceAccountFlat *ServiceAccountFlat) GenerateRequest() (*corev1.ServiceAccount, error) {

	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: *serviceAccountFlat.Name}}
	return serviceAccount, nil
}

type RoleFlat struct {
	Name      *string
	Namespace *string
	Rules     []rbacv1.PolicyRule
}

func (roleFlat *RoleFlat) GenerateRequest() (*rbacv1.Role, error) {

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: *roleFlat.Name,
		},
		Rules: roleFlat.Rules}

	return role, nil
}

type RoleBindingFlat struct {
	Name               *string
	Namespace          *string
	ServiceAccountName *string
	RoleName           *string
}

func (roleBindingFlat *RoleBindingFlat) GenerateRequest() (*rbacv1.RoleBinding, error) {

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *roleBindingFlat.Name,
			Namespace: *roleBindingFlat.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      *roleBindingFlat.ServiceAccountName,
				Namespace: *roleBindingFlat.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     *roleBindingFlat.RoleName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return roleBinding, nil
}

type IngressFlat struct {
	Name               *string
	Namespace          *string
	IngressClassName   *string
	RuleHost           *string
	BackendServiceName *string
	BackendServicePort *int32
	TLSSecretName      *string
	Annotations        map[string]string
}

func (ingressFlat *IngressFlat) GenerateRequest() (*networkingv1.Ingress, error) {
	ingress := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        *ingressFlat.Name,
			Namespace:   *ingressFlat.Namespace, // Make sure this namespace exists in your cluster
			Annotations: ingressFlat.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ingressFlat.IngressClassName, // This should match your Ingress Controller's class
			Rules: []networkingv1.IngressRule{
				{
					Host: *ingressFlat.RuleHost,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(), // Use helper for pointer to PathType
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: *ingressFlat.BackendServiceName, // Name of the Service your Ingress routes to
											Port: networkingv1.ServiceBackendPort{
												Number: *ingressFlat.BackendServicePort, // Port of that Service
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{ // TLS configuration
				{
					Hosts: []string{
						*ingressFlat.RuleHost,
					},
					SecretName: *ingressFlat.TLSSecretName, // As per provided YAML, SecretName is omitted.
					// For networking.k8s.io/v1, SecretName is a string field.
					// An empty string here would imply using a default certificate
					// configured on the Nginx Ingress Controller if it supports it,
					// or it might indicate a missing Secret for TLS if the controller
					// doesn't have a default.
				},
			},
		},
	}
	return ingress, nil
}

type ServiceFlat struct {
	Name      *string
	Namespace *string
	Port      *int32
	Selector  map[string]string
	Labels    map[string]string
}

func (serviceFlat *ServiceFlat) GenerateRequest() (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *serviceFlat.Name,
			Namespace: *serviceFlat.Namespace,
			Labels:    serviceFlat.Labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: serviceFlat.Selector,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     *serviceFlat.Port,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP, // Use a ClusterIP for internal access
		},
	}
	return service, nil
}

func (job *JobFlat) GenerateRequest() (ret *batchv1.Job, err error) {
	ret = new(batchv1.Job)
	*ret = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: *job.JobName,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: job.TTLSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure, // Recommended for Jobs
					Containers: []corev1.Container{
						{
							Name:    *job.ContainerName,
							Image:   *job.ContainerImage,
							Command: *job.ContainerCommand,
						},
					},
				},
			},
		},
	}
	return ret, nil
}

type SecretFlat struct {
	Name        *string
	Namespace   *string
	Labels      map[string]string
	Annotations map[string]string
	Data        map[string][]byte
	StringData  map[string]string
	Type        *string
}

func (secretFlat *SecretFlat) GenerateRequest() (*corev1.Secret, error) {

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        *secretFlat.Name,       // Keep the same name
			Namespace:   *secretFlat.Namespace,  // <--- Set the new namespace
			Labels:      secretFlat.Labels,      // Copy existing labels
			Annotations: secretFlat.Annotations, // Copy existing annotations (optional, consider filtering)
		},
		Type: corev1.SecretType(*secretFlat.Type), // Copy the secret type (e.g., Opaque, kubernetes.io/tls)
	}

	if secretFlat.StringData != nil {
		(*newSecret).StringData = secretFlat.StringData // Copy StringData if present (mutually exclusive with Data for creation)
	}
	if secretFlat.Data != nil {
		(*newSecret).Data = secretFlat.Data // Copy the actual secret data (base64 encoded bytes)
	}
	return newSecret, nil
}

type NamespaceFlat struct {
	Name   *string
	Labels map[string]string
}

func (namespaceFlat *NamespaceFlat) GenerateRequest() (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1", // Namespace is in the core API group, version v1
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   *namespaceFlat.Name,
			Labels: namespaceFlat.Labels,
		},
	}

	return namespace, nil
}

func KubAPINew() (*KubAPI, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	namespace := flag.String("namespace", "default", "namespace to list pods in")
	flag.Parse()

	ret := KubAPI{Kubeconfig: kubeconfig, Namespace: namespace}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	// Create a Kubernetes kapi.clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating kapi.clientset: %v\n", err)
		os.Exit(1)
	}
	ret.clientset = clientset
	ret.FieldManager = strPtr("horey_kub_api-go-updater")

	return &ret, nil
}

func (kapi *KubAPI) GetPods() ([]corev1.Pod, error) {

	// List pods in the specified namespace
	pods, err := kapi.clientset.CoreV1().Pods(*kapi.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing pods in namespace '%s': %v\n", *kapi.Namespace, err)
		return nil, err
	}

	return pods.Items, nil
}

func (kapi *KubAPI) GetDeployments() ([]appsv1.Deployment, error) {

	// List pods in the specified namespace
	deployments, err := kapi.clientset.AppsV1().Deployments(*kapi.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing Deployments in namespace '%s': %v\n", *kapi.Namespace, err)
		return nil, err
	}

	return deployments.Items, nil
}

func (kapi *KubAPI) GetNamespaces() ([]corev1.Namespace, error) {
	// List pods in the specified namespace
	namespaces, err := kapi.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing namespaces: %v\n", err)
		return nil, err
	}

	return namespaces.Items, nil
}

func (kapi *KubAPI) GetActiveNamespace() (ret *string, err error) {
	if *kapi.Namespace == "default" {
		return ret, fmt.Errorf("active namespace was not set")
	}
	return kapi.Namespace, nil
}

func (kapi *KubAPI) CreateJob(job *JobFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}
	batchJob, err := job.GenerateRequest()
	if err != nil {
		return err
	}
	batchJob.ObjectMeta.Namespace = *namespace

	createdJob, err := kapi.clientset.BatchV1().Jobs(*kapi.Namespace).Create(context.TODO(), batchJob, metav1.CreateOptions{})

	if err != nil {
		fmt.Printf("Error Creating Job: %v\n", err)
		return err
	}
	job.UID = &createdJob.UID
	fmt.Printf("Job created successfully! Name: %s, Namespace: %s\n", createdJob.Name, createdJob.Namespace)
	return nil
}

func (kapi *KubAPI) DeleteJob(job *JobFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}
	batchJob, err := job.GenerateRequest()
	if err != nil {
		panic(fmt.Sprintf("Error generating request: %v\n", err))
	}

	batchJob.ObjectMeta.Namespace = *namespace

	err = kapi.clientset.BatchV1().Jobs(*kapi.Namespace).Delete(context.TODO(), *job.JobName, metav1.DeleteOptions{})

	if err != nil {
		panic(fmt.Sprintf("Error deleting job: %v\n", err))
	}

	fmt.Printf("Job deleted successfully! Name: %s, Namespace: %s\n", *job.JobName, *kapi.Namespace)
	return nil
}

func (kapi *KubAPI) CreatePod(job *JobFlat, podID string) error {
	podName := fmt.Sprintf("%s-%s-%s", *job.JobName, *job.JobName, podID)
	batchv1JobP, err := kapi.Getbatchv1Job(job)
	if err != nil {
		return err
	}

	// Add a label to the pod template that indicates the pod ordinal.
	if batchv1JobP.Spec.Template.ObjectMeta.Labels == nil {
		batchv1JobP.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	}
	batchv1JobP.Spec.Template.ObjectMeta.Labels["controller-type"] = "job"

	//create pod object.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: *kapi.Namespace,
			Labels: map[string]string{
				"job-name":       *job.JobName,
				"controller-uid": string(batchv1JobP.UID),
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(batchv1JobP, batchv1.SchemeGroupVersion.WithKind("Job")),
			},
		},
		Spec: batchv1JobP.Spec.Template.Spec, //use the pod spec from the job
	}
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name: "POD_ORDINAL",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['job-index']", // Get the ordinal from the label
			},
		},
	})

	corev1Pod, err := kapi.clientset.CoreV1().Pods(*kapi.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		fmt.Printf("Error creating pod %s: %v\n", podName, err)
		return err
	}
	_ = corev1Pod

	return nil
}

func (kapi *KubAPI) PrunePods(jobName *string) error {
	allPods, err := kapi.clientset.CoreV1().Pods(*kapi.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "job-name=" + *jobName, // Select pods created by this job
	})
	if err != nil {
		return err
	}
	podCount := len(allPods.Items)
	_ = allPods
	podWatch, err := kapi.clientset.CoreV1().Pods(*kapi.Namespace).Watch(context.TODO(), metav1.ListOptions{
		LabelSelector: "job-name=" + *jobName, // Select pods created by this job
	})
	if err != nil {
		return err
	}
	defer podWatch.Stop()
	podsDeleted := 0
	for event := range podWatch.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			fmt.Printf("Unexpected type from Pod watcher: %v\n", event.Object)
			continue // Don't exit, just skip this event
		}

		switch pod.Status.Phase {
		case corev1.PodSucceeded, corev1.PodFailed:
			fmt.Printf("Pod %s finished with status: %s, deleting...\n", pod.Name, pod.Status.Phase)
			deletePolicy := metav1.DeletePropagationForeground
			err := kapi.clientset.CoreV1().Pods(*kapi.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
			if err != nil {
				fmt.Printf("Error deleting Pod %s: %v\n", pod.Name, err)
				// Log the error and continue, don't exit.  Deletion might fail due to network issues,
				// but we want to try to delete other pods.
			} else {
				podsDeleted++
				fmt.Printf("Deleted Pod %s\n", pod.Name)
			}
		}
		if podsDeleted >= int(podCount) {
			fmt.Println("All pods have been deleted.")
			break
		}
	}
	return nil
}

func (kapi *KubAPI) Getbatchv1Job(job *JobFlat) (*batchv1.Job, error) {
	ret, err := kapi.clientset.BatchV1().Jobs(*kapi.Namespace).Get(context.TODO(), *job.JobName, metav1.GetOptions{})
	return ret, err
}

func (kapi *KubAPI) GetLogs() {
	/*
	   // List pods in the specified namespace
	   //pods, err := kapi.clientset.CoreV1().GetLogs() (*kapi.Namespace).List(context.TODO(), metav1.ListOptions{})

	   	if err != nil {
	   		fmt.Printf("Error listing pods in namespace '%s': %v\n", *kapi.Namespace, err)
	   		os.Exit(1)
	   	}

	   fmt.Printf("Pods in namespace '%s':\n", *kapi.Namespace)

	   	for _, pod := range pods.Items {
	   		fmt.Printf("- Name: %s, Status: %s\n", pod.Name, pod.Status.Phase)
	   	}
	*/
}

func (kapi *KubAPI) CreateService(serviceF *ServiceFlat) error {
	// Create the Service
	// Define the Service object
	serviceF.Namespace = kapi.Namespace
	service, err := serviceF.GenerateRequest()
	if err != nil {
		return err
	}
	createdService, err := kapi.clientset.CoreV1().Services(*kapi.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Error creating Service: %v\n", err)
		return err
	}

	fmt.Printf("Service created successfully! Name: %s, Namespace: %s\n", createdService.Name, createdService.Namespace)
	return nil
}

func (kapi *KubAPI) GetServices() (ret []corev1.Service, err error) {
	// List Services in the specified namespace
	services, err := kapi.clientset.CoreV1().Services(*kapi.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing Services in namespace '%s': %v\n", *kapi.Namespace, err)
		return nil, err
	}

	return services.Items, nil
}

func (kapi *KubAPI) GetIngresses() ([]networkingv1.Ingress, error) {
	// List Services in the specified namespace
	ingressList, err := kapi.clientset.NetworkingV1().Ingresses(*kapi.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing Ingresses in namespace %s: %v\n", *kapi.Namespace, err)
		return nil, err
	}

	return ingressList.Items, nil
}

func (kapi *KubAPI) ProvisionRole(role *rbacv1.Role) error {
	// List Services in the specified namespace
	// 2. Create a Role

	_, err := kapi.clientset.RbacV1().Roles(*kapi.Namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Error creating Role: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Role created successfully")
	return nil
}

func (kapi *KubAPI) ProvisionServiceAccount(serviceAccount *corev1.ServiceAccount) error {
	// List Services in the specified namespace
	// 2. Create a Role

	_, err := kapi.clientset.CoreV1().ServiceAccounts(*kapi.Namespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Error creating Role: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Service Account created successfully")
	return nil
}

func (kapi *KubAPI) ProvisionRoleBinding(roleBinding *rbacv1.RoleBinding) error {
	// 3. Create a RoleBinding

	_, err := kapi.clientset.RbacV1().RoleBindings(*kapi.Namespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Error creating RoleBinding: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("RoleBinding created successfully")
	return nil
}

func (kapi *KubAPI) ProvisionNamespace(name *string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: *name,
		},
	}

	namespace, err := kapi.clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Created namespace: %s, %s\n", *name, namespace.UID)
		return err
	}
	return nil
}

type JSONPatchOperation struct {
	Op    string      `json:"op"`              // "remove", "add", "replace", "test", etc.
	Path  string      `json:"path"`            // JSON Pointer (e.g., "/metadata/labels/my-label-key")
	Value interface{} `json:"value,omitempty"` // Value for "add" or "replace" operations
}

func (kapi *KubAPI) UpdateNamespace(namespaceFlat *NamespaceFlat, declarative bool) error {
	var patchBytes []byte
	var requestType types.PatchType
	var err error
	patchOptions := metav1.PatchOptions{
		FieldManager: *kapi.FieldManager, // <--- The name identifying your declarative client
	}

	if declarative {
		// todo: Fix this!
		request := []JSONPatchOperation{
			{
				Op:   "remove",                                   // The "remove" operation
				Path: fmt.Sprintf("/metadata/labels/%s", "test"), // JSON Pointer to the specific label
			},
		}

		patchBytes, err = json.Marshal(request)
		if err != nil {
			log.Fatalf("Error marshaling desired Namespace to JSON: %v", err)
		}
		requestType = types.JSONPatchType
	} else {
		request, err := namespaceFlat.GenerateRequest()
		if err != nil {
			return err
		}
		patchBytes, err = json.Marshal(request)
		if err != nil {
			log.Fatalf("Error marshaling desired Namespace to JSON: %v", err)
		}
		requestType = types.ApplyPatchType
		patchOptions.Force = boolPtr(true) // <--- Set force to true for initial adoption or conflicts

	}

	log.Printf("DEBUG: Patch JSON: %s", string(patchBytes)) // For debugging the payload

	namespace, err := kapi.clientset.CoreV1().Namespaces().Patch(context.TODO(), *namespaceFlat.Name, requestType, patchBytes,
		patchOptions)

	if err != nil {
		fmt.Printf("Created namespace: %s, %s\n", *namespaceFlat.Name, namespace.UID)
		return err
	}
	return nil
}

func (kapi *KubAPI) ListIngressClasses() ([]networkingv1.IngressClass, error) {
	// NetworkingV1() gives access to the networking.k8s.io/v1 API group
	// IngressClasses() gives access to the IngressClass resource
	// List() retrieves a list of these resources
	ingressClassList, err := kapi.clientset.NetworkingV1().IngressClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing IngressClasses: %w", err)
	}
	return ingressClassList.Items, nil
}

func int32Ptr(i int32) *int32 {
	return &i
}

func strPtr(src string) *string {
	return &src
}

func boolPtr(src bool) *bool {
	return &src
}

func (kapi *KubAPI) CreateDeployment(deploymentFlat *DeploymentFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}

	if deploymentFlat.Namespace == nil {
		deploymentFlat.Namespace = kapi.Namespace
	}

	deploymentAPI, err := deploymentFlat.GenerateRequest()
	if err != nil {
		return err
	}
	_, err = kapi.clientset.AppsV1().Deployments(*namespace).Create(context.TODO(), deploymentAPI, metav1.CreateOptions{})
	if err != nil {
		if os.IsExist(err) {
			fmt.Printf("Deployment '%s' already exists. Skipping creation.\n", deploymentAPI.Name)
		} else {
			return fmt.Errorf("rrror creating Deployment: %v", err)
		}
	} else {
		fmt.Println("Deployment created successfully.")
	}
	return nil
}

func (kapi *KubAPI) UpdateDeployment(deploymentFlat *DeploymentFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}

	if deploymentFlat.Namespace == nil {
		deploymentFlat.Namespace = kapi.Namespace
	}

	request, err := deploymentFlat.GenerateRequest()
	if err != nil {
		return err
	}
	patchBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling desired Deployment to JSON: %v", err)
	}

	updatedDeployment, err := kapi.clientset.AppsV1().Deployments(*namespace).Patch(context.TODO(), request.Name, types.ApplyPatchType, patchBytes,
		metav1.PatchOptions{
			FieldManager: *kapi.FieldManager, // Provide the FieldManager
			Force:        boolPtr(true),      // Use force=true for initial adoption or to resolve conflicts
		})
	if err != nil {
		return fmt.Errorf("error performing Server-Side Apply patch on Deployment: %v", err)
	}

	fmt.Printf("Deployment '%s' updated successfully!\n", updatedDeployment.Name)
	fmt.Printf("New Image: %s\n", updatedDeployment.Spec.Template.Spec.Containers[0].Image)
	fmt.Printf("New Replicas: %d\n", *updatedDeployment.Spec.Replicas)
	return nil
}

func (kapi *KubAPI) CreateIngress(ingressFlat *IngressFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}

	if ingressFlat.Namespace == nil {
		ingressFlat.Namespace = namespace
	}

	ingress, err := ingressFlat.GenerateRequest()
	if err != nil {
		return err
	}
	_, err = kapi.clientset.NetworkingV1().Ingresses(ingress.Namespace).Create(context.TODO(), ingress, metav1.CreateOptions{})
	if err != nil {
		if os.IsExist(err) {
			fmt.Printf("Ingress '%s' already exists. Skipping creation.\n", ingress.Name)
		} else {
			return fmt.Errorf("error creating Ingress: %v", err)
		}
	} else {
		fmt.Println("Ingress created successfully.")
	}
	return nil
}

func (kapi *KubAPI) UpdateIngress(ingressFlat *IngressFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}

	if ingressFlat.Namespace == nil {
		ingressFlat.Namespace = namespace
	}

	desiredIngress, err := ingressFlat.GenerateRequest()
	if err != nil {
		return err
	}
	// --- Step 2: Marshal the desired Ingress object to JSON bytes ---
	patchBytes, err := json.Marshal(desiredIngress)
	if err != nil {
		return fmt.Errorf("error marshaling desired Ingress to JSON: %v", err)
	}

	// --- Step 3: Perform the Server-Side Apply Patch ---

	fmt.Printf("Updating Ingress '%s' in namespace '%s'\n",
		*ingressFlat.Name, *namespace)

	updatedIngress, err := kapi.clientset.NetworkingV1().Ingresses(*namespace).Patch(
		context.TODO(),
		*ingressFlat.Name,
		types.ApplyPatchType, // Specify Server-Side Apply patch type
		patchBytes,
		metav1.PatchOptions{
			FieldManager: *kapi.FieldManager,
			Force:        boolPtr(true), // Force initial adoption/overwrite conflicts
		},
	)
	if err != nil {
		return fmt.Errorf("error performing Server-Side Apply patch on Ingress: %v", err)
	}

	fmt.Printf("Ingress '%s' updated successfully!\n", updatedIngress.Name)
	fmt.Printf("New Host: %s\n", updatedIngress.Spec.Rules[0].Host)
	fmt.Printf("New Path: %s\n", updatedIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Path)
	fmt.Printf("New proxy-read-timeout: %s\n", updatedIngress.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"])
	return nil
}

func (kapi *KubAPI) CreateServiceAccount(serviceAccountFlat *ServiceAccountFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}

	if serviceAccountFlat.Namespace == nil {
		serviceAccountFlat.Namespace = namespace
	}

	serviceAccount, err := serviceAccountFlat.GenerateRequest()
	if err != nil {
		return err
	}

	_, err = kapi.clientset.CoreV1().ServiceAccounts(*serviceAccountFlat.Namespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{})
	if err != nil {
		if os.IsExist(err) {
			fmt.Println("ServiceAccount already exists.")
		} else {
			return fmt.Errorf("error creating ServiceAccount: %w", err)
		}
	} else {
		fmt.Println("ServiceAccount created.")
	}
	return nil
}

func (kapi *KubAPI) CreateRole(roleFlat *RoleFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}

	if roleFlat.Namespace == nil {
		roleFlat.Namespace = namespace
	}

	role, err := roleFlat.GenerateRequest()
	if err != nil {
		return err
	}

	_, err = kapi.clientset.RbacV1().Roles(*roleFlat.Namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		if os.IsExist(err) {
			fmt.Println("Role already exists.")
		} else {
			return fmt.Errorf("error creating Role: %w", err)
		}
	} else {
		fmt.Println("Role created.")
	}
	return nil
}

func (kapi *KubAPI) CreateRoleBinding(roleBindingFlat *RoleBindingFlat) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}

	if roleBindingFlat.Namespace == nil {
		roleBindingFlat.Namespace = namespace
	}

	role, err := roleBindingFlat.GenerateRequest()
	if err != nil {
		return err
	}

	_, err = kapi.clientset.RbacV1().RoleBindings(*roleBindingFlat.Namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		if os.IsExist(err) {
			fmt.Println("RoleBinding already exists.")
		} else {
			return fmt.Errorf("error creating RoleBinding: %w", err)
		}
	} else {
		fmt.Println("RoleBinding created.")
	}
	return nil
}

func (kapi *KubAPI) CopySecret(srcSecretFlat, dstSecretFlat *SecretFlat) error {

	fmt.Printf("Attempting to copy Secret '%s' from namespace '%s' to '%s' in namespace %s...\n",
		srcSecretFlat.Labels, *srcSecretFlat.Namespace, *dstSecretFlat.Name, *srcSecretFlat.Namespace)

	// --- Step 1: Get the Secret from the source namespace ---
	sourceSecret, err := kapi.clientset.CoreV1().Secrets(*srcSecretFlat.Namespace).Get(context.TODO(), *srcSecretFlat.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting secret '%s' from namespace '%s': %v", *srcSecretFlat.Name, *srcSecretFlat.Namespace, err)
	}
	fmt.Printf("Successfully retrieved Secret '%s' from '%s'.\n", sourceSecret.Name, sourceSecret.Namespace)

	// --- Step 2: Prepare the new Secret object for the destination namespace ---
	// Create a new Secret object based on the retrieved one.
	// IMPORTANT: Clear cluster-specific and read-only metadata fields.
	if dstSecretFlat.Type == nil {
		dstSecretFlat.Type = strPtr(string(sourceSecret.Type))
	}
	newSecretRequest, err := dstSecretFlat.GenerateRequest()
	if err != nil {
		return err
	}
	newSecretRequest.Data = sourceSecret.Data
	newSecretRequest.StringData = sourceSecret.StringData
	newSecretRequest.Type = sourceSecret.Type

	// If you want to remove specific annotations/labels that are source-namespace specific,
	// you would do it here. E.g., delete "meta.helm.sh/release-namespace" if copying a Helm-managed secret.
	// delete(newSecret.ObjectMeta.Annotations, "meta.helm.sh/release-namespace")

	// --- Step 3: Create the new Secret in the destination namespace ---
	fmt.Printf("Creating Secret '%s' in destination namespace '%s'...\n", newSecretRequest.Name, newSecretRequest.Namespace)
	createdSecret, err := kapi.clientset.CoreV1().Secrets(newSecretRequest.Namespace).Create(context.TODO(), newSecretRequest, metav1.CreateOptions{})
	if err != nil {
		if os.IsExist(err) {
			log.Printf("Secret '%s' already exists in namespace '%s'. Use kubectl delete to remove it first if you want to overwrite.", newSecretRequest.Name, newSecretRequest.Namespace)
			return fmt.Errorf("secret creation failed: Secret already exists")
		} else {
			return fmt.Errorf("error creating secret '%s' in namespace '%s': %v", newSecretRequest.Name, newSecretRequest.Namespace, err)
		}
	}

	fmt.Printf("Successfully copied Secret '%s' to namespace '%s'.\n", createdSecret.Name, createdSecret.Namespace)
	fmt.Println("You can verify with: kubectl get secret", createdSecret.Name, "-n", createdSecret.Namespace, "-o yaml")
	return nil
}
