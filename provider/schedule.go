package adaptive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ===========================================================================
// Schedule  (adaptive:index:Schedule)
// ===========================================================================
//
// Schedules define windows during which access requests are auto-approved (or
// auto-rejected) for a set of users, groups and endpoints. The Terraform
// provider has supported them since the schedules release; this brings the
// Pulumi provider to parity.

func (c *Client) scheduleAPI() string { return c.workspaceURL + "/terraform/schedule" }

// ScheduleRequest is the flat JSON shape the backend's schedule API consumes.
type ScheduleRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ScheduleType string `json:"scheduleType"`
	IsActive     *bool  `json:"isActive,omitempty"`

	AllDay      bool `json:"allDay,omitempty"`
	StartHour   int  `json:"startHour"`
	StartMinute int  `json:"startMinute"`
	EndHour     int  `json:"endHour"`
	EndMinute   int  `json:"endMinute"`

	Weekdays      []string `json:"weekdays,omitempty"`
	StartDay      int      `json:"startDay,omitempty"`
	EndDay        int      `json:"endDay,omitempty"`
	SpecificDates []string `json:"specificDates,omitempty"`

	Users     []string `json:"users,omitempty"`
	Teams     []string `json:"teams,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`

	ExpiresAt     *string `json:"expiresAt,omitempty"`
	MaxAccessTime *int    `json:"maxAccessTime,omitempty"`
	Timezone      string  `json:"timezone,omitempty"`
	OperationType string  `json:"operationType,omitempty"`
}

// ScheduleResponse mirrors the backend's TerraformScheduleResponse. On a 400 the
// backend reuses this shape to report names it could not resolve to ids.
type ScheduleResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ScheduleType string `json:"scheduleType"`
	IsActive     bool   `json:"isActive"`

	AllDay      bool `json:"allDay"`
	StartHour   int  `json:"startHour"`
	StartMinute int  `json:"startMinute"`
	EndHour     int  `json:"endHour"`
	EndMinute   int  `json:"endMinute"`

	Weekdays      []string `json:"weekdays,omitempty"`
	StartDay      int      `json:"startDay,omitempty"`
	EndDay        int      `json:"endDay,omitempty"`
	SpecificDates []string `json:"specificDates,omitempty"`

	Users     []string `json:"users"`
	Teams     []string `json:"teams"`
	Endpoints []string `json:"endpoints"`

	ExpiresAt     string `json:"expiresAt,omitempty"`
	MaxAccessTime *int   `json:"maxAccessTime,omitempty"`
	Timezone      string `json:"timezone,omitempty"`
	OperationType string `json:"operationType,omitempty"`

	UnresolvedUsers     []string `json:"unresolvedUsers,omitempty"`
	UnresolvedTeams     []string `json:"unresolvedTeams,omitempty"`
	UnresolvedEndpoints []string `json:"unresolvedEndpoints,omitempty"`
}

// unresolvedErr turns a backend 400 listing unresolved names into a message that
// says which name was wrong, rather than just "bad request".
func (r *ScheduleResponse) unresolvedErr() error {
	var parts []string
	if len(r.UnresolvedUsers) > 0 {
		parts = append(parts, fmt.Sprintf("users %v", r.UnresolvedUsers))
	}
	if len(r.UnresolvedTeams) > 0 {
		parts = append(parts, fmt.Sprintf("teams %v", r.UnresolvedTeams))
	}
	if len(r.UnresolvedEndpoints) > 0 {
		parts = append(parts, fmt.Sprintf("endpoints %v", r.UnresolvedEndpoints))
	}
	if len(parts) == 0 {
		return nil
	}
	return fmt.Errorf("could not resolve %v — check the names exist in this workspace", parts)
}

func (c *Client) writeSchedule(ctx context.Context, url string, req ScheduleRequest) (*ScheduleResponse, error) {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, err
	}

	resp, err := c.do(ctx, mustReq("POST", url, body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var out ScheduleResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, fmt.Errorf("failed to decode schedule response: %w", err)
		}
		return &out, nil
	}

	if resp.StatusCode == http.StatusBadRequest {
		var out ScheduleResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil {
			if unresolved := out.unresolvedErr(); unresolved != nil {
				return nil, unresolved
			}
		}
		return nil, fmt.Errorf("schedule %q rejected (status 400)", req.Name)
	}

	return nil, fmt.Errorf("error writing schedule %q (status %d): %s", req.Name, resp.StatusCode, readBody(resp))
}

func (c *Client) CreateSchedule(ctx context.Context, req ScheduleRequest) (*ScheduleResponse, error) {
	return c.writeSchedule(ctx, c.scheduleAPI()+"/create", req)
}

func (c *Client) UpdateSchedule(ctx context.Context, id string, req ScheduleRequest) (*ScheduleResponse, error) {
	return c.writeSchedule(ctx, fmt.Sprintf("%s/update/%s", c.scheduleAPI(), id), req)
}

// GetSchedule reads a schedule. Returns (nil, nil) when it no longer exists.
func (c *Client) GetSchedule(ctx context.Context, id string) (*ScheduleResponse, error) {
	var resp ScheduleResponse
	found, err := c.readInto(ctx, fmt.Sprintf("%s/read/%s", c.scheduleAPI(), id), "schedule", id, &resp)
	if err != nil || !found {
		return nil, err
	}
	return &resp, nil
}

// DeleteSchedule removes a schedule. The backend delete is idempotent, so a
// schedule already gone still reports success.
func (c *Client) DeleteSchedule(ctx context.Context, id string) error {
	resp, err := c.do(ctx, mustReq("POST", fmt.Sprintf("%s/delete/%s", c.scheduleAPI(), id), nil))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting schedule %s (status %d): %s", id, resp.StatusCode, readBody(resp))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Resource
// ---------------------------------------------------------------------------

type Schedule struct{}

type ScheduleArgs struct {
	Name         string  `pulumi:"name"`
	ScheduleType string  `pulumi:"scheduleType"`
	Description  *string `pulumi:"description,optional"`
	IsActive     *bool   `pulumi:"isActive,optional"`

	AllDay      *bool `pulumi:"allDay,optional"`
	StartHour   *int  `pulumi:"startHour,optional"`
	StartMinute *int  `pulumi:"startMinute,optional"`
	EndHour     *int  `pulumi:"endHour,optional"`
	EndMinute   *int  `pulumi:"endMinute,optional"`

	Weekdays      []string `pulumi:"weekdays,optional"`
	StartDay      *int     `pulumi:"startDay,optional"`
	EndDay        *int     `pulumi:"endDay,optional"`
	SpecificDates []string `pulumi:"specificDates,optional"`

	Users     []string `pulumi:"users,optional"`
	Teams     []string `pulumi:"teams,optional"`
	Endpoints []string `pulumi:"endpoints,optional"`

	ExpiresAt     *string `pulumi:"expiresAt,optional"`
	MaxAccessTime *int    `pulumi:"maxAccessTime,optional"`
	Timezone      *string `pulumi:"timezone,optional"`
	OperationType *string `pulumi:"operationType,optional"`
}

type ScheduleState struct {
	ScheduleArgs
}

func (s *ScheduleArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.Name, "Name of the schedule. Must be unique within the workspace.")
	a.Describe(&s.ScheduleType, "One of weekdays, weekends, everyday, monthly, specific, or custom.")
	a.Describe(&s.Description, "An optional description of the schedule.")
	a.Describe(&s.IsActive, "Whether the schedule is active.")
	a.SetDefault(&s.IsActive, true)
	a.Describe(&s.AllDay, "Whether the schedule covers the whole day, ignoring the hour and minute bounds.")
	a.SetDefault(&s.AllDay, false)
	a.Describe(&s.StartHour, "Hour the window opens (0-23). Ignored when allDay is set.")
	a.Describe(&s.StartMinute, "Minute the window opens (0-59). Ignored when allDay is set.")
	a.Describe(&s.EndHour, "Hour the window closes (0-23). Ignored when allDay is set.")
	a.Describe(&s.EndMinute, "Minute the window closes (0-59). Ignored when allDay is set.")
	a.Describe(&s.Weekdays, "Weekday names the schedule applies to. Used when scheduleType is custom.")
	a.Describe(&s.StartDay, "Day of month the window opens. Used when scheduleType is monthly.")
	a.Describe(&s.EndDay, "Day of month the window closes. Used when scheduleType is monthly.")
	a.Describe(&s.SpecificDates, "RFC3339 dates the schedule applies to. Used when scheduleType is specific.")
	a.Describe(&s.Users, "Emails of users this schedule applies to.")
	a.Describe(&s.Teams, "Names of groups this schedule applies to.")
	a.Describe(&s.Endpoints, "Names of endpoints this schedule applies to.")
	a.Describe(&s.ExpiresAt, "RFC3339 timestamp after which the schedule stops applying.")
	a.Describe(&s.MaxAccessTime, "Maximum granted access duration, in minutes.")
	a.Describe(&s.Timezone, "IANA timezone the window is evaluated in. Empty inherits the workspace default.")
	a.Describe(&s.OperationType, "Either autoapprove (default) or autoreject.")
	a.SetDefault(&s.OperationType, "autoapprove")
}

func (a ScheduleArgs) toScheduleRequest() ScheduleRequest {
	req := ScheduleRequest{
		Name:          a.Name,
		ScheduleType:  a.ScheduleType,
		Description:   sv(a.Description),
		IsActive:      a.IsActive,
		AllDay:        bv(a.AllDay),
		StartHour:     iv(a.StartHour),
		StartMinute:   iv(a.StartMinute),
		EndHour:       iv(a.EndHour),
		EndMinute:     iv(a.EndMinute),
		Weekdays:      a.Weekdays,
		StartDay:      iv(a.StartDay),
		EndDay:        iv(a.EndDay),
		SpecificDates: a.SpecificDates,
		Users:         a.Users,
		Teams:         a.Teams,
		Endpoints:     a.Endpoints,
		MaxAccessTime: a.MaxAccessTime,
		Timezone:      sv(a.Timezone),
		OperationType: sv(a.OperationType),
	}
	// The backend only parses expiresAt when it is present and non-empty.
	if a.ExpiresAt != nil && *a.ExpiresAt != "" {
		req.ExpiresAt = a.ExpiresAt
	}
	return req
}

func (*Schedule) Create(ctx context.Context, req infer.CreateRequest[ScheduleArgs]) (infer.CreateResponse[ScheduleState], error) {
	out := infer.CreateResponse[ScheduleState]{Output: ScheduleState{ScheduleArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	resp, err := c.CreateSchedule(ctx, req.Inputs.toScheduleRequest())
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	return out, nil
}

func (*Schedule) Update(ctx context.Context, req infer.UpdateRequest[ScheduleArgs, ScheduleState]) (infer.UpdateResponse[ScheduleState], error) {
	out := infer.UpdateResponse[ScheduleState]{Output: ScheduleState{ScheduleArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	_, err = c.UpdateSchedule(ctx, req.ID, req.Inputs.toScheduleRequest())
	return out, err
}

func (*Schedule) Read(ctx context.Context, req infer.ReadRequest[ScheduleArgs, ScheduleState]) (infer.ReadResponse[ScheduleArgs, ScheduleState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[ScheduleArgs, ScheduleState]{}, err
	}

	schedule, err := c.GetSchedule(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[ScheduleArgs, ScheduleState]{}, err
	}
	if schedule == nil {
		return infer.ReadResponse[ScheduleArgs, ScheduleState]{}, nil
	}

	args := req.Inputs
	args.Name = schedule.Name
	args.ScheduleType = schedule.ScheduleType
	args.Description = optString(schedule.Description)
	args.IsActive = &schedule.IsActive
	args.AllDay = &schedule.AllDay
	args.StartHour = &schedule.StartHour
	args.StartMinute = &schedule.StartMinute
	args.EndHour = &schedule.EndHour
	args.EndMinute = &schedule.EndMinute
	args.Weekdays = schedule.Weekdays
	args.StartDay = &schedule.StartDay
	args.EndDay = &schedule.EndDay
	args.SpecificDates = schedule.SpecificDates
	// Users, teams and endpoints come back as emails and names, matching what
	// the write path accepts.
	args.Users = schedule.Users
	args.Teams = schedule.Teams
	args.Endpoints = schedule.Endpoints
	args.ExpiresAt = optString(schedule.ExpiresAt)
	args.MaxAccessTime = schedule.MaxAccessTime
	args.Timezone = optString(schedule.Timezone)
	args.OperationType = optString(schedule.OperationType)

	return infer.ReadResponse[ScheduleArgs, ScheduleState]{
		ID:     req.ID,
		Inputs: args,
		State:  ScheduleState{ScheduleArgs: args},
	}, nil
}

func (*Schedule) Delete(ctx context.Context, req infer.DeleteRequest[ScheduleState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteSchedule(ctx, req.ID)
}
