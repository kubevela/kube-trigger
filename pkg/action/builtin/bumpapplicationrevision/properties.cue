// TODO(charlie0129): make these markers work

//+type=bump-application-revision
//+description=TODO

//+usage=Only bump Applications in this namespace. Leave empty to select all namespaces.
namespace: *"" | string
//+usage=Only bump Applications with this name.
name: *"" | string
//+usage=Only bump Applications with these labels.
labelSelectors?: [string]: string
