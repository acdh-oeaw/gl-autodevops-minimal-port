A port of the Gitlab AutoDevOps Workflow to GitHub Actions
==========================================================

This repository contains [reusable workflows](https://docs.github.com/en/actions/using-workflows/reusing-workflows) that try to achieve the same result as the

* build
* custom test and
* deploy

[stages of the Gitlab AutoDevOps](https://docs.gitlab.com/ee/topics/autodevops/stages.html) workflow.

Usage
-----

To use this you add a `starter.yaml` to `.github/workflows` in your GitHub repository with something like this:

```yaml
name: workflows starter
# env: is empty, see setup-env and the outputs there
on:
  push: {}
jobs:
  setup_workflow_env:
    runs-on: ubuntu-latest
# Optionally specify the environment that should be used on this branch
#    environment: my environment
    outputs:
# It seems you have to specify the environment twice (passed to reusable workflow)
# as there is no way yet to get the active environment
#      environment: my environment
      image_tagged: your-image-name
      registry_root: ghcr.io/${{ github.repository }}/
      default_port: "5000"
      source_image: tmp-cnb-image
#      herokuish_base_image: ghcr.io/acdh-oeaw/herokuish-for-cypress/main
      APP_NAME: your-app-name
# This together with the branch name is also used as the namespace to deploy to
      APP_ROOT: "/"
      SERVICE_ID: "99999999"
      PUBLIC_URL: https://your service host name.acdh-cluster.arz.oeaw.ac.at or acdh-dev.oeaw.acat or acdh.oeaw.ac.at
      POSTGRES_ENABLED: "false"
# You should not need to have to change anything below this line
#-----------------------------------------------------------------------------------------------------
    steps:
      - run: "/bin/true"      
  _1:
    needs: [setup_workflow_env]
    uses: acdh-oeaw/gl-autodevops-minimal-port/.github/workflows/build-cnb-and-push-to-registry.yaml@main
    secrets: inherit
# if you run this outside of acdh-oeaw yo uneed to specify every secret you want to pass by name
    with:
      registry_root: ${{ needs.setup_workflow_env.outputs.registry_root }}
      image_tagged: ${{ needs.setup_workflow_env.outputs.image_tagged }}
      source_image: ${{ needs.setup_workflow_env.outputs.source_image }}
      default_port: ${{ needs.setup_workflow_env.outputs.default_port }}
  _2:
    needs: [setup_workflow_env]
    uses: acdh-oeaw/gl-autodevops-minimal-port/.github/workflows/herokuish-tests-db-url.yaml@main
    secrets: inherit
# if you run this outside of acdh-oeaw yo uneed to specify every secret you want to pass by name
    with:
      registry_root: ${{ needs.setup_workflow_env.outputs.registry_root }}
      image_tagged: ${{ needs.setup_workflow_env.outputs.image_tagged }}
      source_image: ${{ needs.setup_workflow_env.outputs.source_image }}
      default_port: ${{ needs.setup_workflow_env.outputs.default_port }}
      herokuish_base_image: ${{ needs.setup_workflow_env.outputs.herokuish_base_image }}
      POSTGRES_ENABLED: ${{ needs.setup_workflow_env.outputs.POSTGRES_ENABLED }}
  _3:
    needs: [setup_workflow_env, _1, _2]
    uses: acdh-oeaw/gl-autodevops-minimal-port/.github/workflows/deploy.yml@main
    secrets: inherit
# if you run this outside of acdh-oeaw yo uneed to specify every secret you want to pass by name
#      KUBE_CONFIG: ${{ secrets.KUBE_CONFIG }}
#      KUBE_INGRESS_BASE_DOMAIN: ${{ secrets.KUBE_INGRESS_BASE_DOMAIN }}
#      POSTGRES_USER: ${{ secrets.POSTGRES_USER }}
#      POSTGRES_PASSWORD: ${{ secrets.POSTGRES_PASSWORD }}
#      POSTGRES_DB: ${{ secrets.POSTGRES_DB }}
#      K8S_SECRET_A_VAR_NAME: ${{  }}
    with:
      DOCKER_TAG: ${{ needs.setup_workflow_env.outputs.registry_root }}${{ needs.setup_workflow_env.outputs.image_tagged }}/${{ github.ref_name }}
      APP_NAME: ${{ needs.setup_workflow_env.outputs.APP_NAME }}
      APP_ROOT: ${{ needs.setup_workflow_env.outputs.APP_ROOT }}
      SERVICE_ID: ${{ needs.setup_workflow_env.outputs.SERVICE_ID }}
      PUBLIC_URL: ${{ needs.setup_workflow_env.outputs.PUBLIC_URL }}
      POSTGRES_ENABLED: ${{ needs.setup_workflow_env.outputs.POSTGRES_ENABLED == 'true'}}
      environment: "${{ needs.setup_workflow_env.outputs.environment}}"
      default_port: "${{ needs.setup_workflow_env.outputs.default_port}}"
```

You can pass many parameters as secrets like in gitlab. For example `KUBE_NAMESPACE` or `HELM_UPGRADE_EXTRA_ARGS`.
You can also use environments for having different parameters.

TODO
----

Nothing right now

Source of the auto-deploy-app
-----------------------------

The [auto-deploy-app](https://gitlab.com/gitlab-org/cluster-integration/auto-deploy-image/-/tree/master/assets/auto-deploy-app) helm chart is part of the [Gitlab cluster-integration auto-deploy-image repository](https://gitlab.com/gitlab-org/cluster-integration/auto-deploy-image)

This helm chart should be updated onco in a while.