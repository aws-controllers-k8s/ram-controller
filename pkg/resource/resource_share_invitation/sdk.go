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
	corev1 "k8s.io/api/core/v1"
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
	rm.setStatusDefaults(ko)
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
		return nil, ackerr.NewTerminalError(fmt.Errorf("no pending invitation found for share ARN %s. "+
			"NOTE: If both AWS accounts are in the same AWS Organization and RAM Sharing with AWS Organizations is enabled, "+
			"this resource is not necessary", *desired.ko.Spec.ShareARN))
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
	rm.setStatusDefaults(ko)
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

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.ResourceShareAccepter,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

// updateConditions returns updated resource, true; if conditions were updated
// else it returns nil, false
func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	// Terminal condition
	var terminalCondition *ackv1alpha1.Condition = nil
	var recoverableCondition *ackv1alpha1.Condition = nil
	var syncCondition *ackv1alpha1.Condition = nil
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}
	var termError *ackerr.TerminalError
	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
			errorMessage = err.Error()
		} else {
			awsErr, _ := ackerr.AWSError(err)
			errorMessage = awsErr.Error()
		}
		terminalCondition.Status = corev1.ConditionTrue
		terminalCondition.Message = &errorMessage
	} else {
		// Clear the terminal condition if no longer present
		if terminalCondition != nil {
			terminalCondition.Status = corev1.ConditionFalse
			terminalCondition.Message = nil
		}
		// Handling Recoverable Conditions
		if err != nil {
			if recoverableCondition == nil {
				// Add a new Condition containing a non-terminal error
				recoverableCondition = &ackv1alpha1.Condition{
					Type: ackv1alpha1.ConditionTypeRecoverable,
				}
				ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
			}
			recoverableCondition.Status = corev1.ConditionTrue
			awsErr, _ := ackerr.AWSError(err)
			errorMessage := err.Error()
			if awsErr != nil {
				errorMessage = awsErr.Error()
			}
			recoverableCondition.Message = &errorMessage
		} else if recoverableCondition != nil {
			recoverableCondition.Status = corev1.ConditionFalse
			recoverableCondition.Message = nil
		}
	}
	// Required to avoid the "declared but not used" error in the default case
	_ = syncCondition
	if terminalCondition != nil || recoverableCondition != nil || syncCondition != nil {
		return &resource{ko}, true // updated
	}
	return nil, false // not updated
}

// terminalAWSError returns awserr, true; if the supplied error is an aws Error type
// and if the exception indicates that it is a Terminal exception
// 'Terminal' exception are specified in generator configuration
func (rm *resourceManager) terminalAWSError(err error) bool {
	if err == nil {
		return false
	}

	var terminalErr smithy.APIError
	if !errors.As(err, &terminalErr) {
		return false
	}
	switch terminalErr.ErrorCode() {
	case "MalformedArnException":
		return true
	default:
		return false
	}
}
