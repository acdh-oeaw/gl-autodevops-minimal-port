package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"strings"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	helmChartPath = "../.."
)

var chartName string // dynamically initialized

func init() {
	// init chartName dynamically because it is annoying to update this value, but it is needed for some expected labels
	f, err := os.Open(helmChartPath + "/Chart.yaml")
	if err != nil {
		log.Fatalf("Failed to open Chart.yaml: %v", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read Chart.yaml: %v", err)
	}
	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal(b, m)
	if err != nil {
		log.Fatalf("Failed to unmarshal Chart.yaml: %v", err)
	}
	chartName = "auto-deploy-app-" + m["version"].(string)
}

func mustRenderTemplate(t *testing.T, opts *helm.Options, releaseName string, templates []string, expectedErrorRegexp *regexp.Regexp, extraHelmArgs ...string) (string) {

	output, err := helm.RenderTemplateE(t, opts, helmChartPath, releaseName, templates, extraHelmArgs...)
	if expectedErrorRegexp != nil {
		if err == nil {
			t.Fatalf("Expected error but didn't happen")
		} else {
			require.Regexp(t, expectedErrorRegexp, err.Error())
		}
		return ""
	}
	if err != nil {
		t.Fatalf("failed to render helm template: %s", err.Error())
		return ""
	}

	// yamllint with extra config
	// check indenting of sequences with the default k8s style
	// disable trailing-space check, because sometimes we have empty variables and we don't want to use if blocks around every option
	cmd := exec.Command("yamllint", "-s", "-d", "{extends: default, rules: {line-length: {max: 160}, indentation: {indent-sequences: false}, trailing-spaces: disable}}", "-")
	cmd.Stdin = strings.NewReader(output + "\n")
	var out strings.Builder
	cmd.Stdout = &out
	err = cmd.Run()

	if err != nil {
		t.Fatalf("rendered template had yamllint errors: %s", out.String())
		return ""
	}

	// needed, because yamllint does not detect empty lines containing spaces
	require.NotRegexpf(t, regexp.MustCompile("\n[[:space:]]*\n"), output, "found empty lines in output")

	return output
}

type workerDeploymentTestCase struct {
	ExpectedName           string
	ExpectedCmd            []string
	ExpectedStrategyType   appsV1.DeploymentStrategyType
	ExpectedSelector       *metav1.LabelSelector
	ExpectedLifecycle      *coreV1.Lifecycle
	ExpectedLivenessProbe  *coreV1.Probe
	ExpectedReadinessProbe *coreV1.Probe
	ExpectedNodeSelector   map[string]string
	ExpectedTolerations    []coreV1.Toleration
	ExpectedInitContainers []coreV1.Container
	ExpectedAffinity       *coreV1.Affinity
	ExpectedResources      coreV1.ResourceRequirements
}

type cronjobTestCase struct {
	ExpectedName     string
	ExpectedCmd      []string
	ExpectedSchedule string
}

type workerDeploymentSelectorTestCase struct {
	ExpectedName     string
	ExpectedSelector *metav1.LabelSelector
}

type workerDeploymentServiceAccountTestCase struct {
	ExpectedServiceAccountName string
}

type workerDeploymentHostNetworkTestCase struct {
	ExpectedHostNetwork bool
}

type deploymentList struct {
	metav1.TypeMeta `json:",inline"`

	Items []appsV1.Deployment `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type deploymentAppsV1List struct {
	metav1.TypeMeta `json:",inline"`

	Items []appsV1.Deployment `json:"items" protobuf:"bytes,2,rep,name=items"`
}

func mergeStringMap(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func defaultLivenessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path:   "/",
				Port:   intstr.FromInt(5000),
				Scheme: coreV1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 15,
		TimeoutSeconds:      15,
	}
}

func defaultReadinessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path:   "/",
				Port:   intstr.FromInt(5000),
				Scheme: coreV1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 5,
		TimeoutSeconds:      3,
	}
}

func workerLivenessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path:   "/worker",
				Port:   intstr.FromInt(5000),
				Scheme: coreV1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 0,
		TimeoutSeconds:      0,
	}
}

func workerReadinessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			HTTPGet: &coreV1.HTTPGetAction{
				Path:   "/worker",
				Port:   intstr.FromInt(5000),
				Scheme: coreV1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 0,
		TimeoutSeconds:      0,
	}
}

func execReadinessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			Exec: &coreV1.ExecAction{
				Command: []string{"echo", "hello"},
			},
		},
		InitialDelaySeconds: 0,
		TimeoutSeconds:      0,
	}
}

func execLivenessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			Exec: &coreV1.ExecAction{
				Command: []string{"echo", "hello"},
			},
		},
		InitialDelaySeconds: 0,
		TimeoutSeconds:      0,
	}
}

func tcpLivenessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			TCPSocket: &coreV1.TCPSocketAction{
				Port: intstr.IntOrString{IntVal: 5000},
			},
		},
		InitialDelaySeconds: 0,
		TimeoutSeconds:      0,
	}
}

func tcpReadinessProbe() *coreV1.Probe {
	return &coreV1.Probe{
		ProbeHandler: coreV1.ProbeHandler{
			TCPSocket: &coreV1.TCPSocketAction{
				Port: intstr.IntOrString{IntVal: 5000},
			},
		},
		InitialDelaySeconds: 0,
		TimeoutSeconds:      0,
	}
}
