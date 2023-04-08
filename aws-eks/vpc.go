package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Vpc struct {
	pulumi.ResourceState

	vpcId pulumi.IDOutput `pulumi:"vpcId"`
}

func NewVpc(ctx *pulumi.Context, vpcName string, cidr string) (*Vpc, error) {
	var resource Vpc

	tags := NewTags(ctx)
	tags["Name"] = vpcName

	vpcArgs := &ec2.VpcArgs{
		CidrBlock:          pulumi.String(cidr),
		EnableDnsHostnames: pulumi.Bool(true),
		InstanceTenancy:    pulumi.String("default"),
		Tags:               pulumi.ToStringMap(tags),
	}

	vpc, err := ec2.NewVpc(ctx, vpcName, vpcArgs, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	resource.vpcId = vpc.ID()
	return &resource, nil
}