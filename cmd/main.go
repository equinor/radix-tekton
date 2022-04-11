package main

import (
    "github.com/equinor/radix-tekton/pkg/kubernetes"
    "github.com/equinor/radix-tekton/pkg/pipeline"
    log "github.com/sirupsen/logrus"
    "github.com/spf13/cobra"
)

var (
    // Command line flags
    namespace     string
    configMapName string
    file          string
    pipelineName  string
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "run-tekton",
        Short: "Radix Tekton pipeline",
        Run:   root}

    pf := rootCmd.PersistentFlags()

    pf.StringVar(&namespace, "namespace", "",
        "namespace configmap and Tekton pipeline should be applied to")
    pf.StringVar(&configMapName, "configmap-name", "",
        "name to give to configmap")
    pf.StringVar(&file, "file", "",
        "absolute path to file")
    pf.StringVar(&configMapName, "pipeline", "",
        "name to give to Tekton pipeline-name")

    cobra.MarkFlagRequired(pf, "namespace")
    cobra.MarkFlagRequired(pf, "configmap-name")
    cobra.MarkFlagRequired(pf, "file")
    cobra.MarkFlagRequired(pf, "pipeline-name")

    if err := rootCmd.Execute(); err != nil {
        log.Fatal(err)
    }
}

func root(cmd *cobra.Command, args []string) {
    kubeClient, radixClient, err := kubernetes.GetClients()
    if err != nil {
        log.Fatal(err.Error())
    }

    pipelineContext := pipeline.NewPipelineContext(kubeClient, radixClient, namespace, configMapName, file,
        pipelineName)
    _, err = pipelineContext.ProcessRadixConfig()
    if err != nil {
        log.Fatal(err.Error())
    }

    log.Infof("Successfully completed")
}
