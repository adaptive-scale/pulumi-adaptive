package adaptive

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Variable is an encrypted, workspace-scoped value referenced by Adaptive
// resource configurations with Mustache syntax such as {{database_host}}.
type Variable struct{}

type VariableArgs struct {
	Name   string   `pulumi:"name" provider:"replaceOnChanges"`
	Value  string   `pulumi:"value" provider:"secret"`
	Labels []string `pulumi:"labels,optional"`
}

type VariableState struct {
	VariableArgs
}

func (v *VariableArgs) Annotate(a infer.Annotator) {
	a.Describe(&v.Name, "Workspace-unique variable name. Use it in resource configuration as {{name}}. Changing it replaces the variable.")
	a.Describe(&v.Value, "Secret or configuration value substituted into Adaptive resource configurations at runtime.")
	a.Describe(&v.Labels, "Permission labels attached to the variable.")
}

func (a VariableArgs) request() VariableRequest {
	return VariableRequest{Name: a.Name, Value: a.Value, Labels: a.Labels}
}

func applyVariableRead(prior VariableArgs, read *VariableReadResponse) VariableArgs {
	return VariableArgs{
		Name: read.Name, Value: read.Value, Labels: setList(prior.Labels, read.Labels),
	}
}

func (*Variable) Create(ctx context.Context, req infer.CreateRequest[VariableArgs]) (infer.CreateResponse[VariableState], error) {
	out := infer.CreateResponse[VariableState]{Output: VariableState{VariableArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	created, err := c.CreateVariable(ctx, req.Inputs.request())
	if err != nil {
		return out, err
	}
	out.ID = created.ID
	out.Output.VariableArgs = applyVariableRead(req.Inputs, created)
	return out, nil
}

func (*Variable) Update(ctx context.Context, req infer.UpdateRequest[VariableArgs, VariableState]) (infer.UpdateResponse[VariableState], error) {
	out := infer.UpdateResponse[VariableState]{Output: VariableState{VariableArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	updated, err := c.UpdateVariable(ctx, req.ID, req.Inputs.request())
	if err != nil {
		return out, err
	}
	out.Output.VariableArgs = applyVariableRead(req.Inputs, updated)
	return out, nil
}

func (*Variable) Read(ctx context.Context, req infer.ReadRequest[VariableArgs, VariableState]) (infer.ReadResponse[VariableArgs, VariableState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[VariableArgs, VariableState]{}, err
	}
	read, err := c.ReadVariable(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[VariableArgs, VariableState]{}, err
	}
	if read == nil {
		return infer.ReadResponse[VariableArgs, VariableState]{}, nil
	}
	inputs := applyVariableRead(req.Inputs, read)
	return infer.ReadResponse[VariableArgs, VariableState]{
		ID: req.ID, Inputs: inputs, State: VariableState{VariableArgs: inputs},
	}, nil
}

func (*Variable) Delete(ctx context.Context, req infer.DeleteRequest[VariableState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteVariable(ctx, req.ID)
}
