package adaptive

import (
	"context"
	"strings"
	"time"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// ===========================================================================
// Schedule  (adaptive:index:Schedule) — auto-approval schedule
// ===========================================================================

type Schedule struct{}

type ScheduleArgs struct {
	Name          string   `pulumi:"name"`
	ScheduleType  string   `pulumi:"scheduleType"`
	Description   *string  `pulumi:"description,optional"`
	IsActive      *bool    `pulumi:"isActive,optional"`
	AllDay        *bool    `pulumi:"allDay,optional"`
	StartHour     *int     `pulumi:"startHour,optional"`
	StartMinute   *int     `pulumi:"startMinute,optional"`
	EndHour       *int     `pulumi:"endHour,optional"`
	EndMinute     *int     `pulumi:"endMinute,optional"`
	Weekdays      []string `pulumi:"weekdays,optional"`
	StartDay      *int     `pulumi:"startDay,optional"`
	EndDay        *int     `pulumi:"endDay,optional"`
	SpecificDates []string `pulumi:"specificDates,optional"`
	Users         []string `pulumi:"users,optional"`
	Teams         []string `pulumi:"teams,optional"`
	Endpoints     []string `pulumi:"endpoints,optional"`
	ExpiresAt     *string  `pulumi:"expiresAt,optional"`
	MaxAccessTime *int     `pulumi:"maxAccessTime,optional"`
	Timezone      *string  `pulumi:"timezone,optional"`
	OperationType *string  `pulumi:"operationType,optional"`
}

type ScheduleState struct {
	ScheduleArgs
	UpdatedAt string `pulumi:"updatedAt,optional"`
}

func (s *ScheduleArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.Name, "Name of the schedule. Note: the Adaptive API upserts schedules by name, "+
		"so creating a schedule whose name already exists adopts the existing schedule.")
	a.Describe(&s.ScheduleType, "The schedule pattern: weekdays, weekends, everyday, monthly, specific, or custom.")
	a.Describe(&s.Description, "An optional description of the schedule.")
	a.Describe(&s.IsActive, "Whether the schedule is active.")
	a.Describe(&s.AllDay, "Whether the schedule covers the whole day (start/end times are ignored).")
	a.Describe(&s.StartHour, "Hour (0-23) the window opens.")
	a.Describe(&s.StartMinute, "Minute (0-59) the window opens.")
	a.Describe(&s.EndHour, "Hour (0-23) the window closes.")
	a.Describe(&s.EndMinute, "Minute (0-59) the window closes.")
	a.Describe(&s.Weekdays, "Weekday names for a custom schedule (e.g. monday, tuesday).")
	a.Describe(&s.StartDay, "Day of month (1-31) the window opens, for monthly schedules.")
	a.Describe(&s.EndDay, "Day of month (1-31) the window closes, for monthly schedules.")
	a.Describe(&s.SpecificDates, "RFC3339 dates for a specific-dates schedule.")
	a.Describe(&s.Users, "Emails of users the schedule applies to.")
	a.Describe(&s.Teams, "Names of groups the schedule applies to.")
	a.Describe(&s.Endpoints, "Names of endpoints the schedule applies to.")
	a.Describe(&s.ExpiresAt, "RFC3339 timestamp after which the schedule no longer applies.")
	a.Describe(&s.MaxAccessTime, "Maximum access duration in minutes granted by an auto-approval.")
	a.Describe(&s.Timezone, "IANA timezone for the schedule window. Empty inherits the workspace default.")
	a.Describe(&s.OperationType, "What the schedule does inside its window: autoapprove or autoreject.")
	a.SetDefault(&s.OperationType, "autoapprove")
}

func (s *ScheduleState) Annotate(a infer.Annotator) {
	a.Describe(&s.UpdatedAt, "Last time the schedule was modified.")
}

func (a ScheduleArgs) toScheduleRequest() ScheduleRequest {
	return ScheduleRequest{
		Name:          a.Name,
		Description:   sv(a.Description),
		ScheduleType:  a.ScheduleType,
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
		ExpiresAt:     a.ExpiresAt,
		MaxAccessTime: a.MaxAccessTime,
		Timezone:      sv(a.Timezone),
		OperationType: sv(a.OperationType),
	}
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
	out.Output.UpdatedAt = resp.UpdatedAt
	return out, nil
}

func (*Schedule) Update(ctx context.Context, req infer.UpdateRequest[ScheduleArgs, ScheduleState]) (infer.UpdateResponse[ScheduleState], error) {
	out := infer.UpdateResponse[ScheduleState]{Output: ScheduleState{ScheduleArgs: req.Inputs, UpdatedAt: req.State.UpdatedAt}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	resp, err := c.UpdateSchedule(ctx, req.ID, req.Inputs.toScheduleRequest())
	if err != nil {
		return out, err
	}
	out.Output.UpdatedAt = resp.UpdatedAt
	return out, nil
}

func (*Schedule) Read(ctx context.Context, req infer.ReadRequest[ScheduleArgs, ScheduleState]) (infer.ReadResponse[ScheduleArgs, ScheduleState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[ScheduleArgs, ScheduleState]{}, err
	}
	r, err := c.ReadSchedule(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[ScheduleArgs, ScheduleState]{}, err
	}
	if r == nil {
		return infer.ReadResponse[ScheduleArgs, ScheduleState]{}, nil
	}
	isImport := req.Inputs.Name == ""
	inputs := applyScheduleRead(req.Inputs, r, isImport)
	return infer.ReadResponse[ScheduleArgs, ScheduleState]{
		ID:     req.ID,
		Inputs: inputs,
		State:  ScheduleState{ScheduleArgs: inputs, UpdatedAt: r.UpdatedAt},
	}, nil
}

func (*Schedule) Delete(ctx context.Context, req infer.DeleteRequest[ScheduleState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteSchedule(ctx, req.ID)
}

// applyScheduleRead reconciles schedule inputs against a read.
func applyScheduleRead(prior ScheduleArgs, r *ScheduleReadResponse, isImport bool) ScheduleArgs {
	a := prior
	if isImport {
		a = ScheduleArgs{}
	}
	a.Name = r.Name
	a.ScheduleType = r.ScheduleType
	a.Description = strOpt(prior.Description, r.Description, isImport)

	// IsActive defaults to true server-side; an unset input reading back true
	// is not drift.
	if !isImport && prior.IsActive == nil && r.IsActive {
		a.IsActive = nil
	} else {
		active := r.IsActive
		if isImport {
			a.IsActive = &active
		} else {
			a.IsActive = boolOpt(prior.IsActive, r.IsActive, false)
		}
	}

	a.AllDay = boolOpt(prior.AllDay, r.AllDay, isImport)
	a.StartHour = intOpt(prior.StartHour, r.StartHour, isImport)
	a.StartMinute = intOpt(prior.StartMinute, r.StartMinute, isImport)
	a.EndHour = intOpt(prior.EndHour, r.EndHour, isImport)
	a.EndMinute = intOpt(prior.EndMinute, r.EndMinute, isImport)

	// The server lowercases weekday names; keep the user's casing when the
	// sets match case-insensitively.
	if sameSet(lowerAll(prior.Weekdays), lowerAll(r.Weekdays)) {
		a.Weekdays = prior.Weekdays
	} else {
		a.Weekdays = r.Weekdays
	}

	a.StartDay = intOpt(prior.StartDay, r.StartDay, isImport)
	a.EndDay = intOpt(prior.EndDay, r.EndDay, isImport)

	// The server returns dates normalized to UTC RFC3339; keep the user's
	// spelling when the instants match.
	if sameTimeSet(prior.SpecificDates, r.SpecificDates) {
		a.SpecificDates = prior.SpecificDates
	} else {
		a.SpecificDates = r.SpecificDates
	}

	a.Users = setList(prior.Users, r.Users)
	a.Teams = setList(prior.Teams, r.Teams)
	a.Endpoints = setList(prior.Endpoints, r.Endpoints)

	if !isImport && prior.ExpiresAt != nil && sameInstant(*prior.ExpiresAt, r.ExpiresAt) {
		a.ExpiresAt = prior.ExpiresAt
	} else {
		a.ExpiresAt = strOpt(prior.ExpiresAt, r.ExpiresAt, isImport)
	}

	a.MaxAccessTime = r.MaxAccessTime
	if !isImport && prior.MaxAccessTime == nil && r.MaxAccessTime == nil {
		a.MaxAccessTime = nil
	}
	a.Timezone = strOpt(prior.Timezone, r.Timezone, isImport)

	// operationType defaults to autoapprove; keep the user's (possibly unset)
	// value when it resolves to what the server has.
	if !isImport && effectiveOperationType(prior.OperationType) == effectiveOperationType(strPtrOrNil(r.OperationType)) {
		a.OperationType = prior.OperationType
	} else {
		a.OperationType = strOpt(prior.OperationType, r.OperationType, isImport)
	}
	return a
}

func effectiveOperationType(v *string) string {
	if sv(v) == "" {
		return "autoapprove"
	}
	return sv(v)
}

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strings.ToLower(v)
	}
	return out
}

// sameInstant reports whether two RFC3339 timestamps denote the same moment.
func sameInstant(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339, a)
	tb, errB := time.Parse(time.RFC3339, b)
	if errA != nil || errB != nil {
		return a == b
	}
	return ta.Equal(tb)
}

// sameTimeSet compares two timestamp lists as sets of instants.
func sameTimeSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	used := make([]bool, len(b))
outer:
	for _, va := range a {
		for i, vb := range b {
			if !used[i] && sameInstant(va, vb) {
				used[i] = true
				continue outer
			}
		}
		return false
	}
	return true
}

// iv dereferences an optional int pointer, returning 0 for nil.
func iv(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
