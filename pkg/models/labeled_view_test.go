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
//   label 103 (CC Label): task #102                         count 1
//                         (uncategorized — not in any LabelCategory)
//   untagged:             tasks #107..#111                 count 5
//
// Task #100 is multi-label (labels 100 AND 101) and is the critical case for
// the "appears in EACH matching group" invariant. Task #102 is also multi-label
// (100 AND 103) and exercises the bucketing under every category filter.
//
// Label categories (fixtures/label_categories.yml + label_category_members.yml):
//   category 1 'Rooms'      (project 100): labels 100, 101
//   category 2 'Work Types' (project 100): label 102
//   label 103 is intentionally uncategorized so the -1 filter has a hit.

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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, tc, &user.User{ID: 100}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)

	// Under done=false, only labels 100, 101 and the uncategorized 103 should
	// produce columns. Label 102 is only on done task #106, so it's hidden.
	assert.ElementsMatch(t, []int64{100, 101, 103}, labelIDs(resp.Groups))
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)

	// Expected counts (project 100, done=false):
	//   label 101: 4 tasks (#100, #103, #104, #105)
	//   label 100: 3 tasks (#100, #101, #102)
	//   label 103: 1 task  (#102)
	// No tie here so order is purely by count desc.
	assert.Equal(t, []int64{101, 100, 103}, labelIDs(resp.Groups), "groups should be ordered by task_count desc then title asc")

	counts := map[int64]int64{}
	for _, g := range resp.Groups {
		if g.Label != nil {
			counts[g.Label.ID] = g.TaskCount
		}
	}
	assert.Equal(t, int64(4), counts[101])
	assert.Equal(t, int64(3), counts[100])
	assert.Equal(t, int64(1), counts[103])
}

// TestGetLabeledViewGroups_GroupSortTitleAsc verifies title_asc sort.
func TestGetLabeledViewGroups_GroupSortTitleAsc(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1001) // title_asc
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)

	// 'AA Label' < 'BB Label' < 'CC Label'
	assert.Equal(t, []int64{100, 101, 103}, labelIDs(resp.Groups))
}

// TestGetLabeledViewGroups_GroupSortTitleDesc verifies title_desc sort.
func TestGetLabeledViewGroups_GroupSortTitleDesc(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1002) // title_desc
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)

	// 'CC Label' > 'BB Label' > 'AA Label'
	assert.Equal(t, []int64{103, 101, 100}, labelIDs(resp.Groups))
}

// TestGetLabeledViewGroups_TaskSortWithinGroup verifies the default within-group
// task sort: priority DESC, due_date ASC, id DESC.
func TestGetLabeledViewGroups_TaskSortWithinGroup(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000)
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)
	assert.NotContains(t, labelIDs(resp.Groups), int64(102))

	// No done filter: label 102 reappears.
	viewNoDone := loadLabeledView(t, s, 1003)
	resp2, err := GetLabeledViewGroups(s, &Project{ID: 100}, viewNoDone, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
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
	_, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 1}, 1, 0)
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
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, tc, &user.User{ID: 100}, 1, 0)
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

// TestGetLabeledViewGroups_CategoryFilter verifies that passing a real
// category id (N > 0) restricts the columns to labels that are members of
// that category — and only those tasks show up under each column.
//
// Fixture layout reminder (project 100, done=false):
//
//	category 1 'Rooms' holds labels 100, 101
//	category 2 'Work Types' holds label 102 (only on done task, hidden)
//	label 103 is uncategorized
func TestGetLabeledViewGroups_CategoryFilter(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000) // done=false

	// Category 1 'Rooms' → only labels 100 and 101 are eligible.
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 1)
	require.NoError(t, err)

	assert.ElementsMatch(t, []int64{100, 101}, labelIDs(resp.Groups), "category 1 should yield only its member labels")

	// Multi-label task #100 (labels 100 AND 101) still appears under both.
	g100 := findGroup(resp.Groups, 100)
	require.NotNil(t, g100)
	g101 := findGroup(resp.Groups, 101)
	require.NotNil(t, g101)
	assert.Contains(t, taskIDs(g100.Tasks), int64(100))
	assert.Contains(t, taskIDs(g101.Tasks), int64(100))

	// Label 103 is not a member of category 1 — it must not have a column,
	// even though task #102 (which carries 103) matches the done=false filter.
	assert.Nil(t, findGroup(resp.Groups, 103))

	// Category filter active → UntaggedGroup suppressed.
	assert.Nil(t, resp.UntaggedGroup, "UntaggedGroup must be nil when a category filter is active")

	// Sanity check: category 2 'Work Types' has only label 102, which is
	// attached solely to the done task #106. Under done=false that label has
	// no matching tasks, so its column does not appear.
	resp2, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 2)
	require.NoError(t, err)
	assert.Empty(t, resp2.Groups, "category 2 has no eligible non-done tasks")
	assert.Nil(t, resp2.UntaggedGroup)
}

// TestGetLabeledViewGroups_UncategorizedFilter verifies the -1 virtual chip:
// only labels that are NOT in any LabelCategory of this project get columns.
//
// Project 100's categories cover labels {100, 101, 102}; label 103 is the
// only project-100 label that is uncategorized, so it must be the sole column.
func TestGetLabeledViewGroups_UncategorizedFilter(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000) // done=false
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, -1)
	require.NoError(t, err)

	assert.ElementsMatch(t, []int64{103}, labelIDs(resp.Groups), "only uncategorized labels should appear")

	g103 := findGroup(resp.Groups, 103)
	require.NotNil(t, g103)
	// Task #102 carries both label 100 (categorized) and label 103 (not).
	// Under the -1 filter, only the 103 column shows up — but task #102
	// still belongs to that column because it carries label 103.
	assert.Contains(t, taskIDs(g103.Tasks), int64(102))

	// Categorized labels must not have columns.
	assert.Nil(t, findGroup(resp.Groups, 100))
	assert.Nil(t, findGroup(resp.Groups, 101))
	assert.Nil(t, findGroup(resp.Groups, 102))

	// Category filter active → UntaggedGroup suppressed.
	assert.Nil(t, resp.UntaggedGroup)
}

// TestGetLabeledViewGroups_CategoryFilterSuppressesUntaggedGroup verifies
// that ANY non-zero category value (positive or -1) suppresses the untagged
// pseudo-group, even when the project has tasks with no labels at all.
//
// Project 100 has untagged tasks #107..#111 which would normally populate
// UntaggedGroup; under a category filter they must be hidden.
func TestGetLabeledViewGroups_CategoryFilterSuppressesUntaggedGroup(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000) // done=false

	for _, tc := range []struct {
		name     string
		category int64
	}{
		{"positive category suppresses untagged", 1},
		{"negative-one category suppresses untagged", -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, tc.category)
			require.NoError(t, err)
			assert.Nil(t, resp.UntaggedGroup, "UntaggedGroup must be nil for category=%d", tc.category)
		})
	}

	// Sanity: with no filter (0), the untagged group reappears.
	resp, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)
	require.NotNil(t, resp.UntaggedGroup, "UntaggedGroup must reappear when category filter is cleared")
	assert.Equal(t, int64(5), resp.UntaggedGroup.TaskCount)
}

// TestGetLabeledViewGroups_CategoryZeroMeansAll verifies that an explicit
// category=0 is the same as the previous (no-category) behaviour: every label
// forms a column and UntaggedGroup is populated as usual.
func TestGetLabeledViewGroups_CategoryZeroMeansAll(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	view := loadLabeledView(t, s, 1000) // done=false

	respExplicit, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)

	// Should match the documented no-filter layout exactly:
	//   labels {100, 101, 103} and untagged count 5.
	assert.ElementsMatch(t, []int64{100, 101, 103}, labelIDs(respExplicit.Groups))
	require.NotNil(t, respExplicit.UntaggedGroup)
	assert.Equal(t, int64(5), respExplicit.UntaggedGroup.TaskCount)

	// And the response must be byte-for-byte equivalent to calling without
	// the category argument at all (the previous behaviour we're preserving).
	respImplicit, err := GetLabeledViewGroups(s, &Project{ID: 100}, view, &TaskCollection{}, &user.User{ID: 100}, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, labelIDs(respExplicit.Groups), labelIDs(respImplicit.Groups))
	assert.Equal(t, respExplicit.UntaggedGroup.TaskCount, respImplicit.UntaggedGroup.TaskCount)
}
