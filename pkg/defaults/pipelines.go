package defaults

const (
	//PipelineNameAnnotation Original pipeline name, overridden by unique generated name
	PipelineNameAnnotation = "radix.equinor.com/tekton-pipeline-name"
	//DefaultPipelineFileName Default pipeline file name. It can be overridden in the radixconfig.yaml
	DefaultPipelineFileName = "tekton/pipeline.yaml"
)
