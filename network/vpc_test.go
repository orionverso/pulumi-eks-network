package network

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestK8sVpc(t *testing.T) {

	clean := func(ctx context.Context, stackname string, stack auto.Stack) {

		_, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))

		if err != nil {
			fmt.Printf("Failed to resources: %v", err)
			fmt.Println("You must remove the reources manually")
			t.FailNow()
		}

		fmt.Println("Stack resources successfully destroyed")

		err = stack.Workspace().RemoveStack(ctx, stackname)

		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
			fmt.Println("You must remove the stack manually")
			t.FailNow()
		}

		fmt.Println("Stack successfully destroyed")
	}

	resources := func(ctx *pulumi.Context) error {
		args := K8sVpcArgs{}

		_, err := NewK8sVpc(ctx, "MyK8sVpcTesting", &args)

		if err != nil {
			t.FailNow()
		}
		return nil
	}

	ctx := context.Background()
	org := "orionverso"
	projectName := "Test"
	stackName := auto.FullyQualifiedStackName(org, projectName, "ephemeral")

	stack, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, resources)

	fmt.Printf("Created/Selected stack %q", stackName)

	w := stack.Workspace()

	defer clean(ctx, stackName, stack) // PASS or FAIL: Clean resources!!

	//Install dependecies
	fmt.Println("Installing the AWS plugin")

	err = w.InstallPlugin(ctx, "aws", "v5.0.0")
	if err != nil {
		fmt.Printf("Failed to install program plugins: %v", err)
		os.Exit(1)
	}

	fmt.Println("Successfully installed AWS plugin")

	//Set stack configuration
	stack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "us-east-1"})

	fmt.Println("Successfully set config")

	//Deploy Resources
	fmt.Println("Start stack up")

	_, err = stack.Up(ctx, optup.ProgressStreams(os.Stdout))

	if err != nil {
		fmt.Printf("Failed to deploy stack resources: %v", err)
		t.FailNow()
	}

	fmt.Println("Successfully stack resources deployment")

	/*
	   You can run some integration testing here with aws-sdk-go.
	   For example, put object to bucket and check
	*/

}
