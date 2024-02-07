package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestCronjobMeta(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedName    string
		ExpectedRelease string
		ExpectedLabels  map[string]string
	}{
		{
			CaseName: "default",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedLabels:	 nil,
		},
		{
			CaseName: "extraLabels",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
				"extraLabels.firstLabel":    "expected-label",
			},
			ExpectedName:    "production",
			ExpectedRelease: "production",
			ExpectedLabels:	 map[string]string{
				"firstLabel": "expected-label",
			},
		},
		{
			CaseName: "overriden release",
			Release:  "production",
			Values: map[string]string{
				"releaseOverride":          "productionOverridden",
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
			},
			ExpectedName:    "productionOverridden",
			ExpectedRelease: "production",
			ExpectedLabels:	 nil,
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, map[string]string{
					"app.gitlab.com/app": "auto-devops-examples/minimal-ruby-app",
					"app.gitlab.com/env": "prod",
				}, cronjob.Annotations)

				ExpectedLabels := map[string]string{
					"app":                          tc.ExpectedName,
					"chart":                        chartName,
					"heritage":                     "Helm",
					"release":                      tc.ExpectedRelease,
					"tier":                         "web",
					"track":                        "stable",
					"app.kubernetes.io/name":       tc.ExpectedName,
					"helm.sh/chart":                chartName,
					"app.kubernetes.io/managed-by": "Helm",
					"app.kubernetes.io/instance":   tc.ExpectedRelease,
				}
				mergeStringMap(ExpectedLabels, tc.ExpectedLabels)
				require.Equal(t, ExpectedLabels, cronjob.Labels)

				require.Equal(t, map[string]string{
					"app.gitlab.com/app":           "auto-devops-examples/minimal-ruby-app",
					"app.gitlab.com/env":           "prod",
					"checksum/application-secrets": "",
				}, cronjob.Spec.JobTemplate.Spec.Template.Annotations)
				require.Equal(t, map[string]string{
					"app":     tc.ExpectedName,
					"release": tc.ExpectedRelease,
					"tier":    "cronjob",
					"track":   "stable",
				}, cronjob.Spec.JobTemplate.Spec.Template.Labels)
			}
		})
	}
}

func TestCronjobSchedule(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedSchedule string
	}{
		{
			CaseName: "test two schedules for different cronjobs",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.schedule": "*/2 * * * *",

				"cronjobs.job2.schedule": "*/2 * * * *",
			},
			ExpectedSchedule: "*/2 * * * *",
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedSchedule, cronjob.Spec.Schedule)
			}
		})
	}
}

func TestCronjobImage(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Release  string
		Values   map[string]string

		ExpectedImage string
	}{
		{
			CaseName: "default image",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
			},
			ExpectedImage: "gitlab.example.com/group/project:stable",
		},
		{
			CaseName: "alpine latest image",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.image.repository": "alpine",
				"cronjobs.job1.image.tag":        "latest",

				"cronjobs.job2.image.repository": "alpine",
				"cronjobs.job2.image.tag":        "latest",
			},
			ExpectedImage: "alpine:latest",
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedImage, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
			}
		})
	}
}

func TestCronjobLivenessAndReadiness(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		ExpectedLivenessProbe  *coreV1.Probe
		ExpectedReadinessProbe *coreV1.Probe
	}{
		{
			CaseName: "default liveness and readiness values",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
			},
			ExpectedLivenessProbe:  defaultLivenessProbe(),
			ExpectedReadinessProbe: defaultReadinessProbe(),
		},
		{
			CaseName: "enable liveness probe",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.livenessProbe.path":      "/worker",
				"cronjobs.job1.livenessProbe.scheme":    "HTTP",
				"cronjobs.job1.livenessProbe.probeType": "httpGet",
				"cronjobs.job2.livenessProbe.path":      "/worker",
				"cronjobs.job2.livenessProbe.scheme":    "HTTP",
				"cronjobs.job2.livenessProbe.probeType": "httpGet",
			},
			ExpectedLivenessProbe:  workerLivenessProbe(),
			ExpectedReadinessProbe: defaultReadinessProbe(),
		},
		{
			CaseName: "enable readiness probe",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.readinessProbe.path":      "/worker",
				"cronjobs.job1.readinessProbe.scheme":    "HTTP",
				"cronjobs.job1.readinessProbe.probeType": "httpGet",
				"cronjobs.job2.readinessProbe.path":      "/worker",
				"cronjobs.job2.readinessProbe.scheme":    "HTTP",
				"cronjobs.job2.readinessProbe.probeType": "httpGet",
			},

			ExpectedLivenessProbe:  defaultLivenessProbe(),
			ExpectedReadinessProbe: workerReadinessProbe(),
		},
		{
			CaseName: "enable exec readiness probe",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.readinessProbe.command[0]": "echo",
				"cronjobs.job1.readinessProbe.command[1]": "hello",
				"cronjobs.job1.readinessProbe.probeType":  "exec",

				"cronjobs.job2.readinessProbe.command[0]": "echo",
				"cronjobs.job2.readinessProbe.command[1]": "hello",
				"cronjobs.job2.readinessProbe.probeType":  "exec",
			},

			ExpectedLivenessProbe:  defaultLivenessProbe(),
			ExpectedReadinessProbe: execReadinessProbe(),
		},
		{
			CaseName: "enable exec liveness probe",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.livenessProbe.command[0]": "echo",
				"cronjobs.job1.livenessProbe.command[1]": "hello",
				"cronjobs.job1.livenessProbe.probeType":  "exec",

				"cronjobs.job2.livenessProbe.command[0]": "echo",
				"cronjobs.job2.livenessProbe.command[1]": "hello",
				"cronjobs.job2.livenessProbe.probeType":  "exec",
			},

			ExpectedLivenessProbe:  execLivenessProbe(),
			ExpectedReadinessProbe: defaultReadinessProbe(),
		},
		{
			CaseName: "enable TCP readiness probe",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.readinessProbe.port":      "5000",
				"cronjobs.job1.readinessProbe.probeType": "tcpSocket",
				"cronjobs.job2.readinessProbe.port":      "5000",
				"cronjobs.job2.readinessProbe.probeType": "tcpSocket",
			},

			ExpectedLivenessProbe:  defaultLivenessProbe(),
			ExpectedReadinessProbe: tcpReadinessProbe(),
		},
		{
			CaseName: "enable TCP liveness probe",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.livenessProbe.port":      "5000",
				"cronjobs.job1.livenessProbe.probeType": "tcpSocket",
				"cronjobs.job2.livenessProbe.port":      "5000",
				"cronjobs.job2.livenessProbe.probeType": "tcpSocket",
			},

			ExpectedLivenessProbe:  tcpLivenessProbe(),
			ExpectedReadinessProbe: defaultReadinessProbe(),
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedLivenessProbe, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].LivenessProbe)
				require.Equal(t, tc.ExpectedReadinessProbe, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].ReadinessProbe)
			}
		})
	}
}

func TestCronjobNodeSelector(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		ExpectedNodeSelector map[string]string
	}{
		{
			CaseName: "global nodeSelector",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
				"nodeSelector.disktype":    "ssd",
			},

			ExpectedNodeSelector: map[string]string{"disktype": "ssd"},
		},
		{
			CaseName: "added nodeSelector",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.nodeSelector.disktype": "ssd",
				"cronjobs.job2.nodeSelector.disktype": "ssd",
			},

			ExpectedNodeSelector: map[string]string{"disktype": "ssd"},
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedNodeSelector, cronjob.Spec.JobTemplate.Spec.Template.Spec.NodeSelector)
			}
		})
	}
}

func TestCronjobTolerations(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		EoxpectedNodeSelector map[string]string
		ExpectedTolerations   []coreV1.Toleration
	}{
		{
			CaseName: "global tolerations",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
				"tolerations[0].key":       "key1",
				"tolerations[0].operator":  "Equal",
				"tolerations[0].value":     "value1",
				"tolerations[0].effect":    "NoSchedule",
			},

			ExpectedTolerations: []coreV1.Toleration{
				{
					Key:      "key1",
					Operator: "Equal",
					Value:    "value1",
					Effect:   "NoSchedule",
				},
			},
		},
		{
			CaseName: "added tolerations",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.tolerations[0].key":      "key1",
				"cronjobs.job1.tolerations[0].operator": "Equal",
				"cronjobs.job1.tolerations[0].value":    "value1",
				"cronjobs.job1.tolerations[0].effect":   "NoSchedule",

				"cronjobs.job2.tolerations[0].key":      "key1",
				"cronjobs.job2.tolerations[0].operator": "Equal",
				"cronjobs.job2.tolerations[0].value":    "value1",
				"cronjobs.job2.tolerations[0].effect":   "NoSchedule",
			},
			ExpectedTolerations: []coreV1.Toleration{
				{
					Key:      "key1",
					Operator: "Equal",
					Value:    "value1",
					Effect:   "NoSchedule",
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedTolerations, cronjob.Spec.JobTemplate.Spec.Template.Spec.Tolerations)
			}
		})
	}
}

func TestCronjobResources(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		EoxpectedNodeSelector map[string]string
		ExpectedResources     coreV1.ResourceRequirements
	}{
		{
			CaseName: "default",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
			},

			ExpectedResources: coreV1.ResourceRequirements{
				Limits:   coreV1.ResourceList(nil),
				Requests: coreV1.ResourceList{},
			},
		},
		{
			CaseName: "added resources",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"resources.limits.cpu":      "500m",
				"resources.limits.memory":   "4Gi",
				"resources.requests.cpu":    "200m",
				"resources.requests.memory": "2Gi",
			},

			ExpectedResources: coreV1.ResourceRequirements{
				Limits:   coreV1.ResourceList{
					"cpu": resource.MustParse("500m"),
					"memory": resource.MustParse("4Gi"),},
				Requests: coreV1.ResourceList{
					"cpu": resource.MustParse("200m"),
					"memory": resource.MustParse("2Gi"),
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedResources, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Resources )
			}
		})
	}
}

func TestCronjobTemplateWithVolumeMounts(t *testing.T) {
	releaseName := "cronjob-with-volume-mounts-test"

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
			options := &helm.Options{
				ValuesFiles: tc.valueFiles,
				SetValues:   tc.values,
			}
			output := mustRenderTemplate(t, options, releaseName, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				for i, expectedVolume := range tc.expectedVolumes {
					require.Equal(t, expectedVolume.Name, cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].Name)
					if cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].ConfigMap != nil {
						require.Equal(t, expectedVolume.ConfigMap.Name, cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].ConfigMap.Name)
					}
					if cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].HostPath != nil {
						require.Equal(t, expectedVolume.HostPath.Path, cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].HostPath.Path)
						require.Equal(t, expectedVolume.HostPath.Type, cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].HostPath.Type)
					}
					if cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].Secret != nil {
						require.Equal(t, expectedVolume.Secret.SecretName, cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes[i].Secret.SecretName)
					}
				}

				for i, expectedVolumeMount := range tc.expectedVolumeMounts {
					require.Equal(t, expectedVolumeMount.Name, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts[i].Name)
					require.Equal(t, expectedVolumeMount.MountPath, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts[i].MountPath)
					require.Equal(t, expectedVolumeMount.SubPath, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts[i].SubPath)
				}
			}
		})
	}
}

func TestCronjobAffinity(t *testing.T) {
	for _, tc := range []struct {
		CaseName string
		Values   map[string]string
		Release  string

		ExpectedAffinity *coreV1.Affinity
	}{{
		CaseName: "global affinity",
		Release:  "production",
		Values: map[string]string{
			"cronjobs.job1.command[0]": "echo",
			"cronjobs.job1.args[0]":    "hello",
			"cronjobs.job2.command[0]": "echo",
			"cronjobs.job2.args[0]":    "hello",

			"affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key":      "key1",
			"affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator": "DoesNotExist",
		},
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
		{
			CaseName: "added affinity",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key":      "key1",
				"cronjobs.job1.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator": "DoesNotExist",
				"cronjobs.job2.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key":      "key1",
				"cronjobs.job2.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator": "DoesNotExist",
			},
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
	} {
		t.Run(tc.CaseName, func(t *testing.T) {
			namespaceName := "minimal-ruby-app-" + strings.ToLower(random.UniqueId())

			values := map[string]string{
				"gitlab.app": "auto-devops-examples/minimal-ruby-app",
				"gitlab.env": "prod",
			}

			mergeStringMap(values, tc.Values)

			options := &helm.Options{
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedAffinity, cronjob.Spec.JobTemplate.Spec.Template.Spec.Affinity)
			}
		})
	}
}

func TestCronJobTemplateWithExtraEnvFrom(t *testing.T) {
	releaseName := "cronjob-with-extra-envfrom-test"

	tcs := []struct {
		name            string
		values          map[string]string
		expectedEnvFrom coreV1.EnvFromSource
	}{
		{
			name: "with extra envfrom secret test",
			values: map[string]string{
				"cronjobs.job1.schedule":                       "*/2 * * * *",
				"cronjobs.job1.extraEnvFrom[0].secretRef.name": "secret-name-test",
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
				"cronjobs.job1.schedule":                       "*/2 * * * *",
				"application.secretName":                       "gitlab-secretname-test",
				"cronjobs.job1.extraEnvFrom[0].secretRef.name": "secret-name-test",
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
				"cronjobs.job1.schedule":                          "*/2 * * * *",
				"cronjobs.job1.extraEnvFrom[0].configMapRef.name": "configmap-name-test",
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
			options := &helm.Options{
				SetValues: tc.values,
			}
			output := mustRenderTemplate(t, options, releaseName, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)
			for _, cronjob := range cronjobs.Items {
				require.Contains(t, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom, tc.expectedEnvFrom)
			}
		})
	}
}

func TestCronJobTemplateWithSecurityContext(t *testing.T) {
	releaseName := "cronjob-with-security-context"

	tcs := []struct {
		name                        string
		values                      map[string]string
		expectedSecurityContextName string
	}{
		{
			name: "with gMSA security context",
			values: map[string]string{
				"cronjobs.job1.securityContext.windowsOptions.gmsaCredentialSpecName": "gmsa-test",
			},
			expectedSecurityContextName: "gmsa-test",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			options := &helm.Options{
				SetValues: tc.values,
			}

			output := mustRenderTemplate(t, options, releaseName, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)
			for _, cronjob := range cronjobs.Items {
				require.Equal(t, *cronjob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext.WindowsOptions.GMSACredentialSpecName, tc.expectedSecurityContextName)
			}
		})
	}
}

func TestCronJobTemplateWithContainerSecurityContext(t *testing.T) {
	releaseName := "cronjob-with-container-security-context"

	tcs := []struct {
		name                        				string
		values                      				map[string]string
		expectedSecurityContextCapabilities []coreV1.Capability
	}{
		{
			name: "with container security context capabilities",
			values: map[string]string{
				"cronjobs.job1.containerSecurityContext.capabilities.drop[0]": "ALL",
			},
			expectedSecurityContextCapabilities: []coreV1.Capability{
				"ALL",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			options := &helm.Options{
				SetValues: tc.values,
			}

			output := mustRenderTemplate(t, options, releaseName, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)
			for _, cronjob := range cronjobs.Items {
				require.Equal(t, cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Drop, tc.expectedSecurityContextCapabilities)
			}
		})
	}
}

func TestCronjobImagePullSecrets(t *testing.T) {
	for _, tc := range []struct {
		CaseName                   string
		Values                     map[string]string
		Release 				   string
		ExpectedImagePullSecrets   []coreV1.LocalObjectReference
	}{
		{
			CaseName: "default secret",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
			},

			ExpectedImagePullSecrets: []coreV1.LocalObjectReference{
				{
					Name: "gitlab-registry",
				},
			},
		},
		{
			CaseName: "present secret",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
				"image.secrets[0].name":    "expected-secret",
			},

			ExpectedImagePullSecrets: []coreV1.LocalObjectReference{
				{
					Name: "expected-secret",
				},
			},
		},
		{
			CaseName: "multiple secrets",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
				"image.secrets[0].name":    "expected-secret",
				"image.secrets[1].name":    "additional-secret",
			},

			ExpectedImagePullSecrets: []coreV1.LocalObjectReference{
				{
					Name: "expected-secret",
				},
				{
					Name: "additional-secret",
				},
			},
		},
		{
			CaseName: "missing secret",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
				"image.secrets":            "null",
			},

			ExpectedImagePullSecrets: nil,
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedImagePullSecrets, cronjob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets)
			}
		})
	}
}

func TestCronjobPodAnnotations(t *testing.T) {
	for _, tc := range []struct {
		CaseName                   string
		Values                     map[string]string
		Release 				   string
		ExpectedPodAnnotations     map[string]string
	}{
		{
			CaseName: "one podAnnotations",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]":           "echo",
				"cronjobs.job1.args[0]":              "hello",
				"cronjobs.job2.command[0]":           "echo",
				"cronjobs.job2.args[0]":              "hello",
				"podAnnotations.firstAnnotation":    "expected-annotation",
			},

			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
				"app.gitlab.com/app":           "auto-devops-examples/minimal-ruby-app",
				"app.gitlab.com/env":           "prod",
				"firstAnnotation":              "expected-annotation",
			},
		},
		{
			CaseName: "multiple podAnnotations",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]":           "echo",
				"cronjobs.job1.args[0]":              "hello",
				"cronjobs.job2.command[0]":           "echo",
				"cronjobs.job2.args[0]":              "hello",
				"podAnnotations.firstAnnotation":    "expected-annotation",
				"podAnnotations.secondAnnotation":   "expected-annotation",
			},

			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
				"app.gitlab.com/app":           "auto-devops-examples/minimal-ruby-app",
				"app.gitlab.com/env":           "prod",
				"firstAnnotation":              "expected-annotation",
				"secondAnnotation":             "expected-annotation",
			},
		},
		{
			CaseName: "no podAnnotations",
			Release:  "production",
			Values: map[string]string{
				"cronjobs.job1.command[0]": "echo",
				"cronjobs.job1.args[0]":    "hello",
				"cronjobs.job2.command[0]": "echo",
				"cronjobs.job2.args[0]":    "hello",
			},

			ExpectedPodAnnotations: map[string]string{
				"checksum/application-secrets": "",
				"app.gitlab.com/app":           "auto-devops-examples/minimal-ruby-app",
				"app.gitlab.com/env":           "prod",
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
				ValuesFiles:    []string{},
				SetValues:      values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, tc.Release, []string{"templates/cronjob.yaml"}, nil)

			var cronjobs batchV1beta1.CronJobList
			helm.UnmarshalK8SYaml(t, output, &cronjobs)

			for _, cronjob := range cronjobs.Items {
				require.Equal(t, tc.ExpectedPodAnnotations, cronjob.Spec.JobTemplate.Spec.Template.ObjectMeta.Annotations)
			}
		})
	}
}