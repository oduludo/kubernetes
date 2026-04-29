package cel

import (
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

const (
	None    string = ""
	podYAML string = `
apiVersion: v1
kind: Pod
metadata:
    creationTimestamp: "1998-10-21T18:39:43Z"
    generateName: foo-b6699dcfb-
    labels:
      app: nginx
      pod-template-hash: b6699dcfb
    name: foo-b6699dcfb-rnv7t
    namespace: default
    ownerReferences:
    - apiVersion: apps/v1
      blockOwnerDeletion: true
      controller: true
      kind: ReplicaSet
      name: foo-b6699dcfb
      uid: 8fc1088c-15d5-4a8c-8502-4dfcedef97b8
    resourceVersion: "14203463"
    uid: e2cc99fa-5a28-44da-b880-4dded28882ef
spec:
    containers:
    - image: nginx
      imagePullPolicy: IfNotPresent
      name: nginx
      ports:
      - containerPort: 80
      protocol: TCP
      resources:
        limits:
          cpu: 500m
          memory: 128Mi
        requests:
          cpu: 250m
          memory: 64Mi
      terminationMessagePath: /dev/termination-log
      terminationMessagePolicy: File
      volumeMounts:
      - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
        name: kube-api-access-s64k4
        readOnly: true
    dnsPolicy: ClusterFirst
    enableServiceLinks: true
    nodeName: knode0
    preemptionPolicy: PreemptLowerPriority
    priority: 0
    restartPolicy: Always
    schedulerName: default-scheduler
    securityContext: {}
    serviceAccount: default
    serviceAccountName: default
    terminationGracePeriodSeconds: 30
    tolerations:
    - effect: NoExecute
      key: node.kubernetes.io/not-ready
      operator: Exists
      tolerationSeconds: 300
    - effect: NoExecute
      key: node.kubernetes.io/unreachable
      operator: Exists
      tolerationSeconds: 300
    volumes:
    - name: kube-api-access-s64k4
      projected:
        defaultMode: 420
        sources:
        - serviceAccountToken:
            expirationSeconds: 3607
            path: token
        - configMap:
            items:
            - key: ca.crt
              path: ca.crt
            name: kube-root-ca.crt
status:
    conditions:
    - lastProbeTime: null
      lastTransitionTime: "1998-10-21T18:39:37Z"
      status: "True"
      type: Initialized
    - lastProbeTime: null
      lastTransitionTime: "1998-10-21T18:39:42Z"
      status: "True"
      type: Ready
    - lastProbeTime: null
      lastTransitionTime: "1998-10-21T18:39:42Z"
      status: "True"
      type: ContainersReady
    - lastProbeTime: null
      lastTransitionTime: "1998-10-21T18:39:37Z"
      status: "True"
      type: PodScheduled
    containerStatuses:
    - containerID: containerd://e35792ba1d6e9a56629659b35dbdb93dacaa0a413962ee04775319f5438e493c
      image: docker.io/library/nginx:latest
      imageID: docker.io/library/nginx@sha256:644a70516a26004c97d0d85c7fe1d0c3a67ea8ab7ddf4aff193d9f301670cf36
      lastState: {}
      name: nginx
      ready: true
      restartCount: 0
      started: true
      state:
        running:
          startedAt: "1998-10-21T18:39:41Z"
    hostIP: 192.168.0.22
    phase: Running
    podIP: 10.42.1.203
    podIPs:
    - ip: 10.42.1.203
    qosClass: Burstable
    startTime: "1998-10-21T18:39:37Z"
`
)

// createUnstructured parses the yaml string into a map[string]interface{}.  Verifies that the string does not have
// any tab characters.
func createUnstructured(t *testing.T, config string) *unstructured.Unstructured {
	t.Helper()
	result := map[string]interface{}{}

	require.NotContains(t, config, "\t", "Yaml %s cannot contain tabs", config)
	require.NoError(t, yaml.Unmarshal([]byte(config), &result), "Could not parse config:\n\n%s\n", config)

	return &unstructured.Unstructured{
		Object: result,
	}
}

func TestLib(t *testing.T) {
	// Next steps:
	// - Create a mock variant of genericclioptions.RESTClientGetter to control the returns Pod(s)

	scheme := runtime.NewScheme()

	u := createUnstructured(t, podYAML)
	client := fakedynamic.NewSimpleDynamicClient(scheme, u)

	val, err := exec(client, "pods == pods", types.BoolType)

	val, err := runCelCommandOnUnstructuredMapping(u, cel.BoolType, "job", test.command)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val.Value() != test.expected {
		t.Fatalf("unexpected value: %v", val.Value())
	}
}
