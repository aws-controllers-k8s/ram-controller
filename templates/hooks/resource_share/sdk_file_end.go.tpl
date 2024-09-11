{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for ResourceShare */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.ResourceShare.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $resourceShareRefName, $resourceShareMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $resourceShareRefName "Tags" }}
{{- $resourceShareRef := $resourceShareMemberRefs.Shape.MemberRef }}
{{- $resourceShareRefName = "Tag" }}
func (rm *resourceManager) new{{ $resourceShareRefName }}(
	    c svcapitypes.{{ $resourceShareRefName }},
) *svcsdk.{{ $resourceShareRefName }} {
	res := &svcsdk.{{ $resourceShareRefName }}{}
  {{ GoCodeSetSDKForStruct $CRD "" "res" $resourceShareRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}