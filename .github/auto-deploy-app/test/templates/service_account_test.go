package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
)

func TestServiceAccountTemplate(t *testing.T) {
	release := "production"

	for _, tc := range []struct {
		CaseName string
		Values   map[string]string

		ExpectedErrorRegexp *regexp.Regexp

		ExpectedName        string
		ExpectedAnnotations map[string]string
		ExpectedLabels      map[string]string
	}{
		{
			CaseName: "not created by default",
			Values:   map[string]string{},
			ExpectedErrorRegexp: regexp.MustCompile(
				"Error: could not find template templates/service-account.yaml in chart",
			),	
			ExpectedLabels: nil,
		},
		{
			CaseName: "not created if createNew is set to false",
			Values: map[string]string{
				"serviceAccount.createNew": "false",
			},
			ExpectedErrorRegexp: regexp.MustCompile(
				"Error: could not find template templates/service-account.yaml in chart",
			),
			ExpectedLabels: nil,
		},
		{
			CaseName: "no annotations",
			Values: map[string]string{
				"serviceAccount.createNew": "true",
				"serviceAccount.name":      "anAccountName",
			},
			ExpectedName:        "anAccountName",
			ExpectedAnnotations: nil,
			ExpectedLabels: nil,
		},
		{
			CaseName: "with annotations",
			Values: map[string]string{
				"serviceAccount.createNew":        "true",
				"serviceAccount.name":             "anAccountName",
				"serviceAccount.annotations.key1": "value1",
				"serviceAccount.annotations.key2": "value2",
			},
			ExpectedName: "anAccountName",
			ExpectedAnnotations: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			ExpectedLabels: nil,
		},
		{
			CaseName: "with labels",
			Values: map[string]string{
				"serviceAccount.createNew":        "true",
				"serviceAccount.name":             "anAccountName",
				"extraLabels.firstLabel":          "expected-label",
			},
			ExpectedName: "anAccountName",
			ExpectedAnnotations: nil,
			ExpectedLabels: map[string]string{
				"firstLabel": "expected-label",
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

			output := mustRenderTemplate(t, options, release, []string{"templates/service-account.yaml"}, tc.ExpectedErrorRegexp)

			if tc.ExpectedErrorRegexp != nil {
				return
            }

			var serviceAccount coreV1.ServiceAccount
			helm.UnmarshalK8SYaml(t, output, &serviceAccount)

			require.Equal(t, tc.ExpectedName, serviceAccount.Name)
			require.Equal(t, tc.ExpectedAnnotations, serviceAccount.Annotations)
			for key, value := range tc.ExpectedLabels {
				require.Equal(t, serviceAccount.ObjectMeta.Labels[key], value)
			}
		})
	}
}
