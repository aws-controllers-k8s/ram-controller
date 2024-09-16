if resp.Permission.Permission != nil {
  ko.Spec.PolicyTemplate = resp.Permission.Permission
}
