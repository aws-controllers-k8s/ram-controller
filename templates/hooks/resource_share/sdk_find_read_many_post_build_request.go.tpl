
	// defaults owner to Self, only change if we know it's owned by another account
	input.ResourceOwner = svcsdktypes.ResourceOwnerSelf
	if r.ko.Status.OwningAccountID != nil && *r.ko.Status.OwningAccountID != string(rm.awsAccountID) {
		input.ResourceOwner = svcsdktypes.ResourceOwnerOtherAccounts
	}
    // only gets the ResourceShare identified by this ARN
    if r.ko.Status.ACKResourceMetadata != nil && r.ko.Status.ACKResourceMetadata.ARN != nil {
        input.ResourceShareArns = []string{string(*r.ko.Status.ACKResourceMetadata.ARN)}
    }