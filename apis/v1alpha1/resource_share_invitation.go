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

package v1alpha1

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceShareAccepterSpec defines the desired state of ResourceShareAccepter.
//
// This resource accepts a RAM resource share invitation. When you create this
// resource, the controller finds the pending invitation for the specified ShareARN
// and accepts it. When you delete this resource, the controller disassociates
// from the resource share (leaves the share).
type ResourceShareAccepterSpec struct {
	// The Amazon Resource Name (ARN) of the resource share to accept.
	// This is immutable - you cannot change which share you're accepting after creation.
	// +kubebuilder:validation:Required
	ShareARN *string `json:"shareARN"`
}

// ResourceShareAccepterStatus defines the observed state of ResourceShareAccepter
type ResourceShareAccepterStatus struct {
	// All CRs managed by ACK have a common `Status.ACKResourceMetadata` member
	// that is used to contain resource sync state, account ownership,
	// constructed ARN for the resource
	// +kubebuilder:validation:Optional
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	// All CRs managed by ACK have a common `Status.Conditions` member that
	// contains a collection of `ackv1alpha1.Condition` objects that describe
	// the various terminal states of the CR and its backend AWS service API
	// resource
	// +kubebuilder:validation:Optional
	Conditions []*ackv1alpha1.Condition `json:"conditions"`
	// The ARN of the invitation that was accepted.
	// +kubebuilder:validation:Optional
	InvitationARN *string `json:"invitationARN,omitempty"`
	// The current status of the invitation.
	// Possible values: PENDING, ACCEPTED, REJECTED, EXPIRED
	// +kubebuilder:validation:Optional
	InvitationStatus *string `json:"invitationStatus,omitempty"`
	// The ID of the AWS account that sent the invitation.
	// +kubebuilder:validation:Optional
	SenderAccountID *string `json:"senderAccountID,omitempty"`
	// The ID of the AWS account that received the invitation.
	// +kubebuilder:validation:Optional
	ReceiverAccountID *string `json:"receiverAccountID,omitempty"`
	// The name of the resource share.
	// +kubebuilder:validation:Optional
	ShareName *string `json:"shareName,omitempty"`
	// The date and time when the invitation was sent.
	// +kubebuilder:validation:Optional
	InvitationTime *metav1.Time `json:"invitationTime,omitempty"`
	// The resources included in the resource share.
	// +kubebuilder:validation:Optional
	Resources []*string `json:"resources,omitempty"`
	// The current status of the resource share.
	// +kubebuilder:validation:Optional
	ShareStatus *string `json:"shareStatus,omitempty"`
}

// ResourceShareAccepter is the Schema for the ResourceShareAccepters API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SHARE-ARN",type=string,priority=0,JSONPath=`.spec.shareARN`
// +kubebuilder:printcolumn:name="STATUS",type=string,priority=0,JSONPath=`.status.invitationStatus`
// +kubebuilder:printcolumn:name="SENDER",type=string,priority=1,JSONPath=`.status.senderAccountID`
type ResourceShareAccepter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceShareAccepterSpec   `json:"spec,omitempty"`
	Status            ResourceShareAccepterStatus `json:"status,omitempty"`
}

// ResourceShareAccepterList contains a list of ResourceShareAccepter
// +kubebuilder:object:root=true
type ResourceShareAccepterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceShareAccepter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceShareAccepter{}, &ResourceShareAccepterList{})
}

