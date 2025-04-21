package main

import (
	"argo-workflows-spark-plugin/controller"
	"flag"
	"fmt"
	sparkversioned "github.com/kubeflow/spark-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

type option struct {
	port int
}

func main() {
	opt := &option{}
	cmd := &cobra.Command{
		Use:  "argo-spark-plugin",
		RunE: opt.runE,
	}
	flags := cmd.Flags()
	flags.IntVarP(&opt.port, "port", "", 3018, "The port of the HTTP server")
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func (o *option) runE(c *cobra.Command, args []string) (err error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			panic(err.Error())
		}
	}

	ct := &controller.SparkJobController{}
	sparkClient := getSparkClient(config)

	ct.SparkClient = sparkClient
	router := gin.Default()
	router.POST("/api/v1/template.execute", ct.ExecuteSparkJob)
	if err := router.Run(fmt.Sprintf(":%d", o.port)); err != nil {
		klog.Fatal("Failed to start server:", err)
	}
	return
}

// GetSparkClient get a clientset for Spark Job.
func getSparkClient(restConfig *rest.Config) *sparkversioned.Clientset {
	clientset, err := sparkversioned.NewForConfig(restConfig)
	klog.Info(clientset.SparkoperatorV1beta2())
	if err != nil {
		klog.Fatal(err)
	}
	return clientset
}
