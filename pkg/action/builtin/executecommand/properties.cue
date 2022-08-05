// This is a validator for properties of execute-command

cmd:  string
args: *[] | [...string]
pwd:  *"" | string
env?: [string]: string
stdin: *"" | string

// Dynamically build the options above using contexts.
// Available contexts:
// - context.event: event meta
// - context.data: event data
// Will override the options above.
dynamicBuilder: *"" | string
