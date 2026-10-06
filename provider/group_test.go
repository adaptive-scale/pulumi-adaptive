package adaptive

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupCreatePartialFailureCanBeRetried(t *testing.T) {
	var creates, updates atomic.Int32
	srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		switch r.URL.Path {
		case "/api/v1/terraform/team/create":
			creates.Add(1)
			_ = json.NewEncoder(w).Encode(IDResponse{ID: "group-1"})
		case "/api/v1/terraform/team/update/group-1":
			var body struct {
				Name           string
				Members        []string
				Endpoints      []string
				SlackChannelID string
			}
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Equal(t, "team", body.Name)
			assert.Equal(t, []string{"member@example.com"}, body.Members)
			assert.Equal(t, []string{"endpoint"}, body.Endpoints)
			assert.Equal(t, "C123", body.SlackChannelID)
			if updates.Add(1) == 1 {
				http.Error(w, "slack unavailable", http.StatusInternalServerError)
			}
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	inputs := property.NewMap(map[string]property.Value{
		"name":           property.New("team"),
		"members":        property.New([]property.Value{property.New("member@example.com")}),
		"endpoints":      property.New([]property.Value{property.New("endpoint")}),
		"slackChannelId": property.New("C123"),
	})
	urn := urnFor("adaptive:index:Group")

	// Exercise infer's response encoding: a normal error discards the created ID.
	created, err := srv.Create(p.CreateRequest{Urn: urn, Properties: inputs})
	require.Error(t, err)
	require.Equal(t, "group-1", created.ID)
	require.NotNil(t, created.PartialState)
	assert.Contains(t, strings.Join(created.PartialState.Reasons, " "), "setting slackChannelId failed")
	assert.Contains(t, strings.Join(created.PartialState.Reasons, " "), "slack unavailable")
	assert.True(t, created.Properties.Get("slackChannelId").IsNull(), "failed Slack configuration must not be recorded as applied")
	for _, key := range []string{"name", "members", "endpoints"} {
		assert.Equal(t, inputs.Get(key), created.Properties.Get(key), key)
	}

	// Pulumi retries initialization with Update and the retained ID/state.
	updated, err := srv.Update(p.UpdateRequest{
		Urn: urn, ID: created.ID, State: created.Properties, OldInputs: inputs, Inputs: inputs,
	})
	require.NoError(t, err)
	assert.Nil(t, updated.PartialState)
	for _, key := range []string{"name", "members", "endpoints", "slackChannelId"} {
		assert.Equal(t, inputs.Get(key), updated.Properties.Get(key), key)
	}
	assert.EqualValues(t, 1, creates.Load(), "retry must not create a second group")
	assert.EqualValues(t, 2, updates.Load())
}

func TestGroupCreate(t *testing.T) {
	for _, tc := range []struct {
		name       string
		slack      string
		preview    bool
		failCreate bool
	}{
		{name: "with Slack", slack: "C123"},
		{name: "without Slack"},
		{name: "create fails", slack: "C123", failCreate: true},
		{name: "preview", slack: "C123", preview: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var creates, updates atomic.Int32
			srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/terraform/team/create":
					creates.Add(1)
					if tc.failCreate {
						http.Error(w, "create failed", http.StatusInternalServerError)
						return
					}
					_ = json.NewEncoder(w).Encode(IDResponse{ID: "group-1"})
				case "/api/v1/terraform/team/update/group-1":
					updates.Add(1)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			})
			inputs := property.NewMap(map[string]property.Value{"name": property.New("team")})
			if tc.slack != "" {
				inputs = inputs.Set("slackChannelId", property.New(tc.slack))
			}
			created, err := srv.Create(p.CreateRequest{
				Urn: urnFor("adaptive:index:Group"), Properties: inputs, DryRun: tc.preview,
			})
			assert.Nil(t, created.PartialState)
			switch {
			case tc.preview:
				require.NoError(t, err)
				assert.Empty(t, created.ID)
				assert.Zero(t, creates.Load())
				assert.Zero(t, updates.Load())
			case tc.failCreate:
				require.ErrorContains(t, err, "create failed")
				assert.Empty(t, created.ID)
				assert.EqualValues(t, 1, creates.Load())
				assert.Zero(t, updates.Load())
			default:
				require.NoError(t, err)
				assert.Equal(t, "group-1", created.ID)
				assert.Equal(t, inputs.Get("slackChannelId"), created.Properties.Get("slackChannelId"))
				assert.EqualValues(t, 1, creates.Load())
				if tc.slack == "" {
					assert.Zero(t, updates.Load())
				} else {
					assert.EqualValues(t, 1, updates.Load())
				}
			}
		})
	}
}
