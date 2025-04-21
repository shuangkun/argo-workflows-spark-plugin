package controller

import (
	"encoding/json"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"github.com/gin-gonic/gin"
	sparkjob "github.com/kubeflow/spark-operator/api/v1beta2"
	sparkversioned "github.com/kubeflow/spark-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"net/http"
	"time"
)

const (
	LabelKeyWorkflow string = "workflows.argoproj.io/workflow"
)

type SparkJobController struct {
	SparkClient *sparkversioned.Clientset
}

type SparkPluginBody struct {
	SparkJob *sparkjob.SparkApplication `json:"spark"`
}

func (ct *SparkJobController) ExecuteSparkJob(ctx *gin.Context) {
	c := &executorplugins.ExecuteTemplateArgs{}
	err := ctx.BindJSON(&c)
	if err != nil {
		klog.Error(err)
		return
	}

	inputBody := &SparkPluginBody{
		SparkJob: &sparkjob.SparkApplication{},
	}

	pluginJson, _ := c.Template.Plugin.MarshalJSON()
	klog.Info("Receive: ", string(pluginJson))
	err = json.Unmarshal(pluginJson, &inputBody)
	if err != nil {
		klog.Error(err)
		ct.Response404(ctx)
		return
	}

	job := inputBody.SparkJob

	if job.Name == "" {
		job.ObjectMeta.Name = c.Workflow.ObjectMeta.Name
	}

	if job.ObjectMeta.Namespace == "" {
		job.Namespace = "default"
	}

	var exists = false

	// 1. query job exists
	existsJob, err := ct.SparkClient.SparkoperatorV1beta2().SparkApplications(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
	if err != nil {
		exists = false
	} else {
		exists = true
	}
	// 2. found and return
	if exists {
		klog.Info("# found exists Spark Job: ", job.Name, "returning Status...", job.Status)
		ct.ResponseSparkJob(ctx, existsJob)
		return
	}

	// 3.Label keys with workflow Name
	InjectSparkJobWithWorkflowName(job, c.Workflow.ObjectMeta.Name)

	newJob, err := ct.SparkClient.SparkoperatorV1beta2().SparkApplications(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		klog.Error("### " + err.Error())
		ct.ResponseMsg(ctx, wfv1.NodeFailed, err.Error())
		return
	}

	ct.ResponseCreated(ctx, newJob)

}

func (ct *SparkJobController) ResponseCreated(ctx *gin.Context, job *sparkjob.SparkApplication) {
	message := ""
	if job.Status.AppState.ErrorMessage != "" {
		message = job.Status.AppState.ErrorMessage
	}
	ctx.JSON(http.StatusOK, &executorplugins.ExecuteTemplateReply{
		Node: &wfv1.NodeResult{
			Phase:   wfv1.NodePending,
			Message: message,
			Outputs: nil,
		},
		Requeue: &metav1.Duration{
			Duration: 10 * time.Second,
		},
	})
}

func (ct *SparkJobController) ResponseMsg(ctx *gin.Context, status wfv1.NodePhase, msg string) {
	ctx.JSON(http.StatusOK, &executorplugins.ExecuteTemplateReply{
		Node: &wfv1.NodeResult{
			Phase:   status,
			Message: msg,
			Outputs: nil,
		},
	})
}

func (ct *SparkJobController) ResponseSparkJob(ctx *gin.Context, job *sparkjob.SparkApplication) {
	jobPhase := &job.Status.AppState.State
	var status wfv1.NodePhase
	switch *jobPhase {
	case sparkjob.ApplicationStateRunning:
		status = wfv1.NodeRunning
	case sparkjob.ApplicationStateInvalidating:
		status = wfv1.NodeError
	case sparkjob.ApplicationStateCompleted:
		status = wfv1.NodeSucceeded
	case sparkjob.ApplicationStatePendingRerun:
		status = wfv1.NodePending
	case sparkjob.ApplicationStateFailed:
		status = wfv1.NodeFailed
	default:
		status = wfv1.NodeRunning
	}

	var requeue *metav1.Duration
	if status == wfv1.NodeRunning || status == wfv1.NodePending {
		requeue = &metav1.Duration{
			Duration: 10 * time.Second,
		}
	} else {
		requeue = nil
	}
	succeed := int32(0)
	total := *(job.Spec.Executor.Instances)
	if job.Status.ExecutorState != nil {
		for _, v := range job.Status.ExecutorState {
			if v == sparkjob.ExecutorStateCompleted {
				succeed++
			}
		}
	}
	progress, _ := wfv1.NewProgress(int64(succeed), int64(total))
	klog.Infof("### Job %v Phase "+", status: %v", job.Name, status)
	message := job.Status.AppState.ErrorMessage
	ctx.JSON(http.StatusOK, &executorplugins.ExecuteTemplateReply{
		Node: &wfv1.NodeResult{
			Phase:    status,
			Message:  message,
			Outputs:  nil,
			Progress: progress,
		},
		Requeue: requeue,
	})
}

func (ct *SparkJobController) Response404(ctx *gin.Context) {
	ctx.AbortWithStatus(http.StatusNotFound)
}

func InjectSparkJobWithWorkflowName(job *sparkjob.SparkApplication, workflowName string) {
	driverJob := job.Spec.Driver
	if driverJob.Labels != nil {
		driverJob.Labels[LabelKeyWorkflow] = workflowName
	} else {
		driverJob.Labels = map[string]string{
			LabelKeyWorkflow: workflowName,
		}
	}
	exectorJob := job.Spec.Executor
	if exectorJob.Labels != nil {
		exectorJob.Labels[LabelKeyWorkflow] = workflowName
	} else {
		exectorJob.Labels = map[string]string{
			LabelKeyWorkflow: workflowName,
		}
	}

	job.Spec.Driver = driverJob
	job.Spec.Executor = exectorJob
}
