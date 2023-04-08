package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewTags(ctx *pulumi.Context) map[string]string {
	tags := make(map[string]string)
	tags["region"] = getEnv(ctx, "aws:region", "unknown")
	tags["version"] = getEnv(ctx, "tags:version", "unknown")
	tags["author"] = getEnv(ctx, "tags:author", "unknown")

	return tags
}