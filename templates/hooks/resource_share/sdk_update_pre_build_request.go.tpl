	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	// Resource associations sync first so that dropping a permission and the
	// resources it covers in the same edit works: RAM refuses to disassociate
	// the last permission covering a resource type still in the share.
	if delta.DifferentAt("Spec.ResourceARNs") || delta.DifferentAt("Spec.Principals") || delta.DifferentAt("Spec.Sources") {
		if err := rm.syncResourceShareResources(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.PermissionARNs") {
		if err := rm.syncPermissions(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if !delta.DifferentExcept("Spec.Tags", "Spec.PermissionARNs", "Spec.ResourceARNs", "Spec.Principals", "Spec.Sources") {
		return desired, nil
	}
