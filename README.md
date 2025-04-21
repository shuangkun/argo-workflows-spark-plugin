# argo workflows spark plugin


A plugin lets Argo Workflows orchestrate Spark jobs.


## Why argo-workflows-spark-plugin

* Submit tasks using non-string methods. More flexibly control and observe the status of spark jobs.

* Save costs. In scenarios where a large number of Spark jobs are orchestrated, there is no need to generate an equal number of resource pods.

## Getting Started

1. Enable Plugin capability for controller
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  template:
    spec:
      containers:
        - name: workflow-controller
          env:
            - name: ARGO_EXECUTOR_PLUGINS
              value: "true"
```
2. Build argo-spark-plugin image

```
git clone https://github.com/shuangkun/argo-workflows-spark-plugin.git
cd argo-workflows-spark-plugin
docker build -t argo-spark-plugin:v1 .
```
3. Deploy argo-spark-plugin
```
kubectl apply -f spark-executor-plugin-configmap.yaml
```

4. Permission to create SparkJob CRD

```
kubctl apply -f install/role-secret.yaml
```

4. Submit Spark jobs
```
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: spark-pi-demo-
spec:
  entrypoint: spark-pi
  templates:
    - name: spark-pi
      plugin:
        spark:
          # SparkApplication definition (Spark Operator must be installed in advance)
          apiVersion: "sparkoperator.k8s.io/v1beta2"
          kind: SparkApplication
          metadata:
            name: spark-pi-demo
            namespace: argo
          spec:
            type: Scala
            mode: cluster
            image: "gcr.io/spark-operator/spark:v3.3.1"
            mainClass: org.apache.spark.examples.SparkPi
            mainApplicationFile: "local:///opt/spark/examples/jars/spark-examples_2.12-3.3.1.jar"
            sparkVersion: "3.3.1"
            restartPolicy:
              type: Never
            driver:
              cores: 1
              memory: "2g"
              serviceAccount: spark-sa
              labels:
                version: 3.3.1
            executor:
              cores: 2
              instances: 2
              memory: "4g"
              labels:
                version: 3.3.1
            arguments:
              - "1000" 
```