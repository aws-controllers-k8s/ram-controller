	// The invitation ARN is supplied in the Spec by the user. Because the field
	// is marked is_primary_key, the generator otherwise reads it from
	// Status.ACKResourceMetadata.ARN, which is not yet populated on first Create.
	if desired.ko.Spec.ResourceShareInvitationARN != nil {
		input.ResourceShareInvitationArn = desired.ko.Spec.ResourceShareInvitationARN
	}
