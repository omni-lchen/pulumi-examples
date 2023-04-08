package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Current struct {
	pulumi.ResourceState

	accountId pulumi.String `pulumi:"accountId"`
}

// Get caller identity
func GetCurrent(ctx *pulumi.Context) (*Current, error) {
	var resource Current

	current, err := aws.GetCallerIdentity(ctx, nil, nil, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.accountId = pulumi.String(current.AccountId)
	return &resource, nil
}