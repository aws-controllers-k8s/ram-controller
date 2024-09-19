
	if err = rm.getPermissionArns(ctx, &resource{ko}); err != nil {
		return nil, err
	}
