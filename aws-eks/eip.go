package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Eip struct {
	pulumi.ResourceState

	eipId pulumi.IDOutput `pulumi:"eipId"`
}

// Create an EIP
func NewEip(ctx *pulumi.Context, eipName string) (*Eip, error) {
	var resource Eip

	tags := NewTags(ctx)
	tags["Name"] = eipName
	
	eip, err := ec2.NewEip(ctx, eipName, &ec2.EipArgs{
		Vpc: pulumi.Bool(true),
		Tags:  pulumi.ToStringMap(tags),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.eipId = eip.ID()
	return &resource, nil
}