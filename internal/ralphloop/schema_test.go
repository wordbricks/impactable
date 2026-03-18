package ralphloop

import "testing"

func TestCommandDescriptors_TailAliasesPresent(t *testing.T) {
	t.Parallel()

	descriptor, ok := commandDescriptorByName(commandTail)
	if !ok {
		t.Fatalf("expected %q descriptor", commandTail)
	}

	aliasesByName := map[string][]string{}
	for _, option := range descriptor.Options {
		aliasesByName[option.Name] = option.Aliases
	}

	if len(aliasesByName["--lines"]) != 1 || aliasesByName["--lines"][0] != "-n" {
		t.Fatalf("expected --lines alias -n, got %#v", aliasesByName["--lines"])
	}
	if len(aliasesByName["--follow"]) != 1 || aliasesByName["--follow"][0] != "-f" {
		t.Fatalf("expected --follow alias -f, got %#v", aliasesByName["--follow"])
	}
}

func TestCommandDescriptors_SchemaPayloadShape(t *testing.T) {
	t.Parallel()

	descriptor, ok := commandDescriptorByName(commandSchema)
	if !ok {
		t.Fatalf("expected %q descriptor", commandSchema)
	}

	commandProp, ok := descriptor.RawPayloadSchema.Properties["command"]
	if !ok {
		t.Fatalf("expected schema payload to include command discriminator")
	}
	if commandProp.Default != commandSchema {
		t.Fatalf("expected command default %q, got %#v", commandSchema, commandProp.Default)
	}
	if _, ok := descriptor.RawPayloadSchema.Properties["target_command"]; !ok {
		t.Fatalf("expected schema payload to include target_command")
	}
}

func TestCommandOptionSet_IsCommandSpecific(t *testing.T) {
	t.Parallel()

	listOptions := commandOptionSet(commandList)
	if _, ok := listOptions["--output"]; !ok {
		t.Fatalf("expected ls to support --output")
	}
	if _, ok := listOptions["--max-iterations"]; ok {
		t.Fatalf("did not expect ls to support --max-iterations")
	}

	tailOptions := commandOptionSet(commandTail)
	if _, ok := tailOptions["-n"]; !ok {
		t.Fatalf("expected tail to support -n alias")
	}
}
