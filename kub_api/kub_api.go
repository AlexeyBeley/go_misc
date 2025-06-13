package kub_api

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Kubeconfig *string
	clientset  *kubernetes.Clientset
	Namespace  *string
}

type Job struct {
	JobName                 *string
	ContainerName           *string
	ContainerImage          *string
	ContainerCommand        *[]string
	TTLSecondsAfterFinished *int32
	UID                     *types.UID
}

func (job *Job) GenerateBatchJob() (ret *batchv1.Job, err error) {
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
		fmt.Printf("Error building kubeconfig: %v\n", err)
		//config, err = clientcmd.InClusterConfig()
		if err != nil {
			fmt.Printf("Error building in-cluster config: %v\n", err)
			os.Exit(1)
		}
	}

	// Create a Kubernetes kapi.clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating kapi.clientset: %v\n", err)
		os.Exit(1)
	}
	ret.clientset = clientset

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

func (kapi *KubAPI) CreateJob(job *Job) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}
	batchJob, err := job.GenerateBatchJob()
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

func (kapi *KubAPI) DeleteJob(job *Job) error {
	namespace, err := kapi.GetActiveNamespace()
	if err != nil {
		return err
	}
	batchJob, err := job.GenerateBatchJob()
	batchJob.ObjectMeta.Namespace = *namespace

	err = kapi.clientset.BatchV1().Jobs(*kapi.Namespace).Delete(context.TODO(), *job.JobName, metav1.DeleteOptions{})

	if err != nil {
		fmt.Printf("Error listing namespaces: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Job deleted successfully! Name: %s, Namespace: %s\n", *job.JobName, *kapi.Namespace)
	return nil
}

func (kapi *KubAPI) CreatePod(job *Job, podID string) error {
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

func (kapi *KubAPI) Getbatchv1Job(job *Job) (*batchv1.Job, error) {
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

func (kapi *KubAPI) CreateService(serviceName *string, port int32, selector map[string]string) error {
	// Create the Service
	// Define the Service object
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *serviceName,
			Namespace: *kapi.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     port,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP, // Use a ClusterIP for internal access
		},
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

func (kapi *KubAPI) CreateServiceAccount(serviceAccount *corev1.ServiceAccount) error {
	// List Services in the specified namespace

	_, err := kapi.clientset.CoreV1().ServiceAccounts(*kapi.Namespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Error creating Service Account: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Service Account created successfully")

	return nil
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
