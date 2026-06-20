
	if r.ko.Status.ReceiverAccountID == nil {
		return nil, ackerr.NotFound
	}
	input.Principals = []string{*r.ko.Status.ReceiverAccountID}
