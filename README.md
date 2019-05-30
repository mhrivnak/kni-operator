# KNI Operator

**This is currently a Proof of Concept.**

The goal of this operator is to:

* Ensure a particular CatalogSource is available
* Ensure Subscriptions exist to one or more operators in that CatalogSource
* Ensure a CR exists for each operator so that it deploys its operand
* Watch the ClusterVersion and change the CatalogSource to reference an image that corresponds to the current ClusterVersion

The result enables a collection of operators to be released, installed and
upgraded together via a CatalogSource at carefully-chosen and tested versions.
A new CatalogSource gets created for each ClusterVersion, so that the operators
can be tested on specific releases of OpenShift.

## Try It

In this demo will you will:
* Deploy the operator
* See it deploy the etcd-operator from the associated CatalogSource
* Increment the ClusterVersion
* See the CatalogSource get changed based on the ClusterVersion, and then see the etcd-operator get upgraded.

This demo utilizes a [simple catalog source
image](https://quay.io/repository/mhrivnak/demo-operator-registry). It has two
branches: [`1.0`](https://github.com/mhrivnak/demo-operator-registry/tree/1.0)
and [`1.1`](https://github.com/mhrivnak/demo-operator-registry/tree/1.1). The
latter adds version 0.9.4 of the etcd operator.

All of the manifests in the catalog source image were copied straight from
[operatorhub.io](http://operatorhub.io).

### Setup

Start minikube.

```bash
minikube start --kubernetes-version v1.13.6
```

Install the Operator Lifecycle Manager. This command was copied straight from
[operatorhub.io](http://operatorhub.io). If you see errors, just run it a
second time.

```bash
kubectl create -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/olm.yaml
```

Create required CRDs and a ClusterVersion.

```bash
kubectl create -f deploy/crds/kni_v1alpha1_knicluster_crd.yaml
kubectl create --validate=false -f demo/0000_00_cluster-version-operator_01_clusterversion.crd.yaml
kubectl create -f demo/clusterversion.yaml
```

Start the kni-operator.

```bash
operator-sdk up local
```

Delete the Operator Hub CatalogSource just to keep it out of the way and keep things simple.

```bash
kubectl delete catalogsource operatorhubio-catalog -n olm
```

### Create KNICluster

Create a namespace to work with (it is currently hard-coded, so use this value).

```bash
kubectl create ns kniops
```

Create the KNICluster resource

```bash
kubectl create -f deploy/crds/kni_v1alpha1_knicluster_cr.yaml
```

### Results

You should see a CatalogSource and a Subscription.

```bash
$ kubectl get catalogsources -n olm
NAME            NAME            TYPE       PUBLISHER           AGE
demo-catalog    KNI Operators   grpc       kni.openshift.com   32s
olm-operators   OLM Operators   internal   Red Hat             6m
```

```bash
$ kubectl get subscriptions -n kniops
NAME   PACKAGE   SOURCE         CHANNEL
kni    etcd      demo-catalog   singlenamespace-alpha
```

You may need to wait a while for OLM to notice the CatalogSource, notice the Subscription, and act on them. Eventually you will
see an etcd ClusterServiceVersion:

```bash
$ kubectl get csvs --all-namespaces
NAMESPACE   NAME                    DISPLAY          VERSION   REPLACES              PHASE
kniops      etcdoperator.v0.9.2     etcd             0.9.2     etcdoperator.v0.9.0   Succeeded
olm         packageserver.v0.10.0   Package Server   0.10.0
```

You will also see a package manifest

```bash
$ kubectl get packagemanifests
NAME            CATALOG            AGE
packageserver   OLM Operators      9h
etcd            KNI Operators      9h
```

Finally you will see that the etcd operator was deployed.

```bash
$ kubectl get pods -n kniops
```

Once you see the etcd operator deployed, you can move on to the Upgrade section.

### Upgrade

Edit the ClusterVersion and change the version from "1.0" to "1.1".

```bash
$ kubectl edit clusterversion kni
```

You will then need to wait for OLM to see the change, but eventually the etcd
operator will be upgraded. You can look at the Subscription to see the update.

```
$ kubectl get subscription kni -n kniops -o yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  creationTimestamp: "2019-05-30T05:24:45Z"
  generation: 1
  name: kni
  namespace: kniops
  ownerReferences:
  - apiVersion: kni.openshift.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: KNICluster
    name: example-knicluster
    uid: 112b21f3-829b-11e9-aaed-583824d484b7
  resourceVersion: "66999"
  selfLink: /apis/operators.coreos.com/v1alpha1/namespaces/kniops/subscriptions/kni
  uid: 3c51385d-829b-11e9-aaed-583824d484b7
spec:
  channel: singlenamespace-alpha
  name: etcd
  source: demo-catalog
  sourceNamespace: olm
status:
  currentCSV: etcdoperator.v0.9.4
  installPlanRef:
    apiVersion: operators.coreos.com/v1alpha1
    kind: InstallPlan
    name: install-wbtqn
    namespace: kniops
    resourceVersion: "66990"
    uid: f8280c17-82d4-11e9-aaed-583824d484b7
  installedCSV: etcdoperator.v0.9.4
  installplan:
    apiVersion: operators.coreos.com/v1alpha1
    kind: InstallPlan
    name: install-wbtqn
    uuid: f8280c17-82d4-11e9-aaed-583824d484b7
  lastUpdated: "2019-05-30T12:18:04Z"
  state: AtLatestKnown
```
