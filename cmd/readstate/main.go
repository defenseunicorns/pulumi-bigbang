package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

func main() {

	//try and read the state from the BigBang deployment

	project := flag.String("project", "bigbang", "project name")
	stack := flag.String("stack", "auto", "Stack name")

	flag.Parse()

	ctx := context.Background()

	// create or select a stack matching the specified name and project.
	// this will set up a workspace with everything necessary to run our inline program (deployFunc)
	s, err := auto.UpsertStackInlineSource(ctx, *stack, *project, nil)

	if err != nil {
		panic(err)
	}
	fmt.Printf("Created/Selected stack %q\n", *stack)
	outputs, err := s.Outputs(ctx)
	fmt.Println("Found outputs for stack:")
	for k, v := range outputs {
		fmt.Printf("------------\n%v\n", k)
		fmt.Printf("IsSecret?   %v\n", v.Secret)
		fmt.Printf("Value:      %v\n", v.Value)
	}

}
