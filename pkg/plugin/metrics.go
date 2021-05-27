package plugin

import (
	"context"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

func includePodMetrics(obj runtime.Object, f cmdutil.Factory, out map[string]interface{}) error {
	config, _ := f.ToRESTConfig()
	clientSet, err := metricsv.NewForConfig(config)
	if err != nil {
		return errors.WithMessage(err, "Failed getting metrics clientSet")
	}
	objectMeta := obj.(metav1.Object)
	podMetrics, err := clientSet.MetricsV1beta1().
		PodMetricses(objectMeta.GetNamespace()).
		Get(context.TODO(), objectMeta.GetName(), metav1.GetOptions{})
	if err != nil {
		// swallow any errors while getting PodMetrics
		return nil
	}
	podMetricsKey := make(map[string]interface{})
	err = unmarshal(podMetrics, &podMetricsKey)
	if err != nil {
		return errors.WithMessage(err, "Failed getting JSON for PodMetrics")
	}
	out["podMetrics"] = podMetricsKey
	return nil
}
