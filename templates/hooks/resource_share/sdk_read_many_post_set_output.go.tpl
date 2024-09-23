	if err = rm.getPermissionArns(ctx, &resource{ko}); err != nil {
		return nil, err
	}
	
	if err = rm.getResourceShareAssociations(ctx, &resource{ko}); err != nil {
		return nil, err
	}
