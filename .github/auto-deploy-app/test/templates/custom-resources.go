package main

import (
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCustomResource(t *testing.T) {
	releaseName := "custom-resource-test"
	Template := "templates/custom-resource.yaml" // Your template file path

	tcs := []struct {
		CaseName string
		Values   map[string]string
	}{
		{
			CaseName: "test-single-custom-resource",
			Values: map[string]string{
				"customResources[0].apiVersion":    "traefik.containo.us/v1alpha1",
				"customResources[0].kind":          "IngressRoute",
				"customResources[0].metadata.name": "ingress-route",
			},
		},
		{
			CaseName: "test-multiple-custom-resources",
			Values: map[string]string{
				"customResources[0].apiVersion":    "traefik.containo.us/v1alpha1",
				"customResources[0].kind":          "IngressRoute",
				"customResources[0].metadata.name": "ingress-route",
				"customResources[1].apiVersion":    "v1",
				"customResources[1].kind":          "Pod",
				"customResources[1].metadata.name": "my-pod",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.CaseName, func(t *testing.T) {

			namespaceName := "test-namespace-" + strings.ToLower(random.UniqueId())

			options := &helm.Options{
				SetValues:      tc.Values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{Template})

			if err != nil {
				t.Error(err)
				return
			}

			var renderedObjects []*unstructured.Unstructured
			helm.UnmarshalK8SYaml(t, output, &renderedObjects)

			// Check if at least one custom resource is present
			require.GreaterOrEqual(t, len(renderedObjects), 1)
		})
	}
}
