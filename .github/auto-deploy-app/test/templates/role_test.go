package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	rbacV1 "k8s.io/api/rbac/v1"
	coreV1 "k8s.io/api/core/v1"
)

func TestRoleTemplate(t *testing.T) {
	templates := []string{"templates/role.yaml"}
	release := "production"

	for _, tc := range []struct {
		CaseName string
		Values   map[string]string

		ExpectedErrorRegexp *regexp.Regexp

		ExpectedRoles []struct {
			Name        string
			Labels      map[string]string
			Annotations map[string]string
			Rules       []rbacV1.PolicyRule
		}
	}{
		{
			CaseName: "not created by default",
			Values:   map[string]string{},
			ExpectedErrorRegexp: regexp.MustCompile(
				"Error: could not find template templates/role.yaml in chart",
			),
		},
		{
			CaseName: "single role with basic rules",
			Values: map[string]string{
				"roles.test-role.rules[0].apiGroups[0]": "",
				"roles.test-role.rules[0].resources[0]": "pods",
				"roles.test-role.rules[0].verbs[0]":     "get",
				"roles.test-role.rules[0].verbs[1]":     "list",
			},
			ExpectedRoles: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				Rules       []rbacV1.PolicyRule
			}{
				{
					Name: "test-role",
					Labels: map[string]string{
						"app":                          "production",
						"chart":                        chartName,
						"release":                      "production",
						"heritage":                     "Helm",
						"app.kubernetes.io/name":       "production",
						"helm.sh/chart":                chartName,
						"app.kubernetes.io/managed-by": "Helm",
						"app.kubernetes.io/instance":   "production",
					},
					Annotations: map[string]string{
						"app.gitlab.com/app": "auto-devops-examples/minimal-ruby-app",
						"app.gitlab.com/env": "prod",
					},
					Rules: []rbacV1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"pods"},
							Verbs:     []string{"get", "list"},
						},
					},
				},
			},
		},
		{
			CaseName: "multiple roles with different rules",
			Values: map[string]string{
				"roles.pod-reader.rules[0].apiGroups[0]": "",
				"roles.pod-reader.rules[0].resources[0]": "pods",
				"roles.pod-reader.rules[0].verbs[0]":     "get",
				"roles.pod-reader.rules[0].verbs[1]":     "list",
				"roles.secret-reader.rules[0].apiGroups[0]": "",
				"roles.secret-reader.rules[0].resources[0]": "secrets",
				"roles.secret-reader.rules[0].verbs[0]":     "get",
			},
			ExpectedRoles: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				Rules       []rbacV1.PolicyRule
			}{
				{
					Name: "pod-reader",
					Rules: []rbacV1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"pods"},
							Verbs:     []string{"get", "list"},
						},
					},
				},
				{
					Name: "secret-reader",
					Rules: []rbacV1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"secrets"},
							Verbs:     []string{"get"},
						},
					},
				},
			},
		},
		{
			CaseName: "role with complex rules",
			Values: map[string]string{
				"roles.admin-role.rules[0].apiGroups[0]":    "apps",
				"roles.admin-role.rules[0].resources[0]":    "deployments",
				"roles.admin-role.rules[0].resources[1]":    "replicasets",
				"roles.admin-role.rules[0].verbs[0]":        "get",
				"roles.admin-role.rules[0].verbs[1]":        "list",
				"roles.admin-role.rules[0].verbs[2]":        "create",
				"roles.admin-role.rules[0].verbs[3]":        "update",
				"roles.admin-role.rules[0].verbs[4]":        "delete",
				"roles.admin-role.rules[1].apiGroups[0]":    "",
				"roles.admin-role.rules[1].resources[0]":    "pods",
				"roles.admin-role.rules[1].resourceNames[0]": "specific-pod",
				"roles.admin-role.rules[1].verbs[0]":        "get",
			},
			ExpectedRoles: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				Rules       []rbacV1.PolicyRule
			}{
				{
					Name: "admin-role",
					Rules: []rbacV1.PolicyRule{
						{
							APIGroups: []string{"apps"},
							Resources: []string{"deployments", "replicasets"},
							Verbs:     []string{"get", "list", "create", "update", "delete"},
						},
						{
							APIGroups:     []string{""},
							Resources:     []string{"pods"},
							ResourceNames: []string{"specific-pod"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},
		{
			CaseName: "role with extra labels",
			Values: map[string]string{
				"roles.labeled-role.rules[0].apiGroups[0]": "",
				"roles.labeled-role.rules[0].resources[0]": "configmaps",
				"roles.labeled-role.rules[0].verbs[0]":     "get",
				"extraLabels.environment":                  "test",
				"extraLabels.team":                         "platform",
			},
			ExpectedRoles: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				Rules       []rbacV1.PolicyRule
			}{
				{
					Name: "labeled-role",
					Labels: map[string]string{
						"environment": "test",
						"team":        "platform",
					},
					Rules: []rbacV1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"configmaps"},
							Verbs:     []string{"get"},
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

			output := mustRenderTemplate(t, options, release, templates, tc.ExpectedErrorRegexp)

			if tc.ExpectedErrorRegexp != nil {
				return
			}

			var list coreV1.List
			helm.UnmarshalK8SYaml(t, output, &list)

			require.Equal(t, len(tc.ExpectedRoles), len(list.Items))

			for i, expectedRole := range tc.ExpectedRoles {
				var role rbacV1.Role
				helm.UnmarshalK8SYaml(t, string(list.Items[i].Raw), &role)

				require.Equal(t, expectedRole.Name, role.Name)
				require.Equal(t, expectedRole.Rules, role.Rules)

				// Check GitLab annotations are present when expected
				if expectedRole.Annotations != nil {
					for key, value := range expectedRole.Annotations {
						require.Equal(t, value, role.Annotations[key])
					}
				}

				// Check extra labels are present when expected
				if expectedRole.Labels != nil {
					for key, value := range expectedRole.Labels {
						require.Equal(t, value, role.Labels[key])
					}
				}

				// Verify standard labels are always present
				require.Equal(t, release, role.Labels["app"])
				require.Equal(t, release, role.Labels["release"])
				require.Equal(t, "Helm", role.Labels["heritage"])
			}
		})
	}
}
