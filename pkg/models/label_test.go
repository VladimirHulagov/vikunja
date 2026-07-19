// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"reflect"
	"runtime"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestLabel_ReadAll(t *testing.T) {
	type fields struct {
		ID          int64
		Title       string
		Description string
		HexColor    string
		CreatedByID int64
		CreatedBy   *user.User
		Created     time.Time
		Updated     time.Time
		CRUDable    web.CRUDable
		Permissions web.Permissions
	}
	type args struct {
		search string
		a      web.Auth
		page   int
	}
	user1 := &user.User{
		ID:                           1,
		Username:                     "user1",
		Password:                     "$2a$04$X4aRMEt0ytgPwMIgv36cI..7X9.nhY/.tYwxpqSi0ykRHx2CwQ0S6",
		Issuer:                       "local",
		EmailRemindersEnabled:        true,
		OverdueTasksRemindersEnabled: true,
		OverdueTasksRemindersTime:    "09:00",
		Created:                      testCreatedTime,
		Updated:                      testUpdatedTime,
		ExportFileID:                 1,
	}
	user2 := &user.User{
		ID:                           2,
		Username:                     "user2",
		Password:                     "$2a$04$X4aRMEt0ytgPwMIgv36cI..7X9.nhY/.tYwxpqSi0ykRHx2CwQ0S6",
		Issuer:                       "local",
		EmailRemindersEnabled:        true,
		OverdueTasksRemindersEnabled: true,
		OverdueTasksRemindersTime:    "09:00",
		DefaultProjectID:             4,
		Created:                      testCreatedTime,
		Updated:                      testUpdatedTime,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantLs  []*LabelWithTaskID
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				a: &user.User{ID: 1},
			},
			wantLs: []*LabelWithTaskID{
				{
					Label: Label{
						ID:          1,
						Title:       "Label #1",
						CreatedByID: 1,
						CreatedBy:   user1,
						Created:     testCreatedTime,
						Updated:     testUpdatedTime,
					},
				},
				{
					Label: Label{
						ID:          2,
						Title:       "Label #2",
						CreatedByID: 1,
						CreatedBy:   user1,
						Created:     testCreatedTime,
						Updated:     testUpdatedTime,
					},
				},
				{
					Label: Label{
						ID:          4,
						Title:       "Label #4 - visible via other task",
						Created:     testCreatedTime,
						Updated:     testUpdatedTime,
						CreatedByID: 2,
						CreatedBy:   user2,
					},
				},
				{
					Label: Label{
						ID:          7,
						Title:       "Label #7 - created by user 1, no task attachment",
						CreatedByID: 1,
						CreatedBy:   user1,
						Created:     testCreatedTime,
						Updated:     testUpdatedTime,
					},
				},
				{
					Label: Label{
						ID:          8,
						Title:       "Label #8 - user 1 creator, only attached to inaccessible task",
						CreatedByID: 1,
						CreatedBy:   user1,
						Created:     testCreatedTime,
						Updated:     testUpdatedTime,
					},
				},
			},
		},
		{
			name: "invalid user",
			args: args{
				a: &user.User{ID: -1},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Label{
				ID:          tt.fields.ID,
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
				HexColor:    tt.fields.HexColor,
				CreatedByID: tt.fields.CreatedByID,
				CreatedBy:   tt.fields.CreatedBy,
				Created:     tt.fields.Created,
				Updated:     tt.fields.Updated,
				CRUDable:    tt.fields.CRUDable,
				Permissions: tt.fields.Permissions,
			}
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()
			gotLs, _, _, err := l.ReadAll(s, tt.args.a, tt.args.search, tt.args.page, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("Label.ReadAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := gotLs.([]*LabelWithTaskID)

			if diff, equal := messagediff.PrettyDiff(got, tt.wantLs); !equal {
				t.Errorf("Label.ReadAll() = %v, want %v, diff: %v", gotLs, tt.wantLs, diff)
			}
		})
	}
}

func TestLabel_ReadOne(t *testing.T) {
	type fields struct {
		ID          int64
		Title       string
		Description string
		HexColor    string
		CreatedByID int64
		CreatedBy   *user.User
		Created     time.Time
		Updated     time.Time
		CRUDable    web.CRUDable
		Permissions web.Permissions
	}
	user1 := &user.User{
		ID:                           1,
		Username:                     "user1",
		Password:                     "$2a$04$X4aRMEt0ytgPwMIgv36cI..7X9.nhY/.tYwxpqSi0ykRHx2CwQ0S6",
		Issuer:                       "local",
		EmailRemindersEnabled:        true,
		OverdueTasksRemindersEnabled: true,
		OverdueTasksRemindersTime:    "09:00",
		Created:                      testCreatedTime,
		Updated:                      testUpdatedTime,
		ExportFileID:                 1,
	}
	tests := []struct {
		name                string
		fields              fields
		want                *Label
		wantErr             bool
		errType             func(error) bool
		auth                web.Auth
		wantForbidden       bool
		assertMaxPermission bool
		wantMaxPermission   int
	}{
		{
			name: "Get label #1",
			fields: fields{
				ID: 1,
			},
			want: &Label{
				ID:          1,
				Title:       "Label #1",
				CreatedByID: 1,
				CreatedBy:   user1,
				Created:     testCreatedTime,
				Updated:     testUpdatedTime,
			},
			auth:                &user.User{ID: 1},
			assertMaxPermission: true,
			wantMaxPermission:   int(PermissionRead),
		},
		{
			name: "Get nonexistant label",
			fields: fields{
				ID: 9999,
			},
			wantErr:       true,
			errType:       IsErrLabelDoesNotExist,
			wantForbidden: true,
			auth:          &user.User{ID: 1},
		},
		{
			name: "no permissions",
			fields: fields{
				ID: 3,
			},
			wantForbidden: true,
			auth:          &user.User{ID: 1},
		},
		{
			// Label 4 is attached to tasks in project 1 (user 1 is admin),
			// so the accessible-tasks iteration must yield PermissionAdmin.
			name: "Get label #4 - other user",
			fields: fields{
				ID: 4,
			},
			want: &Label{
				ID:          4,
				Title:       "Label #4 - visible via other task",
				CreatedByID: 2,
				CreatedBy: &user.User{
					ID:                           2,
					Username:                     "user2",
					Password:                     "$2a$04$X4aRMEt0ytgPwMIgv36cI..7X9.nhY/.tYwxpqSi0ykRHx2CwQ0S6",
					Issuer:                       "local",
					EmailRemindersEnabled:        true,
					OverdueTasksRemindersEnabled: true,
					OverdueTasksRemindersTime:    "09:00",
					DefaultProjectID:             4,
					Created:                      testCreatedTime,
					Updated:                      testUpdatedTime,
				},
				Created: testCreatedTime,
				Updated: testUpdatedTime,
			},
			auth:                &user.User{ID: 1},
			assertMaxPermission: true,
			wantMaxPermission:   int(PermissionAdmin),
		},
		{
			// PoC for GHSA-hj5c-mhh2-g7jq: label 6 is reachable only via task
			// 34 in the private project 20, user 1 must not see it.
			name: "PoC GHSA-hj5c-mhh2-g7jq: label 6 attached only to unreachable task must be forbidden",
			fields: fields{
				ID: 6,
			},
			wantForbidden: true,
			auth:          &user.User{ID: 1},
		},
		{
			// Creator of an unattached label must still be able to read it.
			name: "creator can read own label with no task attachment",
			fields: fields{
				ID: 7,
			},
			want: &Label{
				ID:          7,
				Title:       "Label #7 - created by user 1, no task attachment",
				CreatedByID: 1,
				CreatedBy:   user1,
				Created:     testCreatedTime,
				Updated:     testUpdatedTime,
			},
			auth:                &user.User{ID: 1},
			assertMaxPermission: true,
			wantMaxPermission:   int(PermissionRead),
		},
		{
			// Label 8's only label_tasks row points at inaccessible task 34,
			// so access must come from the creator branch and the
			// maxPermission fallback to PermissionRead must kick in.
			name: "creator can read own label only attached to inaccessible task",
			fields: fields{
				ID: 8,
			},
			want: &Label{
				ID:          8,
				Title:       "Label #8 - user 1 creator, only attached to inaccessible task",
				CreatedByID: 1,
				CreatedBy:   user1,
				Created:     testCreatedTime,
				Updated:     testUpdatedTime,
			},
			auth:                &user.User{ID: 1},
			assertMaxPermission: true,
			wantMaxPermission:   int(PermissionRead),
		},
		{
			// Non-creator must not be able to read an unattached label owned
			// by someone else — label 3 in fixtures.
			name: "non-creator cannot read label with no task attachment",
			fields: fields{
				ID: 3,
			},
			wantForbidden: true,
			auth:          &user.User{ID: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			l := &Label{
				ID:          tt.fields.ID,
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
				HexColor:    tt.fields.HexColor,
				CreatedByID: tt.fields.CreatedByID,
				CreatedBy:   tt.fields.CreatedBy,
				Created:     tt.fields.Created,
				Updated:     tt.fields.Updated,
				CRUDable:    tt.fields.CRUDable,
				Permissions: tt.fields.Permissions,
			}

			s := db.NewSession()
			defer s.Close()

			allowed, maxPermission, _ := l.CanRead(s, tt.auth)
			if !allowed && !tt.wantForbidden {
				t.Errorf("Label.CanRead() forbidden, want %v", tt.wantForbidden)
			}
			if allowed && tt.wantForbidden {
				t.Errorf("Label.CanRead() allowed, want forbidden")
			}
			if tt.assertMaxPermission && maxPermission != tt.wantMaxPermission {
				t.Errorf("Label.CanRead() maxPermission = %d, want %d", maxPermission, tt.wantMaxPermission)
			}
			err := l.ReadOne(s, tt.auth)
			if (err != nil) != tt.wantErr {
				t.Errorf("Label.ReadOne() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (err != nil) && tt.wantErr && !tt.errType(err) {
				t.Errorf("Label.ReadOne() Wrong error type! Error = %v, want = %v", err, runtime.FuncForPC(reflect.ValueOf(tt.errType).Pointer()).Name())
			}
			if diff, equal := messagediff.PrettyDiff(l, tt.want); !equal && !tt.wantErr && !tt.wantForbidden {
				t.Errorf("Label.ReadAll() = %v, want %v, diff: %v", l, tt.want, diff)
			}
		})
	}
}

func TestLabel_Create(t *testing.T) {
	type fields struct {
		ID          int64
		Title       string
		Description string
		HexColor    string
		CreatedByID int64
		CreatedBy   *user.User
		Created     time.Time
		Updated     time.Time
		CRUDable    web.CRUDable
		Permissions web.Permissions
	}
	type args struct {
		a web.Auth
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantErr       bool
		wantForbidden bool
	}{
		{
			name: "normal",
			fields: fields{
				Title:       "Test #1",
				Description: "Lorem Ipsum",
				HexColor:    "ffccff",
			},
			args: args{
				a: &user.User{ID: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Label{
				ID:          tt.fields.ID,
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
				HexColor:    tt.fields.HexColor,
				CreatedByID: tt.fields.CreatedByID,
				CreatedBy:   tt.fields.CreatedBy,
				Created:     tt.fields.Created,
				Updated:     tt.fields.Updated,
				CRUDable:    tt.fields.CRUDable,
				Permissions: tt.fields.Permissions,
			}
			s := db.NewSession()
			defer s.Close()
			allowed, _ := l.CanCreate(s, tt.args.a)
			if !allowed && !tt.wantForbidden {
				t.Errorf("Label.CanCreate() forbidden, want %v", tt.wantForbidden)
			}
			if err := l.Create(s, tt.args.a); (err != nil) != tt.wantErr {
				t.Errorf("Label.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				require.NoError(t, s.Commit())
				db.AssertExists(t, "labels", map[string]interface{}{
					"id":          l.ID,
					"title":       l.Title,
					"description": l.Description,
					"hex_color":   l.HexColor,
				}, false)
			}
		})
	}
}

func TestLabel_Update(t *testing.T) {
	type fields struct {
		ID          int64
		Title       string
		Description string
		HexColor    string
		CreatedByID int64
		CreatedBy   *user.User
		Created     time.Time
		Updated     time.Time
		CRUDable    web.CRUDable
		Permissions web.Permissions
	}
	tests := []struct {
		name          string
		fields        fields
		wantErr       bool
		auth          web.Auth
		wantForbidden bool
	}{
		{
			name: "normal",
			fields: fields{
				ID:    1,
				Title: "new and better",
			},
			auth: &user.User{ID: 1},
		},
		{
			name: "nonexisting",
			fields: fields{
				ID:    99999,
				Title: "new and better",
			},
			auth:          &user.User{ID: 1},
			wantForbidden: true,
			wantErr:       true,
		},
		{
			name: "no permissions",
			fields: fields{
				ID:    3,
				Title: "new and better",
			},
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name: "no permissions other creator but access",
			fields: fields{
				ID:    4,
				Title: "new and better",
			},
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Label{
				ID:          tt.fields.ID,
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
				HexColor:    tt.fields.HexColor,
				CreatedByID: tt.fields.CreatedByID,
				CreatedBy:   tt.fields.CreatedBy,
				Created:     tt.fields.Created,
				Updated:     tt.fields.Updated,
				CRUDable:    tt.fields.CRUDable,
				Permissions: tt.fields.Permissions,
			}
			s := db.NewSession()
			defer s.Close()
			allowed, _ := l.CanUpdate(s, tt.auth)
			if !allowed && !tt.wantForbidden {
				t.Errorf("Label.CanUpdate() forbidden, want %v", tt.wantForbidden)
			}
			if err := l.Update(s, tt.auth); (err != nil) != tt.wantErr {
				t.Errorf("Label.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !tt.wantForbidden {
				require.NoError(t, s.Commit())
				db.AssertExists(t, "labels", map[string]interface{}{
					"id":    tt.fields.ID,
					"title": tt.fields.Title,
				}, false)
			}
		})
	}
}

func TestLabel_Delete(t *testing.T) {
	type fields struct {
		ID          int64
		Title       string
		Description string
		HexColor    string
		CreatedByID int64
		CreatedBy   *user.User
		Created     time.Time
		Updated     time.Time
		CRUDable    web.CRUDable
		Permissions web.Permissions
	}
	tests := []struct {
		name          string
		fields        fields
		wantErr       bool
		auth          web.Auth
		wantForbidden bool
	}{

		{
			name: "normal",
			fields: fields{
				ID: 1,
			},
			auth: &user.User{ID: 1},
		},
		{
			name: "nonexisting",
			fields: fields{
				ID: 99999,
			},
			auth:          &user.User{ID: 1},
			wantForbidden: true, // When the label does not exist, it is forbidden. We should fix this, but for everything.
		},
		{
			name: "no permissions",
			fields: fields{
				ID: 3,
			},
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name: "no permissions but visible",
			fields: fields{
				ID: 4,
			},
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Label{
				ID:          tt.fields.ID,
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
				HexColor:    tt.fields.HexColor,
				CreatedByID: tt.fields.CreatedByID,
				CreatedBy:   tt.fields.CreatedBy,
				Created:     tt.fields.Created,
				Updated:     tt.fields.Updated,
				CRUDable:    tt.fields.CRUDable,
				Permissions: tt.fields.Permissions,
			}
			s := db.NewSession()
			defer s.Close()
			allowed, _ := l.CanDelete(s, tt.auth)
			if !allowed && !tt.wantForbidden {
				t.Errorf("Label.CanDelete() forbidden, want %v", tt.wantForbidden)
			}
			if err := l.Delete(s, tt.auth); (err != nil) != tt.wantErr {
				t.Errorf("Label.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !tt.wantForbidden {
				require.NoError(t, s.Commit())
				db.AssertMissing(t, "labels", map[string]interface{}{
					"id": l.ID,
				})
			}
		})
	}
}

// labelIDsFromReadAll is a helper for the project-filter tests: it asserts
// the ReadAll result is a []*LabelWithTaskID and returns just the label IDs
// in the order they came back from the database.
func labelIDsFromReadAll(t *testing.T, result interface{}) []int64 {
	t.Helper()
	labels, ok := result.([]*LabelWithTaskID)
	require.True(t, ok, "ReadAll result is not []*LabelWithTaskID")
	ids := make([]int64, 0, len(labels))
	for _, l := range labels {
		ids = append(ids, l.ID)
	}
	return ids
}

// TestLabel_ReadAll_FilteredByProject verifies that supplying ProjectIDFilter
// narrows the result to labels actually attached to tasks in that project,
// owned by the requesting user. Uses the project-100 / user-100 fixture
// cluster (see fixtures/labels.yml + label_tasks.yml): labels 100-103 are all
// created by user 100 and attached to tasks 100-106 in project 100.
func TestLabel_ReadAll_FilteredByProject(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	l := &Label{ProjectIDFilter: 100}
	result, count, total, err := l.ReadAll(s, &user.User{ID: 100}, "", 1, 0)
	require.NoError(t, err)

	assert.Equal(t, []int64{100, 101, 102, 103}, labelIDsFromReadAll(t, result))
	assert.Equal(t, 4, count)
	assert.Equal(t, int64(4), total)
}

// TestLabel_ReadAll_NoProjectFilter verifies the existing user-scoped path is
// preserved when ProjectIDFilter is zero. User 100 owns labels 100-103 and
// project 100, so the unfiltered result matches the filtered one in this
// fixture setup — but the code path taken is the legacy GetLabelsByTaskIDs.
func TestLabel_ReadAll_NoProjectFilter(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	l := &Label{}
	result, _, _, err := l.ReadAll(s, &user.User{ID: 100}, "", 1, 0)
	require.NoError(t, err)

	// User 100 has no access to any other user's labels and owns 100-103,
	// so the unfiltered result is exactly those four labels.
	assert.Equal(t, []int64{100, 101, 102, 103}, labelIDsFromReadAll(t, result))
}

// TestLabel_ReadAll_ProjectFilterWithSearch verifies the project filter and
// the search ILIKE compose: only labels matching both predicates come back.
// Label 101 is titled "BB Label" — the only project-100 label matching "BB".
func TestLabel_ReadAll_ProjectFilterWithSearch(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	l := &Label{ProjectIDFilter: 100}
	result, count, total, err := l.ReadAll(s, &user.User{ID: 100}, "BB", 1, 0)
	require.NoError(t, err)

	assert.Equal(t, []int64{101}, labelIDsFromReadAll(t, result))
	assert.Equal(t, 1, count)
	assert.Equal(t, int64(1), total)
}

// TestLabel_ReadAll_ProjectFilterEmpty verifies that a project whose tasks
// have no labels owned by the requesting user yields an empty result, not an
// error. Project 1 has label 4 on tasks 1 and 2, but label 4 is owned by
// user 2 — so user 100's filtered view of project 1 is empty.
func TestLabel_ReadAll_ProjectFilterEmpty(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	l := &Label{ProjectIDFilter: 1}
	result, count, total, err := l.ReadAll(s, &user.User{ID: 100}, "", 1, 0)
	require.NoError(t, err)

	assert.Empty(t, labelIDsFromReadAll(t, result))
	assert.Equal(t, 0, count)
	assert.Equal(t, int64(0), total)
}

// TestLabel_ReadAll_ProjectFilterRespectsUserScope verifies the strict
// user-scope semantics of the project filter: a label created by another user
// and attached to a task in the queried project must NOT appear in the
// requesting user's filtered result. User 1 has full access to project 1,
// but the only label there (#4) is owned by user 2 — so user 1 gets nothing.
func TestLabel_ReadAll_ProjectFilterRespectsUserScope(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	l := &Label{ProjectIDFilter: 1}
	result, count, total, err := l.ReadAll(s, &user.User{ID: 1}, "", 1, 0)
	require.NoError(t, err)

	assert.Empty(t, labelIDsFromReadAll(t, result))
	assert.Equal(t, 0, count)
	assert.Equal(t, int64(0), total)
}
