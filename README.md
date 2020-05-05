# operator-example

This is a test project used to deal with custom controller in Kubernetes. 
In order to create an Operator it has been used the Kubebuilder framework.

## Overview

For test purpose operator-example is made of two Kind(CRD), as Kubebuilder calls them, and the corresponding controllers.
The first Kind is called "FireWall" and the second "LoadBalancer". The purpose of the FireWall is to watch the 
Deployment which name is specified in the SPEC field DeploymentName and updates the STATUS field DeploymentReplicas. 
At the same time the FireWall updates the SPEC field DeploymentReplicas of the LoadBalancer, who compares the number of 
replicas with a threshold and updates the STATUS field AboveThreshold with true or false.

In config/sample folder there are 2 example of CR for both Kind, but also there are example of Deployment, HPA and Service that 
can be used to try the custom controller. In this way, when the Autoscaler will increase or decrease (based on CPU utilization)
the number of the Pod, the reconcile method of FirstPlayer will be triggered and the number of replicas will be updated.

## Simulate load

To simulate the load on pod and force the Autoscaler to increase the number of replicas it's possible to use the python script 
in utils directory and run it with [locust](https://locust.io/).
First expose the service with:

```bash
kubectl port-forward svc/nginx-service 8080:80
```

Then start the locust server locally with 

```bash
locust -f locust-get-test.py
```

and use the web interface exposed on [http://localhost:8089/](http://localhost:8089/) to manage the simulation.
