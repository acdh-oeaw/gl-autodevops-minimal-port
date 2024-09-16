package main

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCustomResource(t *testing.T) {
	releaseName := "custom-resource-test"
	Template := "templates/custom-resources.yaml" // Your template file path

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

			output := mustRenderTemplate(t, options, releaseName, []string{Template}, nil)

			var renderedObjects *unstructured.Unstructured
			helm.UnmarshalK8SYaml(t, output, &renderedObjects)
		})
	}
}

func TestCustomResourceWithTemplate(t *testing.T) {
	releaseName := "custom-resource-test-with-template"
	Template := "templates/custom-resources.yaml"

	tcs := []struct {
		CaseName     string
		Values       map[string]string
		expectedName string
	}{
		{
			CaseName: "test-single-custom-resource-template",
			Values: map[string]string{
				"customResources[0].apiVersion":    "traefik.containo.us/v1alpha1",
				"customResources[0].kind":          "IngressRoute",
				"customResources[0].metadata.name": "ingress-route-{{ .Release.Name }}",
			},
			expectedName: "ingress-route-" + releaseName,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.CaseName, func(t *testing.T) {

			namespaceName := "test-namespace-" + strings.ToLower(random.UniqueId())

			options := &helm.Options{
				SetValues:      tc.Values,
				KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
			}

			output := mustRenderTemplate(t, options, releaseName, []string{Template}, nil)

			var renderedObjects *unstructured.Unstructured
			helm.UnmarshalK8SYaml(t, output, &renderedObjects)

			// Check the name of the rendered object
			require.Equal(t, tc.expectedName, renderedObjects.GetName(), "The name of the custom resource should be %s as it is templated", tc.expectedName)
		})
	}
}
