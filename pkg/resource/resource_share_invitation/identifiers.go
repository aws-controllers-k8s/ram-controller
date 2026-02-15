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

package resource_share_invitation

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
)

// resourceIdentifiers implements the AWSResourceIdentifiers interface
type resourceIdentifiers struct {
	r *resource
}

// ARN returns the AWS Resource Name for the resource
// For ResourceShareInvitation, we use the ShareARN as the primary identifier
func (ri *resourceIdentifiers) ARN() *ackv1alpha1.AWSResourceName {
	if ri.r.ko.Spec.ShareARN == nil {
		return nil
	}
	arn := ackv1alpha1.AWSResourceName(*ri.r.ko.Spec.ShareARN)
	return &arn
}

// OwnerAccountID returns the AWS account identifier for the owner of the resource
func (ri *resourceIdentifiers) OwnerAccountID() *ackv1alpha1.AWSAccountID {
	if ri.r.ko.Status.ACKResourceMetadata == nil {
		return nil
	}
	return ri.r.ko.Status.ACKResourceMetadata.OwnerAccountID
}

// Region returns the AWS region the resource exists in
func (ri *resourceIdentifiers) Region() *ackv1alpha1.AWSRegion {
	if ri.r.ko.Status.ACKResourceMetadata == nil {
		return nil
	}
	return ri.r.ko.Status.ACKResourceMetadata.Region
}

// NameOrID returns a string that can be used to uniquely identify the resource
// For ResourceShareInvitation, we use the ShareARN
func (ri *resourceIdentifiers) NameOrID() string {
	if ri.r.ko.Spec.ShareARN == nil {
		return ""
	}
	return *ri.r.ko.Spec.ShareARN
}

