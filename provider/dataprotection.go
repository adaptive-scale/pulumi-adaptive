package adaptive

import (
	"context"
	"fmt"
	"sort"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ===========================================================================
// DataProtection  (adaptive:index:DataProtection) — masking policy
// ===========================================================================

type DataProtection struct{}

type DataProtectionArgs struct {
	Resource string               `pulumi:"resource"`
	Scoped   *bool                `pulumi:"scoped,optional"`
	Masks    []DataProtectionMask `pulumi:"masks,optional"`
}

type DataProtectionState struct {
	DataProtectionArgs
	AuthorizationName string `pulumi:"authorizationName,optional"`
	Status            string `pulumi:"status,optional"`
	LastAppliedAt     string `pulumi:"lastAppliedAt,optional"`
}

func (d *DataProtectionArgs) Annotate(a infer.Annotator) {
	a.Describe(&d.Resource, "Name of the resource (integration) this masking policy protects. "+
		"Cannot be changed after creation. Note: creating a policy for a resource that already has "+
		"one (e.g. configured in the UI) adopts and overwrites it.")
	a.Describe(&d.Scoped, "When true (the default), session users only see the generated masked views. "+
		"When false, users see everything except the raw masked tables (requires synced metadata).")
	a.Describe(&d.Masks, "Masking rules, grouped per database and table.")
}

func (d *DataProtectionState) Annotate(a infer.Annotator) {
	a.Describe(&d.AuthorizationName, "Name of the generated masking authorization (masked_<resource>). "+
		"Attach it to an Endpoint's `authorization` to serve masked sessions.")
	a.Describe(&d.Status, "Provisioning status of the policy. Masked views are created lazily at session start.")
	a.Describe(&d.LastAppliedAt, "Last time the policy was applied to the target database.")
}

func (a DataProtectionArgs) toDataProtectionRequest() DataProtectionRequest {
	return DataProtectionRequest{
		ResourceName: a.Resource,
		Scoped:       a.Scoped,
		Masks:        a.Masks,
	}
}

// fillDataProtectionOutputs copies server-computed fields onto the state.
func fillDataProtectionOutputs(s *DataProtectionState, r *DataProtectionReadResponse) {
	if r == nil {
		return
	}
	s.AuthorizationName = r.AuthorizationName
	s.Status = r.Status
	s.LastAppliedAt = r.LastAppliedAt
}

func (*DataProtection) Create(ctx context.Context, req infer.CreateRequest[DataProtectionArgs]) (infer.CreateResponse[DataProtectionState], error) {
	out := infer.CreateResponse[DataProtectionState]{Output: DataProtectionState{DataProtectionArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	resp, err := c.CreateDataProtection(ctx, req.Inputs.toDataProtectionRequest())
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	fillDataProtectionOutputs(&out.Output, resp)
	return out, nil
}

func (*DataProtection) Update(ctx context.Context, req infer.UpdateRequest[DataProtectionArgs, DataProtectionState]) (infer.UpdateResponse[DataProtectionState], error) {
	out := infer.UpdateResponse[DataProtectionState]{Output: DataProtectionState{
		DataProtectionArgs: req.Inputs,
		AuthorizationName:  req.State.AuthorizationName,
		Status:             req.State.Status,
		LastAppliedAt:      req.State.LastAppliedAt,
	}}
	if req.DryRun {
		return out, nil
	}
	// The policy is bound to its resource; moving it means a new policy.
	if req.Inputs.Resource != req.State.Resource {
		return out, fmt.Errorf("resource cannot be updated for an existing data protection policy")
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	resp, err := c.UpdateDataProtection(ctx, req.ID, req.Inputs.toDataProtectionRequest())
	if err != nil {
		return out, err
	}
	fillDataProtectionOutputs(&out.Output, resp)
	return out, nil
}

func (*DataProtection) Read(ctx context.Context, req infer.ReadRequest[DataProtectionArgs, DataProtectionState]) (infer.ReadResponse[DataProtectionArgs, DataProtectionState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[DataProtectionArgs, DataProtectionState]{}, err
	}
	r, err := c.ReadDataProtection(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[DataProtectionArgs, DataProtectionState]{}, err
	}
	isImport := req.Inputs.Resource == ""
	if r == nil {
		if isImport {
			return infer.ReadResponse[DataProtectionArgs, DataProtectionState]{}, notFoundOnImport("data protection policy", req.ID)
		}
		// Deleted out-of-band: an empty response drops the resource from state.
		return infer.ReadResponse[DataProtectionArgs, DataProtectionState]{}, nil
	}
	// A 200 carrying no identity is not a real record — a server that answers
	// unknown ids with a zero-valued body would otherwise fabricate a resource
	// out of nothing. Only enforced on import, where there is no prior state to
	// lose.
	if isImport && r.ResourceName == "" {
		return infer.ReadResponse[DataProtectionArgs, DataProtectionState]{}, notFoundOnImport("data protection policy", req.ID)
	}
	inputs := applyDataProtectionRead(req.Inputs, r, isImport)
	state := DataProtectionState{DataProtectionArgs: inputs}
	fillDataProtectionOutputs(&state, r)
	return infer.ReadResponse[DataProtectionArgs, DataProtectionState]{ID: req.ID, Inputs: inputs, State: state}, nil
}

// Delete turns masking off (the server applies scoped:false, masks:[] — the
// admin CLI's documented "off" shape). The masked_<resource> authorization and
// any already-provisioned masked views are left in place.
func (*DataProtection) Delete(ctx context.Context, req infer.DeleteRequest[DataProtectionState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteDataProtection(ctx, req.ID)
}

// applyDataProtectionRead reconciles inputs against a policy read.
func applyDataProtectionRead(prior DataProtectionArgs, r *DataProtectionReadResponse, isImport bool) DataProtectionArgs {
	a := prior
	if isImport {
		a = DataProtectionArgs{}
	}
	a.Resource = r.ResourceName

	// scoped defaults to true server-side: an unset input reading back true is
	// not drift, and false must always surface.
	serverScoped := true
	if r.Scoped != nil {
		serverScoped = *r.Scoped
	}
	if isImport {
		if !serverScoped {
			f := false
			a.Scoped = &f
		}
	} else if prior.Scoped == nil && serverScoped {
		a.Scoped = nil
	} else {
		a.Scoped = boolOptDefaultTrue(prior.Scoped, serverScoped)
	}

	// Masks: keep the user's ordering/spelling when semantically equal,
	// otherwise take the server's view.
	if !isImport && sameMasks(prior.Masks, r.Masks) {
		a.Masks = prior.Masks
	} else if len(r.Masks) == 0 {
		a.Masks = nil
	} else {
		a.Masks = r.Masks
	}
	return a
}

// boolOptDefaultTrue reconciles an optional bool whose server default is true.
func boolOptDefaultTrue(prior *bool, server bool) *bool {
	if prior != nil && *prior == server {
		return prior
	}
	return &server
}

// sameMasks compares two mask sets structurally, ignoring ordering of
// databases, tables, and columns.
func sameMasks(a, b []DataProtectionMask) bool {
	if len(a) != len(b) {
		return false
	}
	return maskKey(a) == maskKey(b)
}

func maskKey(masks []DataProtectionMask) string {
	dbs := make([]string, 0, len(masks))
	for _, m := range masks {
		tables := make([]string, 0, len(m.Tables))
		for _, t := range m.Tables {
			cols := make([]string, 0, len(t.MaskedColumns))
			for _, c := range t.MaskedColumns {
				cols = append(cols, c.ColumnName+"="+c.MaskingType)
			}
			sort.Strings(cols)
			masked := t.Masked != nil && *t.Masked
			tables = append(tables, fmt.Sprintf("%s|%s|%t|%v", t.TableName, t.Schema, masked, cols))
		}
		sort.Strings(tables)
		dbs = append(dbs, fmt.Sprintf("%s>>%v", m.DatabaseName, tables))
	}
	sort.Strings(dbs)
	return fmt.Sprintf("%v", dbs)
}
