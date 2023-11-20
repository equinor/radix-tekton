![Build status](https://github.com/equinor/radix-tekton/actions/workflows/build-push.yml/badge.svg)  

# radix-tekton
Radix Tekton CD pipeline runner
# radix-tektonPrepare and process Radix application repository for pipeline job and sub-pipeline, when applicable.

* Run the CLI with a prepare pipeline action `prepare` as an argument `RADIX_PIPELINE_ACTION`:
  * Clones the Radix application repository
  * Reads the `radixconfig.yaml`, convert it to the `RadixApplication` to validate it, return it to the pipeline orchestration job
  * Reads the Tekton pipeline and tasks, if they exist. Create Tekton Pipeline and Tekton Task objects, filled by secrets and  environment variables corresponding the `radixconfig.yaml`, if applicable.
  * Analyses the source code for the `build and deploy` pipeline workflow to detect which component need to be built. Returns this information to the pipeline orchestration job
  * Gets corresponding GitHub `Commit ID` and `Tags` if applicable. Returns this information to the pipeline orchestration job
* Run the CLI with a prepare pipeline action `run` as an argument `RADIX_PIPELINE_ACTION`:
  * Runs the Tekton pipeline with tasks, if they have been created in the `prepare` action. 

## Contribution

Want to contribute? Read our [contributing guidelines](./CONTRIBUTING.md)

## Security

This is how we handle [security issues](./SECURITY.md)