var resOwner string
if r.ko.Status.OwningAccountID != nil && *r.ko.Status.OwningAccountID == string(rm.awsAccountID) {
	resOwner = "SELF"
} else {
	resOwner = "OTHER-ACCOUNTS"
}
input.ResourceOwner = &resOwner