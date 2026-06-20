	// A PENDING invitation has not yet been accepted by this account, so from
	// ACK's perspective the resource does not exist. Returning NotFound here
	// causes the runtime to invoke sdkCreate, which maps to
	// AcceptResourceShareInvitation.
	if ko.Status.Status != nil && *ko.Status.Status == "PENDING" {
		return nil, ackerr.NotFound
	}
