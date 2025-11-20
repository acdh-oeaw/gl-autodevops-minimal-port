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

func TestRoleBindingTemplate(t *testing.T) {
	templates := []string{"templates/rolebinding.yaml"}
	release := "production"

	for _, tc := range []struct {
		CaseName string
		Values   map[string]string

		ExpectedErrorRegexp *regexp.Regexp

		ExpectedRoleBindings []struct {
			Name        string
			Labels      map[string]string
			Annotations map[string]string
			RoleRef     rbacV1.RoleRef
			Subjects    []rbacV1.Subject
		}
	}{
		{
			CaseName: "not created by default",
			Values:   map[string]string{},
			ExpectedErrorRegexp: regexp.MustCompile(
				"Error: could not find template templates/rolebinding.yaml in chart",
			),
		},
		{
			CaseName: "single role binding with basic configuration",
			Values: map[string]string{
				"roleBindings.test-binding.roleRefName":          "test-role",
				"roleBindings.test-binding.subjects[0].kind":    "ServiceAccount",
				"roleBindings.test-binding.subjects[0].name":    "test-account",
				"roleBindings.test-binding.subjects[0].namespace": "default",
			},
			ExpectedRoleBindings: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				RoleRef     rbacV1.RoleRef
				Subjects    []rbacV1.Subject
			}{
				{
					Name: "test-binding",
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
					RoleRef: rbacV1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "test-role",
					},
					Subjects: []rbacV1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "test-account",
							Namespace: "default",
						},
					},
				},
			},
		},
		{
			CaseName: "multiple role bindings",
			Values: map[string]string{
				"roleBindings.pod-reader-binding.roleRefName":          "pod-reader",
				"roleBindings.pod-reader-binding.subjects[0].kind":    "ServiceAccount",
				"roleBindings.pod-reader-binding.subjects[0].name":    "pod-reader-sa",
				"roleBindings.pod-reader-binding.subjects[0].namespace": "default",
				"roleBindings.secret-reader-binding.roleRefName":          "secret-reader",
				"roleBindings.secret-reader-binding.subjects[0].kind":    "ServiceAccount",
				"roleBindings.secret-reader-binding.subjects[0].name":    "secret-reader-sa",
				"roleBindings.secret-reader-binding.subjects[0].namespace": "default",
			},
			ExpectedRoleBindings: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				RoleRef     rbacV1.RoleRef
				Subjects    []rbacV1.Subject
			}{
				{
					Name: "pod-reader-binding",
					RoleRef: rbacV1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "pod-reader",
					},
					Subjects: []rbacV1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "pod-reader-sa",
							Namespace: "default",
						},
					},
				},
				{
					Name: "secret-reader-binding",
					RoleRef: rbacV1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "secret-reader",
					},
					Subjects: []rbacV1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "secret-reader-sa",
							Namespace: "default",
						},
					},
				},
			},
		},
		{
			CaseName: "role binding with multiple subjects",
			Values: map[string]string{
				"roleBindings.multi-subject-binding.roleRefName":          "admin-role",
				"roleBindings.multi-subject-binding.subjects[0].kind":    "ServiceAccount",
				"roleBindings.multi-subject-binding.subjects[0].name":    "admin-sa",
				"roleBindings.multi-subject-binding.subjects[0].namespace": "default",
				"roleBindings.multi-subject-binding.subjects[1].kind":    "User",
				"roleBindings.multi-subject-binding.subjects[1].name":    "admin-user",
				"roleBindings.multi-subject-binding.subjects[1].apiGroup": "rbac.authorization.k8s.io",
			},
			ExpectedRoleBindings: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				RoleRef     rbacV1.RoleRef
				Subjects    []rbacV1.Subject
			}{
				{
					Name: "multi-subject-binding",
					RoleRef: rbacV1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "admin-role",
					},
					Subjects: []rbacV1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "admin-sa",
							Namespace: "default",
						},
						{
							Kind:     "User",
							Name:     "admin-user",
							APIGroup: "rbac.authorization.k8s.io",
						},
					},
				},
			},
		},
		{
			CaseName: "role binding with extra labels",
			Values: map[string]string{
				"roleBindings.labeled-binding.roleRefName":          "test-role",
				"roleBindings.labeled-binding.subjects[0].kind":    "ServiceAccount",
				"roleBindings.labeled-binding.subjects[0].name":    "test-sa",
				"roleBindings.labeled-binding.subjects[0].namespace": "default",
				"extraLabels.environment":                          "test",
				"extraLabels.team":                                 "platform",
			},
			ExpectedRoleBindings: []struct {
				Name        string
				Labels      map[string]string
				Annotations map[string]string
				RoleRef     rbacV1.RoleRef
				Subjects    []rbacV1.Subject
			}{
				{
					Name: "labeled-binding",
					Labels: map[string]string{
						"environment": "test",
						"team":        "platform",
					},
					RoleRef: rbacV1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "test-role",
					},
					Subjects: []rbacV1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "test-sa",
							Namespace: "default",
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

			require.Equal(t, len(tc.ExpectedRoleBindings), len(list.Items))

			for i, expectedRoleBinding := range tc.ExpectedRoleBindings {
				var roleBinding rbacV1.RoleBinding
				helm.UnmarshalK8SYaml(t, string(list.Items[i].Raw), &roleBinding)

				require.Equal(t, expectedRoleBinding.Name, roleBinding.Name)
				require.Equal(t, expectedRoleBinding.RoleRef, roleBinding.RoleRef)
				require.Equal(t, expectedRoleBinding.Subjects, roleBinding.Subjects)

				// Check GitLab annotations are present when expected
				if expectedRoleBinding.Annotations != nil {
					for key, value := range expectedRoleBinding.Annotations {
						require.Equal(t, value, roleBinding.Annotations[key])
					}
				}

				// Check extra labels are present when expected
				if expectedRoleBinding.Labels != nil {
					for key, value := range expectedRoleBinding.Labels {
						require.Equal(t, value, roleBinding.Labels[key])
					}
				}

				// Verify standard labels are always present
				require.Equal(t, release, roleBinding.Labels["app"])
				require.Equal(t, release, roleBinding.Labels["release"])
				require.Equal(t, "Helm", roleBinding.Labels["heritage"])
			}
		})
	}
}
