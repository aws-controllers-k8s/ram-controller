
	if r.ko.Spec.ResourceShareInvitationARN != nil {
		input.ResourceShareInvitationArns = []string{*r.ko.Spec.ResourceShareInvitationARN}
	}
