package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestWorkerDeploymentTemplate(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedErrorRegexp *regexp.Regexp

		ExpectedName        string
		ExpectedRelease     string
		ExpectedDeployments []workerDeploymentTestCase
	}{
		{
			CaseName: "happy",
			Release:  "production",
			Values: map[string]string{
				"releaseOverride":            "productionOverridden",
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
				"workers.worker2.command[0]": "echo",
				"workers.worker2.command[1]": "worker2",
			},
			ExpectedName:    "productionOverridden",
			ExpectedRelease: "production",
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:         "productionOverridden-worker1",
					ExpectedCmd:          []string{"echo", "worker1"},
					ExpectedStrategyType: appsV1.DeploymentStrategyType(""),
				},
				{
					ExpectedName:         "productionOverridden-worker2",
					ExpectedCmd:          []string{"echo", "worker2"},
					ExpectedStrategyType: appsV1.DeploymentStrategyType(""),
				},
			},
		}, {
			// See https://github.com/helm/helm/issues/6006
			CaseName: "long release name",
			Release:  strings.Repeat("r", 80),

			ExpectedErrorRegexp: regexp.MustCompile("Error: release name .* length must not be longer than 53"),
		},
		{
			CaseName: "strategyType",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":   "echo",
				"workers.worker1.command[1]":   "worker1",
				"workers.worker1.strategyType": "Recreate",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:         "production" + "-worker1",
					ExpectedCmd:          []string{"echo", "worker1"},
					ExpectedStrategyType: appsV1.RecreateDeploymentStrategyType,
				},
			},
		},
		{
			CaseName: "nodeSelector",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":            "echo",
				"workers.worker1.command[1]":            "worker1",
				"workers.worker1.nodeSelector.disktype": "ssd",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:         "production" + "-worker1",
					ExpectedCmd:          []string{"echo", "worker1"},
					ExpectedStrategyType: appsV1.DeploymentStrategyType(""),
					ExpectedNodeSelector: map[string]string{"disktype": "ssd"},
				},
			},
		},
		{
			CaseName: "tolerations",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":              "echo",
				"workers.worker1.command[1]":              "worker1",
				"workers.worker1.tolerations[0].key":      "key1",
				"workers.worker1.tolerations[0].operator": "Equal",
				"workers.worker1.tolerations[0].value":    "value1",
				"workers.worker1.tolerations[0].effect":   "NoSchedule",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:         "production" + "-worker1",
					ExpectedCmd:          []string{"echo", "worker1"},
					ExpectedStrategyType: appsV1.DeploymentStrategyType(""),
					ExpectedTolerations: []coreV1.Toleration{
						{
							Key:      "key1",
							Operator: "Equal",
							Value:    "value1",
							Effect:   "NoSchedule",
						},
					},
				},
			},
		},
		{
			CaseName: "initContainers",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":                   "echo",
				"workers.worker1.command[1]":                   "worker1",
				"workers.worker1.initContainers[0].name":       "myservice",
				"workers.worker1.initContainers[0].image":      "myimage:1",
				"workers.worker1.initContainers[0].command[0]": "sh",
				"workers.worker1.initContainers[0].command[1]": "-c",
				"workers.worker1.initContainers[0].command[2]": "until nslookup myservice; do echo waiting for myservice to start; sleep 1; done;",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:         "production" + "-worker1",
					ExpectedCmd:          []string{"echo", "worker1"},
					ExpectedStrategyType: appsV1.DeploymentStrategyType(""),
					ExpectedInitContainers: []coreV1.Container{
						{
							Name:    "myservice",
							Image:   "myimage:1",
							Command: []string{"sh", "-c", "until nslookup myservice; do echo waiting for myservice to start; sleep 1; done;"},
						},
					},
				},
			},
		},
		{
			CaseName: "affinity",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
				"workers.worker1.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key":      "key1",
				"workers.worker1.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator": "DoesNotExist",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:         "production" + "-worker1",
					ExpectedCmd:          []string{"echo", "worker1"},
					ExpectedStrategyType: appsV1.DeploymentStrategyType(""),
					ExpectedAffinity: &coreV1.Affinity{
						NodeAffinity: &coreV1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &coreV1.NodeSelector{
								NodeSelectorTerms: []coreV1.NodeSelectorTerm{
									{
										MatchExpressions: []coreV1.NodeSelectorRequirement{
											{
												Key:      "key1",
												Operator: "DoesNotExist",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app": "auto-devops-examples/minimal-ruby-app",
				"gitlab.env": "prod",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, tc.ExpectedErrorRegexp)

			if tc.ExpectedErrorRegexp != nil {
				return
            }

			var deployments deploymentList
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))
			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]

				require.Equal(t, expectedDeployment.ExpectedName, deployment.Name)
				require.Equal(t, expectedDeployment.ExpectedStrategyType, deployment.Spec.Strategy.Type)

				require.Equal(t, map[string]string{
					"app.gitlab.com/app": "auto-devops-examples/minimal-ruby-app",
					"app.gitlab.com/env": "prod",
				}, deployment.Annotations)
				require.Equal(t, map[string]string{
					"chart":    chartName,
					"heritage": "Helm",
					"release":  tc.ExpectedRelease,
					"tier":     "worker",
					"track":    "stable",
				}, deployment.Labels)

				require.Equal(t, map[string]string{
					"app.gitlab.com/app":           "auto-devops-examples/minimal-ruby-app",
					"app.gitlab.com/env":           "prod",
					"checksum/application-secrets": "",
				}, deployment.Spec.Template.Annotations)
				require.Equal(t, map[string]string{
					"release": tc.ExpectedRelease,
					"tier":    "worker",
					"track":   "stable",
				}, deployment.Spec.Template.Labels)

				require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
				require.Equal(t, expectedDeployment.ExpectedCmd, deployment.Spec.Template.Spec.Containers[0].Command)

				require.Equal(t, expectedDeployment.ExpectedNodeSelector, deployment.Spec.Template.Spec.NodeSelector)
				require.Equal(t, expectedDeployment.ExpectedTolerations, deployment.Spec.Template.Spec.Tolerations)
				require.Equal(t, expectedDeployment.ExpectedInitContainers, deployment.Spec.Template.Spec.InitContainers)
				require.Equal(t, expectedDeployment.ExpectedAffinity, deployment.Spec.Template.Spec.Affinity)
			}
		})
	}

	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedImagePullPolicy coreV1.PullPolicy
		ExpectedImageRepository string
	}{
		{
			CaseName: "worker image is defined",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.image.repository": "worker1/image/repo",
				"workers.worker1.image.tag":        "worker1-tag",
			},
			ExpectedImageRepository: string("worker1/image/repo:worker1-tag"),
		},
		{
			CaseName: "worker image pullPolicy is defined",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.image.repository": "worker1/image/repo",
				"workers.worker1.image.tag":        "worker1-tag",
				"workers.worker1.image.pullPolicy": "Always",
			},
			ExpectedImagePullPolicy: coreV1.PullAlways,
			ExpectedImageRepository: string("worker1/image/repo:worker1-tag"),
		},
		{
			CaseName: "root image is defined",
			Release:  "production",
			Values: map[string]string{
				"image.repository": "root/image/repo",
				"image.tag":        "root-tag",
			},
			ExpectedImagePullPolicy: coreV1.PullIfNotPresent,
			ExpectedImageRepository: string("root/image/repo:root-tag"),
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app":                 "auto-devops-examples/minimal-ruby-app",
				"gitlab.env":                 "prod",
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentList
			helm.UnmarshalK8SYaml(t, output, &deployments)
			for i := range deployments.Items {
				deployment := deployments.Items[i]
				require.Equal(
					t,
					tc.ExpectedImageRepository,
					deployment.Spec.Template.Spec.Containers[0].Image,
				)
				require.Equal(
					t,
					tc.ExpectedImagePullPolicy,
					deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy,
				)
			}
		})
	}

	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedImagePullSecrets []coreV1.LocalObjectReference
	}{
		{
			CaseName: "global image secrets default",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
			},
			ExpectedImagePullSecrets: []coreV1.LocalObjectReference{
				{
					Name: "gitlab-registry",
				},
			},
		},
		{
			CaseName: "worker image secrets are defined",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.image.secrets[0].name": "expected-secret",
			},
			ExpectedImagePullSecrets: []coreV1.LocalObjectReference{
				{
					Name: "expected-secret",
				},
			},
		},
		{
			CaseName: "global image secrets are defined",
			Release:  "production",
			Values: map[string]string{
				"image.secrets[0].name": "expected-secret",
				"workers.worker1.command[0]": "echo",
			},
			ExpectedImagePullSecrets: []coreV1.LocalObjectReference{
				{
					Name: "expected-secret",
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentList

			helm.UnmarshalK8SYaml(t, output, &deployments)
			for i := range deployments.Items {
				deployment := deployments.Items[i]
				require.Equal(
					t,
					tc.ExpectedImagePullSecrets,
					deployment.Spec.Template.Spec.ImagePullSecrets,
				)
			}
		})
	}

	// podAnnotations & labels
	for _, tc := range []struct {
		CaseName                   string
		Values                     map[string]string
		Release 				   string
		ExpectedPodAnnotations     map[string]string
		ExpectedPodLabels          map[string]string
	}{
		{
			CaseName: "one podAnnotations",
			Release:  "production",
			Values: map[string]string{
				"podAnnotations.firstAnnotation":    "expected-annotation",
				"workers.worker1.command[0]": "echo",
			},
			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
				"firstAnnotation":              "expected-annotation",
			},
			ExpectedPodLabels: map[string]string{
				"release":    "production",
				"tier":       "worker",
				"track":      "stable",
			},
		},
		{
			CaseName: "multiple podAnnotations",
			Release:  "production",
			Values: map[string]string{
				"podAnnotations.firstAnnotation":    "expected-annotation",
				"podAnnotations.secondAnnotation":   "expected-annotation",
				"workers.worker1.command[0]": "echo",
			},
			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
				"firstAnnotation":              "expected-annotation",
				"secondAnnotation":             "expected-annotation",
			},
			ExpectedPodLabels: nil,
		},
		{
			CaseName: "one label",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.labels.firstLabel":    "expected-label",
				"workers.worker1.command[0]": "echo",
			},
			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
			},
			ExpectedPodLabels: map[string]string{
				"firstLabel": "expected-label",
			},
		},
		{
			CaseName: "multiple labels",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.labels.firstLabel":    "expected-label",
				"workers.worker1.labels.secondLabel":    "expected-label",
				"workers.worker1.command[0]": "echo",
			},
			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
			},
			ExpectedPodLabels: map[string]string{
				"firstLabel": "expected-label",
				"secondLabel": "expected-label",
			},
		},
		{
			CaseName: "no podAnnotations & labels",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
			},
			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
			},
			ExpectedPodLabels: nil,
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentList

			helm.UnmarshalK8SYaml(t, output, &deployments)
			for i := range deployments.Items {
				deployment := deployments.Items[i]
				require.Equal(t, tc.ExpectedPodAnnotations, deployment.Spec.Template.ObjectMeta.Annotations)
				for key, value := range tc.ExpectedPodLabels {
					require.Equal(t, deployment.Spec.Template.ObjectMeta.Labels[key], value)
				}
			}
		})
	}

	// hostAliases
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedHostAliases []coreV1.HostAlias
	}{
		{
			CaseName: "hostAliases for two IP addresses",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.hostAliases[0].ip":           "1.2.3.4",
				"workers.worker1.hostAliases[0].hostnames[0]": "host1.example1.com",
				"workers.worker1.hostAliases[1].ip":           "5.6.7.8",
				"workers.worker1.hostAliases[1].hostnames[0]": "host1.example2.com",
				"workers.worker1.hostAliases[1].hostnames[1]": "host2.example2.com",
			},

			ExpectedHostAliases: []coreV1.HostAlias{
				{
					IP:        "1.2.3.4",
					Hostnames: []string{"host1.example1.com"},
				},
				{
					IP:        "5.6.7.8",
					Hostnames: []string{"host1.example2.com", "host2.example2.com"},
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentList

			helm.UnmarshalK8SYaml(t, output, &deployments)
			for i := range deployments.Items {
				deployment := deployments.Items[i]
				require.Equal(t, tc.ExpectedHostAliases, deployment.Spec.Template.Spec.HostAliases)
			}
		})
	}

	// dnsConfig
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedDnsConfig *coreV1.PodDNSConfig
	}{
		{
			CaseName: "dnsConfig with different DNS",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.dnsConfig.nameservers[0]":  "1.2.3.4",
				"workers.worker1.dnsConfig.options[0].name": "edns0",
			},

			ExpectedDnsConfig: &coreV1.PodDNSConfig{
				Nameservers: []string{"1.2.3.4"},
				Options:     []coreV1.PodDNSConfigOption{
					{
						Name: "edns0",
					},
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentList

			helm.UnmarshalK8SYaml(t, output, &deployments)
			for i := range deployments.Items {
				deployment := deployments.Items[i]
				require.Equal(t, tc.ExpectedDnsConfig, deployment.Spec.Template.Spec.DNSConfig)
			}
		})
	}

	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedDeployments []workerDeploymentHostNetworkTestCase
	}{
		{
			CaseName: "worker hostNetwork is defined",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.hostNetwork": "true",
			},
			ExpectedDeployments: []workerDeploymentHostNetworkTestCase{
				{
					ExpectedHostNetwork: bool(true),
				},
			},
		},
		{
			CaseName: "root hostNetwork is defined",
			Release:  "production",
			Values: map[string]string{
				"hostNetwork": "true",
			},
			ExpectedDeployments: []workerDeploymentHostNetworkTestCase{
				{
					ExpectedHostNetwork: bool(true),
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app":                 "auto-devops-examples/minimal-ruby-app",
				"gitlab.env":                 "prod",
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))

			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]
				require.Equal(
					t,
					expectedDeployment.ExpectedHostNetwork,
					deployment.Spec.Template.Spec.HostNetwork,
				)
			}
		})
	}

	// Tests worker selector
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedName        string
		ExpectedRelease     string
		ExpectedDeployments []workerDeploymentSelectorTestCase
	}{
		{
			CaseName: "worker selector",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
				"workers.worker2.command[0]": "echo",
				"workers.worker2.command[1]": "worker2",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedDeployments: []workerDeploymentSelectorTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"release": "production",
							"tier":    "worker",
							"track":   "stable",
						},
					},
				},
				{
					ExpectedName: "production-worker2",
					ExpectedSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"release": "production",
							"tier":    "worker",
							"track":   "stable",
						},
					},
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app": "auto-devops-examples/minimal-ruby-app",
				"gitlab.env": "prod",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))
			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]

				require.Equal(t, expectedDeployment.ExpectedName, deployment.Name)

				require.Equal(t, map[string]string{
					"chart":    chartName,
					"heritage": "Helm",
					"release":  tc.ExpectedRelease,
					"tier":     "worker",
					"track":    "stable",
				}, deployment.Labels)

				require.Equal(t, expectedDeployment.ExpectedSelector, deployment.Spec.Selector)

				require.Equal(t, map[string]string{
					"release": tc.ExpectedRelease,
					"tier":    "worker",
					"track":   "stable",
				}, deployment.Spec.Template.Labels)
			}
		})
	}

	// serviceAccountName
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedDeployments []workerDeploymentServiceAccountTestCase
	}{
		{
			CaseName: "default service account",
			Release:  "production",
			ExpectedDeployments: []workerDeploymentServiceAccountTestCase{
				{
					ExpectedServiceAccountName: "",
				},
			},
		},
		{
			CaseName: "empty service account name",
			Release:  "production",
			Values: map[string]string{
				"serviceAccountName": "",
			},
			ExpectedDeployments: []workerDeploymentServiceAccountTestCase{
				{
					ExpectedServiceAccountName: "",
				},
			},
		},
		{
			CaseName: "custom service account name - myServiceAccount",
			Release:  "production",
			Values: map[string]string{
				"serviceAccountName": "myServiceAccount",
			},
			ExpectedDeployments: []workerDeploymentServiceAccountTestCase{
				{
					ExpectedServiceAccountName: "myServiceAccount",
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app":                 "auto-devops-examples/minimal-ruby-app",
				"gitlab.env":                 "prod",
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))

			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]
				require.Equal(t, expectedDeployment.ExpectedServiceAccountName, deployment.Spec.Template.Spec.ServiceAccountName)
			}
		})
	}

	// serviceAccount
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedDeployments []workerDeploymentServiceAccountTestCase
	}{
		{
			CaseName: "default service account",
			Release:  "production",
			ExpectedDeployments: []workerDeploymentServiceAccountTestCase{
				{
					ExpectedServiceAccountName: "",
				},
			},
		},
		{
			CaseName: "empty service account name",
			Release:  "production",
			Values: map[string]string{
				"serviceAccount.name": "",
			},
			ExpectedDeployments: []workerDeploymentServiceAccountTestCase{
				{
					ExpectedServiceAccountName: "",
				},
			},
		},
		{
			CaseName: "custom service account name - myServiceAccount",
			Release:  "production",
			Values: map[string]string{
				"serviceAccount.name": "myServiceAccount",
			},
			ExpectedDeployments: []workerDeploymentServiceAccountTestCase{
				{
					ExpectedServiceAccountName: "myServiceAccount",
				},
			},
		},
		{
			CaseName: "serviceAccount.name takes precedence over serviceAccountName",
			Release:  "production",
			Values: map[string]string{
				"serviceAccount.name": "myServiceAccount1",
				"serviceAccountName":  "myServiceAccount2",
			},
			ExpectedDeployments: []workerDeploymentServiceAccountTestCase{
				{
					ExpectedServiceAccountName: "myServiceAccount1",
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app":                 "auto-devops-examples/minimal-ruby-app",
				"gitlab.env":                 "prod",
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))

			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]
				require.Equal(
					t,
					expectedDeployment.ExpectedServiceAccountName,
					deployment.Spec.Template.Spec.ServiceAccountName,
				)
			}
		})
	}

	// worker lifecycle
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		ExpectedDeployments []workerDeploymentTestCase
	}{
		{
			CaseName: "lifecycle",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":                        "echo",
				"workers.worker1.command[1]":                        "worker1",
				"workers.worker1.lifecycle.preStop.exec.command[0]": "/bin/sh",
				"workers.worker1.lifecycle.preStop.exec.command[1]": "-c",
				"workers.worker1.lifecycle.preStop.exec.command[2]": "sleep 10",
				"workers.worker2.command[0]":                        "echo",
				"workers.worker2.command[1]":                        "worker2",
				"workers.worker2.lifecycle.preStop.exec.command[0]": "/bin/sh",
				"workers.worker2.lifecycle.preStop.exec.command[1]": "-c",
				"workers.worker2.lifecycle.preStop.exec.command[2]": "sleep 15",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedCmd:  []string{"echo", "worker1"},
					ExpectedLifecycle: &coreV1.Lifecycle{
						PreStop: &coreV1.LifecycleHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"/bin/sh", "-c", "sleep 10"},
							},
						},
					},
				},
				{
					ExpectedName: "production-worker2",
					ExpectedCmd:  []string{"echo", "worker2"},
					ExpectedLifecycle: &coreV1.Lifecycle{
						PreStop: &coreV1.LifecycleHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"/bin/sh", "-c", "sleep 15"},
							},
						},
					},
				},
			},
		},
		{
			CaseName: "preStopCommand",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":        "echo",
				"workers.worker1.command[1]":        "worker1",
				"workers.worker1.preStopCommand[0]": "/bin/sh",
				"workers.worker1.preStopCommand[1]": "-c",
				"workers.worker1.preStopCommand[2]": "sleep 10",
				"workers.worker2.command[0]":        "echo",
				"workers.worker2.command[1]":        "worker2",
				"workers.worker2.preStopCommand[0]": "/bin/sh",
				"workers.worker2.preStopCommand[1]": "-c",
				"workers.worker2.preStopCommand[2]": "sleep 15",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedCmd:  []string{"echo", "worker1"},
					ExpectedLifecycle: &coreV1.Lifecycle{
						PreStop: &coreV1.LifecycleHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"/bin/sh", "-c", "sleep 10"},
							},
						},
					},
				},
				{
					ExpectedName: "production-worker2",
					ExpectedCmd:  []string{"echo", "worker2"},
					ExpectedLifecycle: &coreV1.Lifecycle{
						PreStop: &coreV1.LifecycleHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"/bin/sh", "-c", "sleep 15"},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app": "auto-devops-examples/minimal-ruby-app",
				"gitlab.env": "prod",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))

			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]
				require.Equal(t, expectedDeployment.ExpectedName, deployment.Name)
				require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
				require.Equal(t, expectedDeployment.ExpectedCmd, deployment.Spec.Template.Spec.Containers[0].Command)
				require.Equal(t, expectedDeployment.ExpectedLifecycle, deployment.Spec.Template.Spec.Containers[0].Lifecycle)
			}
		})
	}

	// worker livenessProbe, and readinessProbe tests
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		ExpectedDeployments []workerDeploymentTestCase
	}{
		{
			CaseName: "default liveness and readiness values",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
				"workers.worker2.command[0]": "echo",
				"workers.worker2.command[1]": "worker2",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:           "production-worker1",
					ExpectedCmd:            []string{"echo", "worker1"},
					ExpectedLivenessProbe:  defaultLivenessProbe(),
					ExpectedReadinessProbe: defaultReadinessProbe(),
				},
				{
					ExpectedName:           "production-worker2",
					ExpectedCmd:            []string{"echo", "worker2"},
					ExpectedLivenessProbe:  defaultLivenessProbe(),
					ExpectedReadinessProbe: defaultReadinessProbe(),
				},
			},
		},
		{
			CaseName: "enableWorkerLivenessProbe",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":                         "echo",
				"workers.worker1.command[1]":                         "worker1",
				"workers.worker1.livenessProbe.path":                 "/worker",
				"workers.worker1.livenessProbe.scheme":               "HTTP",
				"workers.worker1.livenessProbe.probeType":            "httpGet",
				"workers.worker1.livenessProbe.httpHeaders[0].name":  "custom-header",
				"workers.worker1.livenessProbe.httpHeaders[0].value": "awesome",
				"workers.worker2.command[0]":                         "echo",
				"workers.worker2.command[1]":                         "worker2",
				"workers.worker2.livenessProbe.path":                 "/worker",
				"workers.worker2.livenessProbe.scheme":               "HTTP",
				"workers.worker2.livenessProbe.probeType":            "httpGet",
				"workers.worker2.livenessProbe.httpHeaders[0].name":  "custom-header",
				"workers.worker2.livenessProbe.httpHeaders[0].value": "awesome",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedCmd:  []string{"echo", "worker1"},
					ExpectedLivenessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							HTTPGet: &coreV1.HTTPGetAction{
								Path:   "/worker",
								Port:   intstr.FromInt(5000),
								Scheme: coreV1.URISchemeHTTP,
								HTTPHeaders: []coreV1.HTTPHeader{
									coreV1.HTTPHeader{
										Name:  "custom-header",
										Value: "awesome",
									},
								},
							},
						},
					},
					ExpectedReadinessProbe: defaultReadinessProbe(),
				},
				{
					ExpectedName: "production-worker2",
					ExpectedCmd:  []string{"echo", "worker2"},
					ExpectedLivenessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							HTTPGet: &coreV1.HTTPGetAction{
								Path:   "/worker",
								Port:   intstr.FromInt(5000),
								Scheme: coreV1.URISchemeHTTP,
								HTTPHeaders: []coreV1.HTTPHeader{
									coreV1.HTTPHeader{
										Name:  "custom-header",
										Value: "awesome",
									},
								},
							},
						},
					},
					ExpectedReadinessProbe: defaultReadinessProbe(),
				},
			},
		},
		{
			CaseName: "enableWorkerLivenessProbe exec",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":               "echo",
				"workers.worker1.command[1]":               "worker1",
				"workers.worker1.livenessProbe.probeType":  "exec",
				"workers.worker1.livenessProbe.command[0]": "echo",
				"workers.worker1.livenessProbe.command[1]": "hello",
				"workers.worker2.command[0]":               "echo",
				"workers.worker2.command[1]":               "worker2",
				"workers.worker2.livenessProbe.probeType":  "exec",
				"workers.worker2.livenessProbe.command[0]": "echo",
				"workers.worker2.livenessProbe.command[1]": "hello",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedCmd:  []string{"echo", "worker1"},
					ExpectedLivenessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"echo", "hello"},
							},
						},
					},
					ExpectedReadinessProbe: defaultReadinessProbe(),
				},
				{
					ExpectedName: "production-worker2",
					ExpectedCmd:  []string{"echo", "worker2"},
					ExpectedLivenessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"echo", "hello"},
							},
						},
					},
					ExpectedReadinessProbe: defaultReadinessProbe(),
				},
			},
		},
		{
			CaseName: "enableWorkerReadinessProbe",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":                          "echo",
				"workers.worker1.command[1]":                          "worker1",
				"workers.worker1.readinessProbe.path":                 "/worker",
				"workers.worker1.readinessProbe.scheme":               "HTTP",
				"workers.worker1.readinessProbe.probeType":            "httpGet",
				"workers.worker1.readinessProbe.httpHeaders[0].name":  "custom-header",
				"workers.worker1.readinessProbe.httpHeaders[0].value": "awesome",
				"workers.worker2.command[0]":                          "echo",
				"workers.worker2.command[1]":                          "worker2",
				"workers.worker2.readinessProbe.path":                 "/worker",
				"workers.worker2.readinessProbe.scheme":               "HTTP",
				"workers.worker2.readinessProbe.probeType":            "httpGet",
				"workers.worker2.readinessProbe.httpHeaders[0].name":  "custom-header",
				"workers.worker2.readinessProbe.httpHeaders[0].value": "awesome",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:          "production-worker1",
					ExpectedCmd:           []string{"echo", "worker1"},
					ExpectedLivenessProbe: defaultLivenessProbe(),
					ExpectedReadinessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							HTTPGet: &coreV1.HTTPGetAction{
								Path:   "/worker",
								Port:   intstr.FromInt(5000),
								Scheme: coreV1.URISchemeHTTP,
								HTTPHeaders: []coreV1.HTTPHeader{
									coreV1.HTTPHeader{
										Name:  "custom-header",
										Value: "awesome",
									},
								},
							},
						},
					},
				},
				{
					ExpectedName:          "production-worker2",
					ExpectedCmd:           []string{"echo", "worker2"},
					ExpectedLivenessProbe: defaultLivenessProbe(),
					ExpectedReadinessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							HTTPGet: &coreV1.HTTPGetAction{
								Path:   "/worker",
								Port:   intstr.FromInt(5000),
								Scheme: coreV1.URISchemeHTTP,
								HTTPHeaders: []coreV1.HTTPHeader{
									coreV1.HTTPHeader{
										Name:  "custom-header",
										Value: "awesome",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			CaseName: "enableWorkerReadinessProbe exec",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":                "echo",
				"workers.worker1.command[1]":                "worker1",
				"workers.worker1.readinessProbe.probeType":  "exec",
				"workers.worker1.readinessProbe.command[0]": "echo",
				"workers.worker1.readinessProbe.command[1]": "hello",
				"workers.worker2.command[0]":                "echo",
				"workers.worker2.command[1]":                "worker2",
				"workers.worker2.readinessProbe.probeType":  "exec",
				"workers.worker2.readinessProbe.command[0]": "echo",
				"workers.worker2.readinessProbe.command[1]": "hello",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName:          "production-worker1",
					ExpectedCmd:           []string{"echo", "worker1"},
					ExpectedLivenessProbe: defaultLivenessProbe(),
					ExpectedReadinessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"echo", "hello"},
							},
						},
					},
				},
				{
					ExpectedName:          "production-worker2",
					ExpectedCmd:           []string{"echo", "worker2"},
					ExpectedLivenessProbe: defaultLivenessProbe(),
					ExpectedReadinessProbe: &coreV1.Probe{
						ProbeHandler: coreV1.ProbeHandler{
							Exec: &coreV1.ExecAction{
								Command: []string{"echo", "hello"},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app": "auto-devops-examples/minimal-ruby-app",
				"gitlab.env": "prod",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))

			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]
				require.Equal(t, expectedDeployment.ExpectedName, deployment.Name)
				require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
				require.Equal(t, expectedDeployment.ExpectedCmd, deployment.Spec.Template.Spec.Containers[0].Command)
				require.Equal(t, expectedDeployment.ExpectedLivenessProbe, deployment.Spec.Template.Spec.Containers[0].LivenessProbe)
				require.Equal(t, expectedDeployment.ExpectedReadinessProbe, deployment.Spec.Template.Spec.Containers[0].ReadinessProbe)
			}
		})
	}

	// worker resources tests
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		ExpectedDeployments []workerDeploymentTestCase
	}{
		{
			CaseName: "default workers resources",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
				"workers.worker2.command[0]": "echo",
				"workers.worker2.command[1]": "worker2",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedCmd:  []string{"echo", "worker1"},
					ExpectedResources: coreV1.ResourceRequirements{
						Requests: coreV1.ResourceList{},
					},
				},
				{
					ExpectedName: "production-worker2",
					ExpectedCmd:  []string{"echo", "worker2"},
					ExpectedResources: coreV1.ResourceRequirements{
						Requests: coreV1.ResourceList{},
					},
				},
			},
		},
		{
			CaseName: "override workers requests resources",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":                "echo",
				"workers.worker1.command[1]":                "worker1",
				"workers.worker1.resources.requests.memory": "250M",
				"workers.worker2.command[0]":                "echo",
				"workers.worker2.command[1]":                "worker2",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedCmd:  []string{"echo", "worker1"},
					ExpectedResources: coreV1.ResourceRequirements{
						Requests: coreV1.ResourceList{
							"memory": resource.MustParse("250M"),
						},
					},
				},
				{
					ExpectedName: "production-worker2",
					ExpectedCmd:  []string{"echo", "worker2"},
					ExpectedResources: coreV1.ResourceRequirements{
						Requests: coreV1.ResourceList{},
					},
				},
			},
		},
		{
			CaseName: "override workers limits resources",
			Release:  "production",
			Values: map[string]string{
				"workers.worker1.command[0]":               "echo",
				"workers.worker1.command[1]":               "worker1",
				"workers.worker1.resources.limits.memory":  "500m",
				"workers.worker1.resources.limits.storage": "8Gi",
				"workers.worker2.command[0]":               "echo",
				"workers.worker2.command[1]":               "worker2",
				"workers.worker2.resources.limits.storage": "16Gi",
			},
			ExpectedDeployments: []workerDeploymentTestCase{
				{
					ExpectedName: "production-worker1",
					ExpectedCmd:  []string{"echo", "worker1"},
					ExpectedResources: coreV1.ResourceRequirements{
						Limits: coreV1.ResourceList{
							"memory":  resource.MustParse("500m"),
							"storage": resource.MustParse("8Gi"),
						},
					},
				},
				{
					ExpectedName: "production-worker2",
					ExpectedCmd:  []string{"echo", "worker2"},
					ExpectedResources: coreV1.ResourceRequirements{
						Limits: coreV1.ResourceList{
							"storage": resource.MustParse("16Gi"),
						},
					},
				},
			},
		},
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app": "auto-devops-examples/minimal-ruby-app",
				"gitlab.env": "prod",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/worker-deployment.yaml"}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			require.Len(t, deployments.Items, len(tc.ExpectedDeployments))

			for i, expectedDeployment := range tc.ExpectedDeployments {
				deployment := deployments.Items[i]
				require.Equal(t, expectedDeployment.ExpectedName, deployment.Name)
				require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
				require.Equal(t, expectedDeployment.ExpectedCmd, deployment.Spec.Template.Spec.Containers[0].Command)
				require.Equal(t, expectedDeployment.ExpectedResources, deployment.Spec.Template.Spec.Containers[0].Resources)
			}
		})
	}
}

func TestWorkerTemplateWithVolumeMounts(t *testing.T) {
	releaseName := "worker-with-volume-mounts-test"
	templates := []string{"templates/worker-deployment.yaml"}

	hostPathDirectoryType := coreV1.HostPathDirectory
	configMapOptional := false
	configMapDefaultMode := coreV1.ConfigMapVolumeSourceDefaultMode

	tcs := []struct {
		name                 string
		values               map[string]string
		valueFiles           []string
		expectedVolumes      []coreV1.Volume
		expectedVolumeMounts []coreV1.VolumeMount
		expectedErrorRegexp  *regexp.Regexp
	}{
		{
			name:       "with extra volume mounts",
			valueFiles: []string{"../testdata/extra-volume-mounts.yaml"},
			expectedVolumes: []coreV1.Volume{
				coreV1.Volume{
					Name: "config-volume",
					VolumeSource: coreV1.VolumeSource{
						ConfigMap: &coreV1.ConfigMapVolumeSource{
							coreV1.LocalObjectReference{
								Name: "test-config",
							},
							[]coreV1.KeyToPath{},
							&configMapDefaultMode,
							&configMapOptional,
						},
					},
				},
				coreV1.Volume{
					Name: "test-host-path",
					VolumeSource: coreV1.VolumeSource{
						HostPath: &coreV1.HostPathVolumeSource{
							Path: "/etc/ssl/certs/",
							Type: &hostPathDirectoryType,
						},
					},
				},
				coreV1.Volume{
					Name: "secret-volume",
					VolumeSource: coreV1.VolumeSource{
						Secret: &coreV1.SecretVolumeSource{
							SecretName: "mysecret",
						},
					},
				},
			},
			expectedVolumeMounts: []coreV1.VolumeMount{
				coreV1.VolumeMount{
					Name:      "config-volume",
					MountPath: "/app/config.yaml",
					SubPath:   "config.yaml",
				},
				coreV1.VolumeMount{
					Name:      "test-host-path",
					MountPath: "/etc/ssl/certs/",
					ReadOnly:  true,
				},
				coreV1.VolumeMount{
					Name:      "secret-volume",
					MountPath: "/etc/specialSecret",
					ReadOnly:  true,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			opts := &helm.Options{
				ValuesFiles: tc.valueFiles,
				SetValues:   tc.values,
			}
			output := mustRenderTemplate(t, opts, releaseName, templates, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			for _, deployment := range deployments.Items {
				for i, expectedVolume := range tc.expectedVolumes {
					require.Equal(t, expectedVolume.Name, deployment.Spec.Template.Spec.Volumes[i].Name)
					if deployment.Spec.Template.Spec.Volumes[i].PersistentVolumeClaim != nil {
						require.Equal(t, expectedVolume.PersistentVolumeClaim.ClaimName, deployment.Spec.Template.Spec.Volumes[i].PersistentVolumeClaim.ClaimName)
					}
					if deployment.Spec.Template.Spec.Volumes[i].ConfigMap != nil {
						require.Equal(t, expectedVolume.ConfigMap.Name, deployment.Spec.Template.Spec.Volumes[i].ConfigMap.Name)
					}
					if deployment.Spec.Template.Spec.Volumes[i].HostPath != nil {
						require.Equal(t, expectedVolume.HostPath.Path, deployment.Spec.Template.Spec.Volumes[i].HostPath.Path)
						require.Equal(t, expectedVolume.HostPath.Type, deployment.Spec.Template.Spec.Volumes[i].HostPath.Type)
					}
					if deployment.Spec.Template.Spec.Volumes[i].Secret != nil {
						require.Equal(t, expectedVolume.Secret.SecretName, deployment.Spec.Template.Spec.Volumes[i].Secret.SecretName)
					}
				}

				for i, expectedVolumeMount := range tc.expectedVolumeMounts {
					require.Equal(t, expectedVolumeMount.Name, deployment.Spec.Template.Spec.Containers[0].VolumeMounts[i].Name)
					require.Equal(t, expectedVolumeMount.MountPath, deployment.Spec.Template.Spec.Containers[0].VolumeMounts[i].MountPath)
					require.Equal(t, expectedVolumeMount.SubPath, deployment.Spec.Template.Spec.Containers[0].VolumeMounts[i].SubPath)
				}
			}
		})
	}
}

func TestWorkerDatabaseUrlEnvironmentVariable(t *testing.T) {
	releaseName := "worker-application-database-url-test"

	tcs := []struct {
		CaseName            string
		Values              map[string]string
		ExpectedDatabaseUrl string
		Template            string
	}{
		{
			CaseName: "present-worker",
			Values: map[string]string{
				"application.database_url":   "PRESENT",
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
			},
			ExpectedDatabaseUrl: "PRESENT",
			Template:            "templates/worker-deployment.yaml",
		},
		{
			CaseName: "missing-db-migrate",
			Values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
			},
			Template: "templates/worker-deployment.yaml",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.CaseName, func(t *testing.T) {

			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app": "auto-devops-examples/minimal-ruby-app",
				"gitlab.env": "prod",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, releaseName, []string{tc.Template}, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)

			if tc.ExpectedDatabaseUrl != "" {
				require.Contains(t, deployments.Items[0].Spec.Template.Spec.Containers[0].Env, coreV1.EnvVar{Name: "DATABASE_URL", Value: tc.ExpectedDatabaseUrl})
			} else {
				for _, envVar := range deployments.Items[0].Spec.Template.Spec.Containers[0].Env {
					require.NotEqual(t, "DATABASE_URL", envVar.Name)
				}
			}
		})
	}
}

func TestWorkerDeploymentTemplateWithExtraEnvFrom(t *testing.T) {
	releaseName := "worker-deployment-with-extra-envfrom-test"
	templates := []string{"templates/worker-deployment.yaml"}

	tcs := []struct {
		name            string
		values          map[string]string
		expectedEnvFrom coreV1.EnvFromSource
	}{
		{
			name: "with extra envfrom secret test",
			values: map[string]string{
				"workers.worker1.command[0]":                     "echo",
				"workers.worker1.command[1]":                     "worker1",
				"workers.worker1.extraEnvFrom[0].secretRef.name": "secret-name-test",
			},
			expectedEnvFrom: coreV1.EnvFromSource{
				SecretRef: &coreV1.SecretEnvSource{
					LocalObjectReference: coreV1.LocalObjectReference{
						Name: "secret-name-test",
					},
				},
			},
		},
		{
			name: "with extra envfrom with secretName test",
			values: map[string]string{
				"workers.worker1.command[0]":                     "echo",
				"workers.worker1.command[1]":                     "worker1",
				"application.secretName":                         "gitlab-secretname-test",
				"workers.worker1.extraEnvFrom[0].secretRef.name": "secret-name-test",
			},
			expectedEnvFrom: coreV1.EnvFromSource{
				SecretRef: &coreV1.SecretEnvSource{
					LocalObjectReference: coreV1.LocalObjectReference{
						Name: "secret-name-test",
					},
				},
			},
		},
		{
			name: "with extra envfrom configmap test",
			values: map[string]string{
				"workers.worker1.command[0]":                        "echo",
				"workers.worker1.command[1]":                        "worker1",
				"workers.worker1.extraEnvFrom[0].configMapRef.name": "configmap-name-test",
			},
			expectedEnvFrom: coreV1.EnvFromSource{
				ConfigMapRef: &coreV1.ConfigMapEnvSource{
					LocalObjectReference: coreV1.LocalObjectReference{
						Name: "configmap-name-test",
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			opts := &helm.Options{
				SetValues: tc.values,
			}
			output := mustRenderTemplate(t, opts, releaseName, templates, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)
			for _, deployment := range deployments.Items {
				require.Contains(t, deployment.Spec.Template.Spec.Containers[0].EnvFrom, tc.expectedEnvFrom)
			}
		})
	}
}

func TestWorkerDeploymentTemplateWithExtraEnv(t *testing.T) {
	releaseName := "worker-deployment-with-extra-env-test"
	templates := []string{"templates/worker-deployment.yaml"}

	tcs := []struct {
		name        string
		values      map[string]string
		expectedEnv coreV1.EnvVar
	}{
		{
			name: "with extra env secret test",
			values: map[string]string{
				"workers.worker1.command[0]": "echo",
				"workers.worker1.command[1]": "worker1",
				"workers.worker1.extraEnv[0].name": "env-name-test",
				"workers.worker1.extraEnv[0].value": "test-value",
			},
			expectedEnv: coreV1.EnvVar{
				Name:  "env-name-test",
				Value: "test-value",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			options := &helm.Options{
				SetValues: tc.values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, releaseName, templates, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)
			for _, deployment := range deployments.Items {
				require.Contains(t, deployment.Spec.Template.Spec.Containers[0].Env, tc.expectedEnv)
			}
		})
	}
}

func TestWorkerDeploymentTemplateWithSecurityContext(t *testing.T) {
	releaseName := "worker-deployment-with-security-context"
	templates := []string{"templates/worker-deployment.yaml"}

	tcs := []struct {
		name                        string
		values                      map[string]string
		expectedSecurityContextName string
	}{
		{
			name: "with gMSA security context",
			values: map[string]string{
				"workers.worker1.securityContext.windowsOptions.gmsaCredentialSpecName": "gmsa-test",
			},
			expectedSecurityContextName: "gmsa-test",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			opts := &helm.Options{
				SetValues: tc.values,
			}
			output := mustRenderTemplate(t, opts, releaseName, templates, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)
			for _, deployment := range deployments.Items {
				require.Equal(t, *deployment.Spec.Template.Spec.SecurityContext.WindowsOptions.GMSACredentialSpecName, tc.expectedSecurityContextName)
			}
		})
	}
}

func TestWorkerDeploymentTemplateWithContainerSecurityContext(t *testing.T) {
	releaseName := "worker-deployment-with-container-security-context"
	templates := []string{"templates/worker-deployment.yaml"}

	tcs := []struct {
		name                                string
		values                              map[string]string
		expectedSecurityContextCapabilities []coreV1.Capability
	}{
		{
			name: "with container security context capabilities",
			values: map[string]string{
				"workers.worker1.containerSecurityContext.capabilities.drop[0]": "ALL",
			},
			expectedSecurityContextCapabilities: []coreV1.Capability{
				"ALL",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			opts := &helm.Options{
				SetValues: tc.values,
			}
			output := mustRenderTemplate(t, opts, releaseName, templates, nil)

			var deployments deploymentAppsV1List
			helm.UnmarshalK8SYaml(t, output, &deployments)
			for _, deployment := range deployments.Items {
				require.Equal(t, deployment.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Drop, tc.expectedSecurityContextCapabilities)
			}
		})
	}
}
