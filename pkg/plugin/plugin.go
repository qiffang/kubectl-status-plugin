//go:generate statik -src templates/

package plugin

import (
	"bufio"
	"time"

	//"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	_ "github.com/qiffang/kubectl-status-plugin/pkg/plugin/statik"
	sfs "github.com/rakyll/statik/fs"
	"io"
	"io/ioutil"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"strings"

	//"k8s.io/cli-runtime/pkg/resource"
	"os"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var funcMap = template.FuncMap{
	"green":                 color.GreenString,
	"yellow":                color.YellowString,
	"red":                   color.RedString,
	"cyan":                  color.CyanString,
	"bold":                  color.New(color.Bold).SprintfFunc(),
	"colorAgo":              colorAgo,
	"colorDuration":         colorDuration,
	"colorBool":             colorBool,
	"colorKeyword":          colorKeyword,
	"colorExitCode":         colorExitCode,
	"markRed":               markRed,
	"markYellow":            markYellow,
	"markGreen":             markGreen,
	"redIf":                 redIf,
	"redBoldIf":             redBoldIf,
	"signalName":            signalName,
	"isPodConditionHealthy": isPodConditionHealthy,
	"quantityToFloat64":     quantityToFloat64,
	"quantityToInt64":       quantityToInt64,
	"percent":               percent,
	"humanizeSI":            humanizeSI,
	"getItemInList":         getItemInList,
}

func RunPlugin() error{
	//allNamespaces := "all-namespaces"
	KubernetesConfigFlags := genericclioptions.NewConfigFlags(false)
	f := cmdutil.NewFactory(KubernetesConfigFlags)
	clientSet, _ := f.KubernetesClientSet()
	//clientConfig := f.ToRawKubeConfigLoader()
	//namespace, enforceNamespace, err := clientConfig.Namespace()
	//
	//r := f.NewBuilder().
	//	Unstructured().
	//	AllNamespaces(true).
	//	//FilenameParam(enforceNamespace, &resource.FilenameOptions{Filenames: filenames}).
	//	//LabelSelectorParam(cmdutil.GetFlagString(cmd, "selector")).
	//	//FieldSelectorParam(cmdutil.GetFlagString(cmd, "field-selector")).
	//	ResourceTypeOrNameArgs(true).
	//	ContinueOnError().
	//	Latest().
	//	Flatten().
	//	Do()

	//err = r.Err()
	//if err != nil {
	//	return errors.WithMessage(err, "Failed during querying of resources")
	//}

	err := createDaemonsetIfAbsent(clientSet)
	if err != nil {
		color.Red("create checker failed, err=%v", err)
	}

	templateText, err := getTemplate()
	if err != nil {
		return err
	}

	var allErrs []error
	//infos, err := r.Infos()
	//if err != nil {
	//	allErrs = append(allErrs, err)
	//}
	//if len(infos) == 0 {
	//	fmt.Printf("No resources found.\n")
	//}

	pods, err := unhealthPods(clientSet)
	if err != nil {
		return err
	}

	for _, obj := range pods {
		if strings.HasPrefix(obj.Name, checkerName) {
			continue
		}

		err = render(templateText, obj, clientSet, f, "Pod")
		if err != nil {
			allErrs = append(allErrs, err)
			continue
		}
		// Add a newline at the end of every template
		fmt.Println("")
	}

	nodes, err := unhealthNodes(clientSet)
	if err != nil {
		return err
	}

	for _, obj := range nodes {
		err = render(templateText, obj, clientSet, f, "Node")
		if err != nil {
			allErrs = append(allErrs, err)
			continue
		}
		// Add a newline at the end of every template
		fmt.Println("")
	}

	//checkedHosts := make(map[string]bool)
	//nodeList, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	//if err != nil {
	//	allErrs = append(allErrs, err)
	//	return utilerrors.NewAggregate(allErrs)
	//}
	//
	//for _, node := range nodeList.Items {
	//	checkedHosts[node.Name] = false
	//}


	results, err := checkerNode(clientSet)
	if err != nil {
		allErrs = append(allErrs, err)
		return utilerrors.NewAggregate(allErrs)
	}

	fmt.Println("")

	successCheckedCount := 0
	for node, v := range results {
		nodeStatus := ""
		for _, str := range v.status {
			if str != "" {
				nodeStatus += str
			}
		}

		if v.inspected {
			successCheckedCount++
		}

		if nodeStatus != "" {
			color.Red("Node/%s need to check parameters: %s", node, nodeStatus)
		}

	}

	if successCheckedCount == 0 {
		color.Red("ApiServer/ControllerManager are not working properly, please check them.")
	}

	allErrs = append(allErrs, clientSet.AppsV1().DaemonSets(checkerNamespace).Delete(context.TODO(), checkerName, metav1.DeleteOptions{}))
	return utilerrors.NewAggregate(allErrs)
}

func checkerNode(clientSet *kubernetes.Clientset) (map[string]NodeStatus, error){
	checkedHosts := make(map[string]NodeStatus)
	nodeList, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodeList.Items {
		checkedHosts[node.Name] = NodeStatus{
			inspected: false,
			status: make(map[int]string),
		}
	}

	process := "..."
	for i := 0; i < 20; i++ {
		fmt.Print(process)
		process += process
		pods, _:= clientSet.CoreV1().Pods(checkerNamespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", checkLableKey, checkerLables[checkLableKey]),
		})
		if pods == nil {
			continue
		}

		for _, pod := range pods.Items {
			if pod.Spec.NodeName == "" {
				continue
			}
			req := clientSet.CoreV1().Pods(checkerNamespace).GetLogs(pod.Name, &v1.PodLogOptions{})
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				continue
			}
			defer podLogs.Close()

			scanner := bufio.NewScanner(podLogs)
			for scanner.Scan() {
				result := statusConvertToCode(scanner.Text())
				v := checkedHosts[pod.Spec.NodeName]
				v.inspected = true
				v.status[result] = StatusConvert[result]
				checkedHosts[pod.Spec.NodeName] = v

			}
		}

		skip := true
		for _, v := range checkedHosts {
			if !v.inspected {
				skip = false
			}
		}

		if skip {
			break
		}

		time.Sleep(time.Second * 5)
	}

	return checkedHosts, nil
}

type NodeStatus struct {
	inspected bool
	status map[int]string
}

const (
	InSpectSuccess = iota
	InSpectSelinuxEnabled
	InSpectSwapDisabled
	InSpectEnableBridgeNfCallIptables
)

var (
	StatusConvert = map[int]string {
		InSpectSuccess: "",
		InSpectSelinuxEnabled: "Need to check SeLinux status.",
		InSpectSwapDisabled: "Need to off swap.",
		InSpectEnableBridgeNfCallIptables: "Need to check bridge-nf-call-iptables parameter and enable it by echo '1' > /proc/sys/net/bridge/bridge-nf-call-iptables.",
	}
)

func statusConvertToCode(log string) int{
	r := InSpectSuccess
	if strings.Contains(log, "SelinuxEnabled:") {
		array := strings.Split(log, "SelinuxEnabled:")
		if !strings.Contains(array[1], "false") {
			r = InSpectSelinuxEnabled
		}
	}

	if strings.Contains(log, "SwapDisabled:") {
		array := strings.Split(log, "SwapDisabled:")
		if !strings.Contains(array[1], "true") {
			r = InSpectSwapDisabled
		}
	}

	if strings.Contains(log, "EnableBridgeNfCallIptables:") {
		array := strings.Split(log, "EnableBridgeNfCallIptables:")
		if !strings.Contains(array[1], "true") {
			r = InSpectEnableBridgeNfCallIptables
		}
	}

	return r
}

var (
	checkerName = "plugin-checker-fang"
	checkerNamespace = "default"
	checkLableKey = "plugin/checker-cluster"
	checkerLables = map[string]string{
		checkLableKey: "fang",
	}
)

func createDaemonsetIfAbsent(clientSet *kubernetes.Clientset) error{
	_, err := clientSet.AppsV1().DaemonSets(checkerNamespace).Get(context.TODO(), checkerName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return createDaemonset(clientSet, checkerName, checkerNamespace)
		}

		return err
	}

	err = clientSet.AppsV1().DaemonSets(checkerNamespace).Delete(context.TODO(), checkerName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return createDaemonset(clientSet, checkerName, checkerNamespace)
}

func createDaemonset(clientSet *kubernetes.Clientset, name string, namespace string) error{
	_, err := clientSet.AppsV1().DaemonSets(namespace).Create(context.TODO(), &v12.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v12.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: checkerLables,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: checkerLables,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "checker",
							Image: "qiffang199133/checker:v0.1",
							SecurityContext: &v1.SecurityContext{
								Privileged: Bool(true),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath: "/proc",
									Name: "proc",
								},

							},
						},
					},
					HostNetwork: true,
					HostPID: true,
					Volumes: []v1.Volume{
						{
							Name: "proc",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/proc",
								},
							},
						},
					},
					Tolerations: []v1.Toleration{
						{
						Operator: "Exists",
						Effect:   "NoSchedule",
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	return err
}

func Bool(v bool)*bool {
	return &v
}

func isPodConditionHealthy(condition map[string]interface{}) bool {
	switch {
	/*
		From https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties:

		> Condition types should indicate state in the "abnormal-true" polarity. For example, if the condition indicates
		> when a policy is invalid, the "is valid" case is probably the norm, so the condition should be called
		> "Invalid".

		But apparently this is not common among most resources, so we have the list of cases that matches the expected
		behaviour rather than the exceptions.
	*/
	case strings.HasSuffix(fmt.Sprint(condition["type"]), "Pressure"), // Node Pressure conditions
		strings.HasSuffix(fmt.Sprint(condition["type"]), "Unavailable"), // Node NetworkUnavailable condition
		strings.HasSuffix(fmt.Sprint(condition["type"]), "Failure"),     // ReplicaSet ReplicaFailure: condition
		strings.HasPrefix(fmt.Sprint(condition["type"]), "Non"),         // CRD NonStructuralSchema condition
		condition["type"] == "Failed":                                   // Failed Jobs has this condition
		switch condition["status"] {
		case "False":
			return true
		case "True", "Unknown":
			return false
		default:
			// not likely to ever happen, but just in case
			return false
		}
	default:
		switch condition["status"] {
		case "True":
			return true
		case "False", "Unknown":
			return false
		default:
			return false
		}
	}
}

func render(templateText string, obj runtime.Object, clientSet *kubernetes.Clientset, f cmdutil.Factory, objKind string) error{
	out := map[string]interface{}{}

	out["kind"] = objKind
	err := includeObj(obj, out)
	if err != nil {
		return err
	}
	err = includeEvents(obj, clientSet, out)
	if err != nil {
		return err
	}

	err = includePodMetrics(obj, f, out)
	if err != nil {
		return err
	}

	err = renderTemplate(templateText, os.Stdout, out)
	return err
}

func unhealthPods(clientSet *kubernetes.Clientset) ([]*v1.Pod, error){
	podList, err := clientSet.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("status.phase!=%s,status.phase!=%s", v1.PodRunning, v1.PodSucceeded),
	})
	if err != nil {
		return nil, err
	}

	pods := make([]*v1.Pod, 0)
	for _, pod := range podList.Items {
		pods = append(pods, &pod)
	}
	return pods, nil
}

func unhealthNodes(clientSet *kubernetes.Clientset) ([]*v1.Node, error){
	nodeList, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodes := make([]*v1.Node, 0)
	for _, node := range nodeList.Items {
		health := nodeHealth(node)

		if !health {
			nodes = append(nodes, &node)
		}
	}

	return nodes, nil
}

func nodeHealth(node v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Reason == "KubeletReady" {
			if condition.Status != v1.ConditionTrue {
				return false
			}
		}

		if condition.Type == v1.NodeMemoryPressure {
			if condition.Status != v1.ConditionFalse {
				return false
			}
		}

		if condition.Type == v1.NodeDiskPressure {
			if condition.Status != v1.ConditionFalse {
				return false
			}
		}

		if condition.Type == v1.NodePIDPressure {
			if condition.Status != v1.ConditionFalse {
				return false
			}
		}

		if condition.Type == v1.NodeNetworkUnavailable {
			if condition.Status != v1.ConditionFalse {
				return false
			}
		}

		if condition.Type == "KernelDeadlock" {
			if condition.Status != v1.ConditionFalse {
				return false
			}
		}

		if condition.Type == "ReadonlyFilesystem" {
			if condition.Status != v1.ConditionFalse {
				return false
			}
		}
	}

	return true
}



func includeEvents(obj runtime.Object, clientSet *kubernetes.Clientset, out map[string]interface{}) error {
	objectMeta := obj.(metav1.Object)
	events, err := clientSet.CoreV1().Events(objectMeta.GetNamespace()).Search(scheme.Scheme, obj)
	if err != nil {
		return errors.WithMessage(err, "Failed getting event")
	}
	eventsKey := make(map[string]interface{})
	err = unmarshal(events, &eventsKey)
	if err != nil {
		return errors.WithMessage(err, "Failed getting JSON for Events")
	}
	out["events"] = eventsKey
	return nil
}

func includeObj(obj runtime.Object, out map[string]interface{}) error {
	return unmarshal(obj, &out)
}

func unmarshal(v interface{}, out *map[string]interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, out)
	if err != nil {
		return err
	}
	return nil
}

func getItemInList(list []interface{}, itemKey, itemValue string) map[string]interface{} {
	var item map[string]interface{}
	for _, untypedItem := range list {
		typedItem := untypedItem.(map[string]interface{})
		if typedItem[itemKey].(string) == itemValue {
			item = typedItem
			break
		}
	}
	return item
}


//func renderFile(manifestFilename string) (string, error) {
//	var out map[string]interface{}
//	manifestFile, _ := ioutil.ReadFile(manifestFilename)
//	err := kyaml.Unmarshal(manifestFile, &out)
//	if err != nil {
//		return "", err
//	}
//	templateText, _ := getTemplate()
//	var output bytes.Buffer
//	err = renderTemplate(templateText, &output, out)
//	if err != nil {
//		return "", err
//	}
//	return output.String(), nil
//}

func getTemplate() (string, error) {
	statikFS, err := sfs.New()
	if err != nil {
		return "", err
	}

	// Access individual files by their paths.
	templatesFile := "/templates.tmpl"
	t, err := statikFS.Open(templatesFile)
	if err != nil {
		return "", err
	}
	defer t.Close()

	contents, err := ioutil.ReadAll(t)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func renderTemplate(templateText string, wr io.Writer, v map[string]interface{}) error {
	tmpl, err := template.
		New("templates.tmpl").
		Funcs(sprig.TxtFuncMap()).
		Funcs(funcMap).
		Parse(templateText)
	if err != nil {
		return err
	}
	kindTemplateName := findTemplateName(tmpl, v)
	return tmpl.ExecuteTemplate(wr, kindTemplateName, v)
}

func findTemplateName(tmpl *template.Template, v map[string]interface{}) string {
	objKind := v["kind"].(string)
	var kindTemplateName string
	if t := tmpl.Lookup(objKind); t != nil {
		kindTemplateName = objKind
	} else {
		kindTemplateName = "DefaultResource"
	}
	return kindTemplateName
}
