package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/kms"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Kms struct {
	pulumi.ResourceState

	kmsKeyArn pulumi.StringOutput `pulumi:"kmsKeyArn"`
	kmsKeyId  pulumi.StringOutput `pulumi:"kmsKeyId"`
}

// Create an EIP
func NewKmsKey(ctx *pulumi.Context, keyName string) (*Kms, error) {
	var resource Kms

	tags := NewTags(ctx)
	tags["Name"] = keyName

	key, err := kms.NewKey(ctx, keyName, &kms.KeyArgs{
		DeletionWindowInDays: pulumi.Int(7),
		Description:          pulumi.String(keyName),
		EnableKeyRotation:    pulumi.Bool(true),
		Tags:                 pulumi.ToStringMap(tags),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.kmsKeyArn = key.Arn
	resource.kmsKeyId = key.KeyId
	return &resource, nil
}
