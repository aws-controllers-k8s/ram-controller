
	// Once an invitation has been accepted, AWS purges it from
	// GetResourceShareInvitations after a retention window (7 days for most
	// resource types, 12 hours for some). After that this call returns an empty
	// list even though this account is still associated with the share. If we
	// fell through to the generated "not found" path the runtime would attempt
	// to re-Accept and fail terminally. So when the invitation has disappeared
	// but we previously recorded the share ARN, confirm the association still
	// exists via GetResourceShares and report the resource as still ACCEPTED.
	if err == nil && len(resp.ResourceShareInvitations) == 0 && r.ko.Status.ResourceShareARN != nil {
		shareResp, shareErr := rm.sdkapi.GetResourceShares(ctx, &svcsdk.GetResourceSharesInput{
			ResourceOwner:     "OTHER-ACCOUNTS",
			ResourceShareArns: []string{*r.ko.Status.ResourceShareARN},
		})
		rm.metrics.RecordAPICall("READ_MANY", "GetResourceShares", shareErr)
		if shareErr == nil {
			for _, share := range shareResp.ResourceShares {
				if share.Status == "ACTIVE" {
					rm.metrics.RecordAPICall("READ_MANY", "GetResourceShareInvitations", err)
					ko := r.ko.DeepCopy()
					ko.Status.Status = aws.String("ACCEPTED")
					rm.setStatusDefaults(ko)
					return &resource{ko}, nil
				}
			}
		}
	}
