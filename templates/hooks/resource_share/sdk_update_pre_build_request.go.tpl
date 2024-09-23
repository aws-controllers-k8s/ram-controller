	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.PermissionARNs") {
		if err := rm.syncPermissions(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.ResourceARNs") || delta.DifferentAt("Spec.Principals") || delta.DifferentAt("Spec.Sources") {
		if err := rm.syncResources(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if !delta.DifferentExcept("Spec.Tags", "Spec.PermissionARNs", "Spec.ResourceARNs", "Spec.Principals", "Spec.Sources") {
		return desired, nil
	}
