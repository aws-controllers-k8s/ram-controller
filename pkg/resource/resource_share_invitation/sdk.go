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
	"context"
	"errors"
	"fmt"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ram"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ram/types"
	smithy "github.com/aws/smithy-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/ram-controller/apis/v1alpha1"
)

// sdkFind returns SDK-specific information about a supplied resource
// For ResourceShareInvitation, this finds the invitation by ShareARN
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer func() {
		exit(err)
	}()

	if r.ko.Spec.ShareARN == nil {
		return nil, ackerr.NotFound
	}

	// Get the invitation for this share ARN
	input := &svcsdk.GetResourceShareInvitationsInput{
		ResourceShareArns: []string{*r.ko.Spec.ShareARN},
	}

	var resp *svcsdk.GetResourceShareInvitationsOutput
	resp, err = rm.sdkapi.GetResourceShareInvitations(ctx, input)
	rm.metrics.RecordAPICall("READ_ONE", "GetResourceShareInvitations", err)
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) {
			if awsErr.ErrorCode() == "ResourceShareInvitationArnNotFoundException" ||
				awsErr.ErrorCode() == "UnknownResourceException" {
				return nil, ackerr.NotFound
			}
		}
		return nil, err
	}

	// Find the accepted invitation for this share
	ko := r.ko.DeepCopy()
	found := false

	for _, invitation := range resp.ResourceShareInvitations {
		// We're looking for an ACCEPTED invitation for this share
		if invitation.Status == svcsdktypes.ResourceShareInvitationStatusAccepted {
			found = true
			rm.setResourceFromInvitation(ko, &invitation)
			break
		}
	}

	if !found {
		return nil, ackerr.NotFound
	}

	return &resource{ko}, nil
}

// sdkCreate creates the supplied resource in the backend AWS service API
// For ResourceShareInvitation, this means finding a pending invitation and accepting it
func (rm *resourceManager) sdkCreate(
	ctx context.Context,
	desired *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkCreate")
	defer func() {
		exit(err)
	}()

	if desired.ko.Spec.ShareARN == nil {
		return nil, fmt.Errorf("ShareARN is required")
	}

	// First, find the pending invitation for this share ARN
	getInput := &svcsdk.GetResourceShareInvitationsInput{
		ResourceShareArns: []string{*desired.ko.Spec.ShareARN},
	}

	var getResp *svcsdk.GetResourceShareInvitationsOutput
	getResp, err = rm.sdkapi.GetResourceShareInvitations(ctx, getInput)
	rm.metrics.RecordAPICall("READ_ONE", "GetResourceShareInvitations", err)
	if err != nil {
		return nil, err
	}

	// Find a pending invitation
	var pendingInvitationARN *string
	for _, invitation := range getResp.ResourceShareInvitations {
		if invitation.Status == svcsdktypes.ResourceShareInvitationStatusPending {
			pendingInvitationARN = invitation.ResourceShareInvitationArn
			break
		}
	}

	if pendingInvitationARN == nil {
		return nil, fmt.Errorf("no pending invitation found for share ARN %s. "+
			"NOTE: If both AWS accounts are in the same AWS Organization and RAM Sharing with AWS Organizations is enabled, "+
			"this resource is not necessary", *desired.ko.Spec.ShareARN)
	}

	// Accept the invitation
	acceptInput := &svcsdk.AcceptResourceShareInvitationInput{
		ResourceShareInvitationArn: pendingInvitationARN,
	}

	var acceptResp *svcsdk.AcceptResourceShareInvitationOutput
	acceptResp, err = rm.sdkapi.AcceptResourceShareInvitation(ctx, acceptInput)
	rm.metrics.RecordAPICall("CREATE", "AcceptResourceShareInvitation", err)
	if err != nil {
		return nil, err
	}

	ko := desired.ko.DeepCopy()
	rm.setResourceFromInvitation(ko, acceptResp.ResourceShareInvitation)

	return &resource{ko}, nil
}



// sdkUpdate is not supported for ResourceShareAccepter
// Invitations are immutable once accepted
func (rm *resourceManager) sdkUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkUpdate")
	defer func() {
		exit(err)
	}()

	// ResourceShareInvitations cannot be updated
	// Return the latest resource as-is
	return latest, nil
}

// sdkDelete deletes the supplied resource in the backend AWS service API
// For ResourceShareInvitation, this means disassociating from the resource share (leaving it)
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()

	if r.ko.Spec.ShareARN == nil {
		return fmt.Errorf("ShareARN is required for deletion")
	}

	if r.ko.Status.ReceiverAccountID == nil {
		return fmt.Errorf("ReceiverAccountID is required for deletion")
	}

	// Disassociate from the resource share (leave it)
	input := &svcsdk.DisassociateResourceShareInput{
		ResourceShareArn: r.ko.Spec.ShareARN,
		Principals:       []string{*r.ko.Status.ReceiverAccountID},
	}

	_, err = rm.sdkapi.DisassociateResourceShare(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DisassociateResourceShare", err)
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) {
			// If the resource is already gone, that's fine
			if awsErr.ErrorCode() == "UnknownResourceException" {
				return nil
			}
			// If we're not permitted to disassociate, log but don't fail
			if awsErr.ErrorCode() == "OperationNotPermittedException" {
				rlog.Info("Resource share could not be disassociated, but continuing", "error", err)
				return nil
			}
		}
		return err
	}

	return nil
}

// setResourceFromInvitation sets the resource fields from an invitation object
func (rm *resourceManager) setResourceFromInvitation(
	ko *svcapitypes.ResourceShareAccepter,
	invitation *svcsdktypes.ResourceShareInvitation,
) {
	if invitation.ResourceShareInvitationArn != nil {
		ko.Status.InvitationARN = invitation.ResourceShareInvitationArn
	}
	if invitation.Status != "" {
		ko.Status.InvitationStatus = aws.String(string(invitation.Status))
	}
	if invitation.SenderAccountId != nil {
		ko.Status.SenderAccountID = invitation.SenderAccountId
	}
	if invitation.ReceiverAccountId != nil {
		ko.Status.ReceiverAccountID = invitation.ReceiverAccountId
	}
	if invitation.ResourceShareName != nil {
		ko.Status.ShareName = invitation.ResourceShareName
	}
	if invitation.InvitationTimestamp != nil {
		ko.Status.InvitationTime = &metav1.Time{*invitation.InvitationTimestamp}
	}
	if invitation.ResourceShareArn != nil {
		if ko.Status.ACKResourceMetadata == nil {
			ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
		}
		tmpARN := ackv1alpha1.AWSResourceName(*invitation.ResourceShareArn)
		ko.Status.ACKResourceMetadata.ARN = &tmpARN
	}
}
