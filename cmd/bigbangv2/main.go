package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/defenseunicorns/pulumi-bigbang/pkg/api"
	v2 "github.com/defenseunicorns/pulumi-bigbang/pkg/corev2"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {

	//try and read the state from the BigBang deployment

	project := flag.String("project", "bigbang", "project name")
	stack := flag.String("stack", "binary", "Stack name")
	// namespace := flag.String("namespace", "", "Namespace to create")
	destroy := flag.Bool("destroy", false, "cleanup namespace")
	debug := flag.Bool("debug", false, "debug")
	configFile := flag.String("config", "./config.yaml", "Config File ")
	// chart := flag.String("chart", "wordpress", "Chart to deploy")
	// name := flag.String("name", "", "Name for helm install")
	// file := flag.String("file", "", "helm values file")

	flag.Parse()

	config, err := api.LoadConfiguration(*configFile)

	if err != nil {
		panic(err)
	}

	if *debug {
		b, _ := json.MarshalIndent(config, "", "\t")
		fmt.Printf("Config:\n%s\n", string(b))
		os.Exit(0)
	}

	deployFunc := func(ctx *pulumi.Context) error {
		// conf := config.New(ctx, "")

		// ctx.GetConfig()

		_, err := v2.DeployBigBang(ctx, *config)

		return err

	}

	ctx := context.Background()

	// stackName := auto.FullyQualifiedStackName("P", projectName, stackName)
	// create or select a stack matching the specified name and project.
	// this will set up a workspace with everything necessary to run our inline program (deployFunc)
	s, err := auto.UpsertStackInlineSource(ctx, *stack, *project, deployFunc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created/Selected stack %v\n", s.Name())
	summary, err := s.Info(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created/Selected project %v\n", summary)

	w := s.Workspace()

	err = w.InstallPlugin(ctx, "kubernetes", "v3.20.5")

	// // set stack configuration specifying the AWS region to deploy
	// s.SetConfig(ctx, "registry.username", auto.ConfigValue{Value: *username})
	// s.SetConfig(ctx, "registry.password", auto.ConfigValue{Value: *password})
	// s.SetConfig(ctx, "registry.registry", auto.ConfigValue{Value: "registry1.dso.mil"})
	// s.SetConfig(ctx, "keyFile", auto.ConfigValue{Value: "./public.key"})
	// s.SetConfig(ctx, "certFile", auto.ConfigValue{Value: "./public.cert"})

	refresh, err := s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("StdOut:")
	fmt.Println(refresh.StdOut)
	fmt.Println("\n\nStdErr")
	fmt.Println(refresh.StdErr)
	// refresh.

	fmt.Println("Refresh succeeded!")

	if *destroy {
		fmt.Println("Starting stack destroy")

		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
		}
		fmt.Println("Stack successfully destroyed")
		os.Exit(0)
	}

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	// run the update to deploy our s3 website
	res, err := s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Update succeeded!")

	b, _ := json.MarshalIndent(res, "", "\t")
	fmt.Println(string(b))

}
