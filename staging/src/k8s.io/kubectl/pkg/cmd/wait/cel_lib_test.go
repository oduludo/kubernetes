package wait

import (
	"testing"

	"github.com/google/cel-go/cel"
)

const (
	jobYamlActive = `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"batch/v1","kind":"Job","metadata":{"annotations":{},"name":"worker1","namespace":"default"},"spec":{"completions":6,"template":{"spec":{"containers":[{"command":["sh","-c","echo Hello, Kubernetes! \u0026\u0026 sleep 10"],"image":"busybox","name":"worker"}],"restartPolicy":"Never"}}}}
  creationTimestamp: "2026-04-26T18:35:42Z"
  generation: 1
  labels:
    batch.kubernetes.io/controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
    batch.kubernetes.io/job-name: worker1
    controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
    job-name: worker1
  name: worker1
  namespace: default
  resourceVersion: "27446"
  uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 6
  manualSelector: false
  parallelism: 1
  podReplacementPolicy: TerminatingOrFailed
  selector:
    matchLabels:
      batch.kubernetes.io/controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
  suspend: false
  template:
    metadata:
      labels:
        batch.kubernetes.io/controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
        batch.kubernetes.io/job-name: worker1
        controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
        job-name: worker1
    spec:
      containers:
      - command:
        - sh
        - -c
        - echo Hello, Kubernetes! && sleep 10
        image: busybox
        imagePullPolicy: Always
        name: worker
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  active: 1
  ready: 0
  startTime: "2026-04-26T18:35:42Z"
  succeeded: 1
  terminating: 0
  uncountedTerminatedPods: {}`
	jobYamlCompletedSuccessfully = `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"batch/v1","kind":"Job","metadata":{"annotations":{},"name":"worker1","namespace":"default"},"spec":{"completions":6,"template":{"spec":{"containers":[{"command":["sh","-c","echo Hello, Kubernetes! \u0026\u0026 sleep 10"],"image":"busybox","name":"worker"}],"restartPolicy":"Never"}}}}
  creationTimestamp: "2026-04-26T18:35:42Z"
  generation: 1
  labels:
    batch.kubernetes.io/controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
    batch.kubernetes.io/job-name: worker1
    controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
    job-name: worker1
  name: worker1
  namespace: default
  resourceVersion: "27656"
  uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 6
  manualSelector: false
  parallelism: 1
  podReplacementPolicy: TerminatingOrFailed
  selector:
    matchLabels:
      batch.kubernetes.io/controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
  suspend: false
  template:
    metadata:
      labels:
        batch.kubernetes.io/controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
        batch.kubernetes.io/job-name: worker1
        controller-uid: 4dc6ef71-c2eb-44af-9b73-bb77ab6463db
        job-name: worker1
    spec:
      containers:
      - command:
        - sh
        - -c
        - echo Hello, Kubernetes! && sleep 10
        image: busybox
        imagePullPolicy: Always
        name: worker
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  completionTime: "2026-04-26T18:37:06Z"
  conditions:
  - lastProbeTime: "2026-04-26T18:37:06Z"
    lastTransitionTime: "2026-04-26T18:37:06Z"
    message: Reached expected number of succeeded pods
    reason: CompletionsReached
    status: "True"
    type: SuccessCriteriaMet
  - lastProbeTime: "2026-04-26T18:37:06Z"
    lastTransitionTime: "2026-04-26T18:37:06Z"
    message: Reached expected number of succeeded pods
    reason: CompletionsReached
    status: "True"
    type: Complete
  ready: 0
  startTime: "2026-04-26T18:35:42Z"
  succeeded: 6
  terminating: 0
  uncountedTerminatedPods: {}
`
	jobYamlCompletedFailed = `
apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"batch/v1","kind":"Job","metadata":{"annotations":{},"name":"worker2","namespace":"default"},"spec":{"completions":6,"template":{"spec":{"containers":[{"command":["sh","-c","echo Hello, Kubernetes! \u0026\u0026 sleep 10 \u0026\u0026 exit 1"],"image":"busybox","name":"worker"}],"restartPolicy":"Never"}}}}
  creationTimestamp: "2026-04-26T18:35:42Z"
  generation: 1
  labels:
    batch.kubernetes.io/controller-uid: a4bb1d1c-2471-457d-a31f-3a8371630c94
    batch.kubernetes.io/job-name: worker2
    controller-uid: a4bb1d1c-2471-457d-a31f-3a8371630c94
    job-name: worker2
  name: worker2
  namespace: default
  resourceVersion: "28542"
  uid: a4bb1d1c-2471-457d-a31f-3a8371630c94
spec:
  backoffLimit: 6
  completionMode: NonIndexed
  completions: 6
  manualSelector: false
  parallelism: 1
  podReplacementPolicy: TerminatingOrFailed
  selector:
    matchLabels:
      batch.kubernetes.io/controller-uid: a4bb1d1c-2471-457d-a31f-3a8371630c94
  suspend: false
  template:
    metadata:
      labels:
        batch.kubernetes.io/controller-uid: a4bb1d1c-2471-457d-a31f-3a8371630c94
        batch.kubernetes.io/job-name: worker2
        controller-uid: a4bb1d1c-2471-457d-a31f-3a8371630c94
        job-name: worker2
    spec:
      containers:
      - command:
        - sh
        - -c
        - echo Hello, Kubernetes! && sleep 10 && exit 1
        image: busybox
        imagePullPolicy: Always
        name: worker
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Never
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  conditions:
  - lastProbeTime: "2026-04-26T18:47:32Z"
    lastTransitionTime: "2026-04-26T18:47:32Z"
    message: Job has reached the specified backoff limit
    reason: BackoffLimitExceeded
    status: "True"
    type: FailureTarget
  - lastProbeTime: "2026-04-26T18:47:32Z"
    lastTransitionTime: "2026-04-26T18:47:32Z"
    message: Job has reached the specified backoff limit
    reason: BackoffLimitExceeded
    status: "True"
    type: Failed
  failed: 7
  ready: 0
  startTime: "2026-04-26T18:35:42Z"
  terminating: 0
  uncountedTerminatedPods: {}`
)

func TestCel_Job_BooleanMethods(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
		yaml     string
	}{
		// hasFinished
		{
			name:     "job has not finished",
			command:  "job.hasFinished()",
			expected: false,
			yaml:     jobYamlActive,
		},
		{
			name:     "job has finished success",
			command:  "job.hasFinished()",
			expected: true,
			yaml:     jobYamlCompletedSuccessfully,
		},
		{
			name:     "job has finished failure",
			command:  "job.hasFinished()",
			expected: true,
			yaml:     jobYamlCompletedFailed,
		},
		// hasSucceeded
		{
			name:     "active job has not succeeded",
			command:  "job.hasSucceeded()",
			expected: false,
			yaml:     jobYamlActive,
		},
		{
			name:     "successful job has succeeded",
			command:  "job.hasSucceeded()",
			expected: true,
			yaml:     jobYamlCompletedSuccessfully,
		},
		{
			name:     "failed job has not succeeded",
			command:  "job.hasSucceeded()",
			expected: false,
			yaml:     jobYamlCompletedFailed,
		},
		// hasFailed
		{
			name:     "active job has not failed",
			command:  "job.hasFailed()",
			expected: false,
			yaml:     jobYamlActive,
		},
		{
			name:     "successful job has not failed",
			command:  "job.hasFailed()",
			expected: false,
			yaml:     jobYamlCompletedSuccessfully,
		},
		{
			name:     "failed job has failed",
			command:  "job.hasFailed()",
			expected: true,
			yaml:     jobYamlCompletedFailed,
		},
		// hasSucceeded + hasFailed
		{
			name:     "active job has not finished (split calls)",
			command:  "job.hasSucceeded() || job.hasFailed()",
			expected: false,
			yaml:     jobYamlActive,
		},
		{
			name:     "successful job has finished (split calls)",
			command:  "job.hasSucceeded() || job.hasFailed()",
			expected: true,
			yaml:     jobYamlCompletedSuccessfully,
		},
		{
			name:     "failed job has finished (split calls)",
			command:  "job.hasSucceeded() || job.hasFailed()",
			expected: true,
			yaml:     jobYamlCompletedFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := createUnstructured(t, test.yaml)
			val, err := runCelCommandOnUnstructuredMapping(u, cel.BoolType, "job", test.command)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if val.Value() != test.expected {
				t.Fatalf("unexpected value: %v", val.Value())
			}
		})
	}
}
