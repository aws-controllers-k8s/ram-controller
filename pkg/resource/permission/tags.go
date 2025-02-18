// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package permission

import (
	"slices"
	"strings"

	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"

	svcapitypes "github.com/aws-controllers-k8s/ram-controller/apis/v1alpha1"
)

var (
	_             = svcapitypes.Permission{}
	_             = acktags.NewTags()
	ACKSystemTags = []string{"services.k8s.aws/namespace", "services.k8s.aws/controller-version"}
)

// ToACKTags converts the tags parameter into 'acktags.Tags' shape.
// This method helps in creating the hub(acktags.Tags) for merging
// default controller tags with existing resource tags.
func ToACKTags(tags []*svcapitypes.Tag) acktags.Tags {
	result := acktags.NewTags()
	if tags == nil || len(tags) == 0 {
		return result
	}

	for _, t := range tags {
		if t.Key != nil {
			if t.Value == nil {
				result[*t.Key] = ""
			} else {
				result[*t.Key] = *t.Value
			}
		}
	}

	return result
}

// FromACKTags converts the tags parameter into []*svcapitypes.Tag shape.
// This method helps in setting the tags back inside AWSResource after merging
// default controller tags with existing resource tags.
func FromACKTags(tags acktags.Tags) []*svcapitypes.Tag {
	result := []*svcapitypes.Tag{}
	for k, v := range tags {
		kCopy := k
		vCopy := v
		tag := svcapitypes.Tag{Key: &kCopy, Value: &vCopy}
		result = append(result, &tag)
	}
	return result
}

// ignoreSystemTags ignores tags that have keys that start with "aws:"
// and ACKSystemTags, to avoid patching them to the resourceSpec.
// Eg. resources created with cloudformation have tags that cannot be
// removed by an ACK controller
func ignoreSystemTags(tags acktags.Tags) {
	for k := range tags {
		if strings.HasPrefix(k, "aws:") ||
			slices.Contains(ACKSystemTags, k) {
			delete(tags, k)
		}
	}
}

// syncAWSTags ensures AWS-managed tags (prefixed with "aws:") from the latest resource state
// are preserved in the desired state. This prevents the controller from attempting to
// modify AWS-managed tags, which would result in an error.
//
// AWS-managed tags are automatically added by AWS services (e.g., CloudFormation, Service Catalog)
// and cannot be modified or deleted through normal tag operations. Common examples include:
// - aws:cloudformation:stack-name
// - aws:servicecatalog:productArn
//
// Parameters:
//   - a: The target Tags map to be updated (typically desired state)
//   - b: The source Tags map containing AWS-managed tags (typically latest state)
//
// Example:
//
//	latest := Tags{"aws:cloudformation:stack-name": "my-stack", "environment": "prod"}
//	desired := Tags{"environment": "dev"}
//	SyncAWSTags(desired, latest)
//	desired now contains {"aws:cloudformation:stack-name": "my-stack", "environment": "dev"}
func syncAWSTags(a acktags.Tags, b acktags.Tags) {
	for k := range b {
		if strings.HasPrefix(k, "aws:") {
			a[k] = b[k]
		}
	}
}
