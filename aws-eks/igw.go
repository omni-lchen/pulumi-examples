package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Igw struct {
	pulumi.ResourceState

	igwId pulumi.IDOutput `pulumi:"igwId"`
}

// Create an Internet gateway
func NewIgw(ctx *pulumi.Context, vpcId pulumi.IDOutput) (*Igw, error) {
	var resource Igw

	tags := NewTags(ctx)
	igwName := NewPrefixName("igw")
	tags["Name"] = igwName
	igw, err := ec2.NewInternetGateway(ctx, igwName, &ec2.InternetGatewayArgs{
		VpcId: vpcId,
		Tags:  pulumi.ToStringMap(tags),
	},  pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	resource.igwId = igw.ID()

	return &resource, nil
}