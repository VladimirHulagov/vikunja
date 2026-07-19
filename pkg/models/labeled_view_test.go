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
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// All tests in this file run against project 100 (owned by user 100, see
// fixtures/projects.yml, fixtures/tasks.yml, fixtures/labels.yml and
// fixtures/label_tasks.yml). The fixture layout under `done = false`:
//
//   label 100 (AA Label): tasks #100, #101, #102           count 3
//   label 101 (BB Label): tasks #100, #103, #104, #105     count 4
//   label 102 (ZZ Label): HIDDEN (only attached to done #106)
//   untagged:             tasks #107..#111                 count 5
//
// Task #100 is multi-label (labels 100 AND 101) and is the critical case for
// the "appears in EACH matching group" invariant.

// loadLabeledView loads a project view by ID from the fixtures and returns it.
// Helper to keep test setup terse.
func loadLabeledView(t *testing.T, s *xorm.Session, viewID int64) *ProjectView {
	t.Helper()
	view, err := GetProjectViewByID(s, viewID)
	require.NoError(t, err)
	return view
}

// labelIDs collects the IDs of the labels on each group for easy comparison.
func labelIDs(groups []*LabeledGroup) []int64 {
	ids := make([]int64, 0, len(groups))
	for _, g := range groups {
		if g.Label != nil {
			ids = append(ids, g.Label.ID)
		}
	}
	return ids
}

func taskIDs(tasks []*Task) []int64 {
	ids := make([]int64, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
	}
	return ids
}

// findGroup returns the group with the given label id, or nil.
func findGroup(groups []*LabeledGroup, labelID int64) *LabeledGroup {
	for _, g := range groups {
		if g.Label != nil && g.Label.ID == labelID {
			return g
		}
	}
	return nil
}

// TestGetLabeledViewGroups_EmptyProject verifies that a filter matching no
// tasks returns an empty (non-nil) Groups slice and a nil UntaggedGroup.
func TestGetLabeledViewGroups_EmptyProject(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000)
	tc := &TaskCollection{Filter: "title = 'DEFINITELY_DOES_NOT_EXIST'"}
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, tc, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	require.NotNil(t, resp)
	assert.NotNil(t, resp.Groups, "Groups should be non-nil even when empty")
	assert.Empty(t, resp.Groups)
	assert.Nil(t, resp.UntaggedGroup, "UntaggedGroup must be nil when nothing matches")
}

// TestGetLabeledViewGroups_TasksWithOneLabel verifies that one group is
// produced per label that has matching tasks.
func TestGetLabeledViewGroups_TasksWithOneLabel(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	// Under done=false, only labels 100 and 101 should produce columns.
	// Label 102 is only on done task #106, so it's hidden.
	assert.ElementsMatch(t, []int64{100, 101}, labelIDs(resp.Groups))
	assert.NotNil(t, resp.UntaggedGroup, "untagged expected (project 100 has tasks 107-111)")
}

// TestGetLabeledViewGroups_MultiLabelTaskInEachGroup verifies the critical
// invariant: a task with multiple labels appears in EACH matching group.
// Task #100 carries labels 100 and 101 — it must show up under both.
func TestGetLabeledViewGroups_MultiLabelTaskInEachGroup(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	g100 := findGroup(resp.Groups, 100)
	g101 := findGroup(resp.Groups, 101)
	require.NotNil(t, g100, "label 100 group must exist")
	require.NotNil(t, g101, "label 101 group must exist")

	assert.Contains(t, taskIDs(g100.Tasks), int64(100), "task #100 must appear in label 100 group")
	assert.Contains(t, taskIDs(g101.Tasks), int64(100), "task #100 must appear in label 101 group")
}

// TestGetLabeledViewGroups_UntaggedTasksPresent verifies that tasks without
// any labels are collected into UntaggedGroup.
func TestGetLabeledViewGroups_UntaggedTasksPresent(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	require.NotNil(t, resp.UntaggedGroup, "UntaggedGroup must be set when project has untagged tasks")
	assert.Nil(t, resp.UntaggedGroup.Label, "UntaggedGroup.Label must be nil")
	// Tasks 107-111 in project 100 are untagged.
	assert.Equal(t, int64(5), resp.UntaggedGroup.TaskCount)
	assert.ElementsMatch(t, []int64{107, 108, 109, 110, 111}, taskIDs(resp.UntaggedGroup.Tasks))
}

// TestGetLabeledViewGroups_NoUntaggedTasks verifies UntaggedGroup is nil when
// every matching task has at least one label.
func TestGetLabeledViewGroups_NoUntaggedTasks(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	// view 1004 filters `done = true` — only task #106 matches, and task #106
	// has label 102. Therefore there are zero untagged tasks.
	view := loadLabeledView(t, s, 1004)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	assert.Nil(t, resp.UntaggedGroup, "UntaggedGroup must be nil when every matching task has labels")
	require.Len(t, resp.Groups, 1, "only label 102 should produce a column")
	assert.Equal(t, int64(102), resp.Groups[0].Label.ID)
}

// TestGetLabeledViewGroups_DoneFilteredOutByDefault verifies that the default
// `done = false` filter excludes done tasks from every group and from
// UntaggedGroup.
func TestGetLabeledViewGroups_DoneFilteredOutByDefault(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000) // filter: done = false
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	for _, g := range resp.Groups {
		assert.NotContains(t, taskIDs(g.Tasks), int64(106), "done task #106 should not be in any group")
	}
	if resp.UntaggedGroup != nil {
		assert.NotContains(t, taskIDs(resp.UntaggedGroup.Tasks), int64(106))
	}
	// Label 102 is only attached to done task #106 — without the done task,
	// label 102 must not be a column.
	assert.NotContains(t, labelIDs(resp.Groups), int64(102))
}

// TestGetLabeledViewGroups_DoneIncludedWithoutFilter verifies that removing
// the done filter includes done tasks (and done-only labels reappear).
func TestGetLabeledViewGroups_DoneIncludedWithoutFilter(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	// view 1003 has no done filter at all.
	view := loadLabeledView(t, s, 1003)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	// task #106 is done; with no done filter it should appear in label 102 group.
	g102 := findGroup(resp.Groups, 102)
	require.NotNil(t, g102, "label 102 group should exist when done task #106 is included")
	assert.Contains(t, taskIDs(g102.Tasks), int64(106))
}

// TestGetLabeledViewGroups_GroupSortTaskCount verifies the default sort:
// groups ordered by TaskCount desc, tie-break by Label.Title asc.
func TestGetLabeledViewGroups_GroupSortTaskCount(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000) // task_count
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	// Expected counts (project 100, done=false):
	//   label 101: 4 tasks (#100, #103, #104, #105)
	//   label 100: 3 tasks (#100, #101, #102)
	// No tie here so order is purely by count desc.
	assert.Equal(t, []int64{101, 100}, labelIDs(resp.Groups), "groups should be ordered by task_count desc then title asc")

	counts := map[int64]int64{}
	for _, g := range resp.Groups {
		if g.Label != nil {
			counts[g.Label.ID] = g.TaskCount
		}
	}
	assert.Equal(t, int64(4), counts[101])
	assert.Equal(t, int64(3), counts[100])
}

// TestGetLabeledViewGroups_GroupSortTitleAsc verifies title_asc sort.
func TestGetLabeledViewGroups_GroupSortTitleAsc(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1001) // title_asc
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	// 'AA Label' < 'BB Label'
	assert.Equal(t, []int64{100, 101}, labelIDs(resp.Groups))
}

// TestGetLabeledViewGroups_GroupSortTitleDesc verifies title_desc sort.
func TestGetLabeledViewGroups_GroupSortTitleDesc(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1002) // title_desc
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	// 'BB Label' > 'AA Label'
	assert.Equal(t, []int64{101, 100}, labelIDs(resp.Groups))
}

// TestGetLabeledViewGroups_TaskSortWithinGroup verifies the default within-group
// task sort: priority DESC, due_date ASC, id DESC.
func TestGetLabeledViewGroups_TaskSortWithinGroup(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	// Label 101 group has tasks:
	//   #100 priority=100, due 2018-12-10
	//   #103 priority=0,   due 2018-12-03
	//   #104 priority=0,   due 2018-12-04
	//   #105 priority=0,   no due date
	// Expected order under priority DESC, due_date ASC, id DESC:
	//   #100 (priority 100)
	//   #103 (priority 0, earliest due_date)
	//   #104 (priority 0, next due_date)
	//   #105 (priority 0, no due_date -> sorts last by id desc)
	g101 := findGroup(resp.Groups, 101)
	require.NotNil(t, g101)
	assert.Equal(t, []int64{100, 103, 104, 105}, taskIDs(g101.Tasks))
}

// TestGetLabeledViewGroups_Pagination verifies the per-group pagination slice.
// We unit-test the helper directly because no fixture has 25+ tasks per label.
func TestGetLabeledViewGroups_Pagination(t *testing.T) {
	tasks := make([]*Task, 30)
	for i := range tasks {
		tasks[i] = &Task{ID: int64(i + 1)}
	}

	page1 := paginateLabeledGroupTasks(tasks, 1)
	assert.Len(t, page1, labeledViewTasksPerGroup)
	assert.Equal(t, int64(1), page1[0].ID)
	assert.Equal(t, int64(25), page1[24].ID)

	page2 := paginateLabeledGroupTasks(tasks, 2)
	assert.Len(t, page2, 30-labeledViewTasksPerGroup)
	assert.Equal(t, int64(26), page2[0].ID)

	page3 := paginateLabeledGroupTasks(tasks, 3)
	assert.Empty(t, page3)
	assert.NotNil(t, page3, "page past end must be empty non-nil slice, not nil")

	// Page < 1 should clamp to page 1.
	assert.Equal(t, page1, paginateLabeledGroupTasks(tasks, 0))
	assert.Equal(t, page1, paginateLabeledGroupTasks(tasks, -1))
}

// TestGetLabeledViewGroups_LabelOnlyOnDoneTasks verifies the spec edge case:
// a label whose only matching task is done is hidden under the default
// `done = false` filter, but reappears once done tasks are included.
func TestGetLabeledViewGroups_LabelOnlyOnDoneTasks(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	// Default filter `done = false`: label 102 hidden.
	view := loadLabeledView(t, s, 1000)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)
	assert.NotContains(t, labelIDs(resp.Groups), int64(102))

	// No done filter: label 102 reappears.
	viewNoDone := loadLabeledView(t, s, 1003)
	resp2, err := GetLabeledViewGroups(s, &Project{ID: 100}, viewNoDone, &TaskCollection{}, &user.User{ID: 100}, 1)
	require.NoError(t, err)
	assert.Contains(t, labelIDs(resp2.Groups), int64(102))
}

// TestGetLabeledViewGroups_PermissionDenied verifies that a user without
// read access to the project gets a forbidden error.
func TestGetLabeledViewGroups_PermissionDenied(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	// Project 100 is owned by user 100; user 1 has no access (no users_projects
	// entry, no team_projects entry).
	view := loadLabeledView(t, s, 1000)
	_, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 1}, 1)
	require.Error(t, err)
	assert.True(t, IsErrGenericForbidden(err))
}

// TestGetLabeledViewGroups_RequestFilterANDedWithViewFilter verifies that a
// caller-supplied filter on the TaskCollection is AND-combined with the
// view filter, narrowing the result.
func TestGetLabeledViewGroups_RequestFilterANDedWithViewFilter(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000) // view filter: done = false
	// Further restrict to high-priority tasks only.
	tc := &TaskCollection{Filter: "priority >= 100"}
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, tc, &user.User{ID: 100}, 1)
	require.NoError(t, err)

	// Only task #100 (priority 100) matches both filters. Task #100 has
	// labels 100 and 101. So 2 groups, each with count 1 and only task #100.
	assert.ElementsMatch(t, []int64{100, 101}, labelIDs(resp.Groups))
	for _, g := range resp.Groups {
		assert.Equal(t, int64(1), g.TaskCount)
		assert.Equal(t, []int64{100}, taskIDs(g.Tasks))
	}
	assert.Nil(t, resp.UntaggedGroup, "no untagged task matches priority >= 100")
}

// TestGetLabeledViewGroups_DispatchViaReadAll verifies the wiring: a
// TaskCollection.ReadAll for a labeled view returns a *LabeledViewResponse
// rather than a []*Task.
func TestGetLabeledViewGroups_DispatchViaReadAll(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	tc := &TaskCollection{
		ProjectID:     100,
		ProjectViewID: 1000,
	}
	result, _, _, err := tc.ReadAll(s, &user.User{ID: 100}, "", 1, 25)
	require.NoError(t, err)

	resp, ok := result.(*LabeledViewResponse)
	require.True(t, ok, "expected *LabeledViewResponse, got %T", result)
	assert.NotEmpty(t, resp.Groups)
	assert.NotNil(t, resp.UntaggedGroup)
}

// TestGetLabeledViewGroups_DispatchDeniedViaReadAll verifies the dispatch
// path enforces permission (a non-reader calling ReadAll gets forbidden).
func TestGetLabeledViewGroups_DispatchDeniedViaReadAll(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	tc := &TaskCollection{
		ProjectID:     100,
		ProjectViewID: 1000,
	}
	_, _, _, err := tc.ReadAll(s, &user.User{ID: 1}, "", 1, 25)
	require.Error(t, err)
	assert.True(t, IsErrGenericForbidden(err))
}
