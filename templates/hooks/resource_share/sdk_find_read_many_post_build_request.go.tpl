	if r.ko.Status.OwningAccountID != nil && *r.ko.Status.OwningAccountID == string(rm.awsAccountID) {
		input.ResourceOwner = svcsdktypes.ResourceOwnerSelf
	} else {
		input.ResourceOwner = svcsdktypes.ResourceOwnerOtherAccounts
	}