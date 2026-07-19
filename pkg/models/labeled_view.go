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
	"sort"

	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

// labeledViewTasksPerGroup is the page size for tasks returned per group.
// Matches the Kanban TASKS_PER_BUCKET convention.
const labeledViewTasksPerGroup = 25

// LabeledGroup is one column of a labeled view: every task whose label set
// includes Label (within the effective filter). A nil Label denotes the
// "untagged" pseudo-group for tasks without any labels.
type LabeledGroup struct {
	// The label this group represents. nil for the untagged group.
	Label *Label `json:"label"`
	// Total number of tasks matching the filter for this group, regardless
	// of pagination. The frontend uses this to render the column header and
	// to decide whether to show a "load more" button.
	TaskCount int64 `json:"task_count"`
	// The current page of tasks (at most labeledViewTasksPerGroup).
	Tasks []*Task `json:"tasks"`
}

// LabeledViewResponse is the payload returned by the labeled view.
// Groups is always non-nil (possibly empty). UntaggedGroup is nil when no
// untagged tasks match the effective filter.
type LabeledViewResponse struct {
	Groups        []*LabeledGroup `json:"groups"`
	UntaggedGroup *LabeledGroup   `json:"untagged_group,omitempty"`
}

// GetLabeledViewGroups returns groups of tasks keyed by label for a labeled view.
//
// Effective filter = AND(view.Filter, taskCollection.Filter). Typically this is
// `done = false` because the labeled view's default filter is `done = false`.
//
// Groups are sorted per view.BucketConfigurationSortBy: "task_count" (default,
// also used when the field is empty), "title_asc", "title_desc".
//
// Tasks within a group are sorted per taskCollection.SortBy/OrderBy, defaulting
// to priority DESC, due_date ASC, id DESC.
//
// A multi-label task appears in EVERY matching group. Tasks with no labels at
// all are collected into UntaggedGroup, which is nil when no such tasks match.
//
// Paginates labeledViewTasksPerGroup tasks per group; page is 1-based and
// defaults to 1 when zero or negative.
func GetLabeledViewGroups(s *xorm.Session, project *Project, view *ProjectView, taskCollection *TaskCollection, auth web.Auth, page int) (*LabeledViewResponse, error) {
	// 1. Permission check — project read access is required. This also
	// covers link shares and inherited team memberships because it goes
	// through the same CanRead used by every other view.
	can, _, err := project.CanRead(s, auth)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrGenericForbidden{}
	}

	if page < 1 {
		page = 1
	}

	// 2. Compose the effective filter = AND(view.Filter, request filter).
	// We re-merge here (rather than trusting the caller) so the function is
	// safe to call from contexts other than TaskCollection.ReadAll.
	effectiveFilter := mergeLabeledViewFilters(view, taskCollection)

	filterTimezone := taskCollection.FilterTimezone
	if view != nil && view.Filter != nil && view.Filter.FilterTimezone != "" {
		filterTimezone = view.Filter.FilterTimezone
	}

	search := taskCollection.Search
	if view != nil && view.Filter != nil && view.Filter.Search != "" {
		search = view.Filter.Search
	}

	// 3. Build the search options. We do NOT paginate at this layer — we
	// need every matching task so we can bucket it under each of its labels
	// and so the per-group TaskCount reflects the full set. Pagination is
	// applied per group after bucketing.
	opts := &taskSearchOptions{
		filter:         effectiveFilter,
		filterTimezone: filterTimezone,
		search:         search,
	}
	opts.parsedFilters, err = getTaskFiltersFromFilterString(effectiveFilter, filterTimezone)
	if err != nil {
		return nil, err
	}

	opts.sortby = labeledViewSortParams(taskCollection)

	// 4. One fetch of every matching task with labels attached. The existing
	// infrastructure applies the filter via getRawTasksForProjects and then
	// addMoreInfoToTasks populates Labels/Assignees/etc. on each Task.
	allTasks, _, _, err := getTasksForProjects(s, []*Project{project}, auth, opts, view)
	if err != nil {
		return nil, err
	}

	if len(allTasks) == 0 {
		return &LabeledViewResponse{Groups: []*LabeledGroup{}}, nil
	}

	// 5. Bucket tasks by label in memory. A multi-label task lands in each
	// of its label buckets; tasks without labels go to the untagged bucket.
	// allTasks is already in the requested sort order, so each bucket slice
	// preserves that order.
	tasksByLabel := make(map[int64][]*Task)
	var untagged []*Task
	for _, t := range allTasks {
		if len(t.Labels) == 0 {
			untagged = append(untagged, t)
			continue
		}
		for _, l := range t.Labels {
			tasksByLabel[l.ID] = append(tasksByLabel[l.ID], t)
		}
	}

	// 6. Pull the label rows we actually need. We do this in one query to
	// avoid an N+1 when there are many labels.
	labelIDs := make([]int64, 0, len(tasksByLabel))
	for id := range tasksByLabel {
		labelIDs = append(labelIDs, id)
	}
	labels := make([]*Label, 0, len(labelIDs))
	if len(labelIDs) > 0 {
		err = s.In("id", labelIDs).OrderBy("id asc").Find(&labels)
		if err != nil {
			return nil, err
		}
	}
	// Resolve CreatedBy on each label so the response is consistent with
	// the rest of the API. getUsersOrLinkSharesFromIDs is a single query.
	if len(labels) > 0 {
		creatorIDs := make([]int64, 0, len(labels))
		for _, l := range labels {
			if l.CreatedByID != 0 {
				creatorIDs = append(creatorIDs, l.CreatedByID)
			}
		}
		creators, err := getUsersOrLinkSharesFromIDs(s, creatorIDs)
		if err != nil {
			return nil, err
		}
		for _, l := range labels {
			if c, ok := creators[l.CreatedByID]; ok {
				l.CreatedBy = c
			}
		}
	}

	// 7. Build groups (in label-id order for now; sorted in step 8).
	groups := make([]*LabeledGroup, 0, len(labels))
	for _, l := range labels {
		tasksForLabel := tasksByLabel[l.ID]
		groups = append(groups, &LabeledGroup{
			Label:     l,
			TaskCount: int64(len(tasksForLabel)),
			Tasks:     paginateLabeledGroupTasks(tasksForLabel, page),
		})
	}

	// 8. Sort groups per view.BucketConfigurationSortBy.
	sortLabeledGroups(groups, view.BucketConfigurationSortBy)

	// 9. Untagged pseudo-group (only if there are untagged tasks).
	var untaggedGroup *LabeledGroup
	if len(untagged) > 0 {
		untaggedGroup = &LabeledGroup{
			Label:     nil,
			TaskCount: int64(len(untagged)),
			Tasks:     paginateLabeledGroupTasks(untagged, page),
		}
	}

	return &LabeledViewResponse{
		Groups:        groups,
		UntaggedGroup: untaggedGroup,
	}, nil
}

// mergeLabeledViewFilters ANDs together the view's saved filter and the
// per-request filter from the TaskCollection. Returns the empty string when
// neither side has a filter.
func mergeLabeledViewFilters(view *ProjectView, taskCollection *TaskCollection) string {
	var parts []string
	if view != nil && view.Filter != nil && view.Filter.Filter != "" {
		parts = append(parts, "("+view.Filter.Filter+")")
	}
	if taskCollection != nil && taskCollection.Filter != "" {
		parts = append(parts, "("+taskCollection.Filter+")")
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		// Strip redundant parentheses when there's only one clause.
		return trimLabeledFilterParens(parts[0])
	default:
		joined := ""
		for i, p := range parts {
			if i > 0 {
				joined += " && "
			}
			joined += p
		}
		return joined
	}
}

// trimLabeledFilterParens removes a single layer of surrounding parentheses
// from a one-clause filter so we don't send "((done = false))" to the parser.
func trimLabeledFilterParens(s string) string {
	if len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')' {
		return s[1 : len(s)-1]
	}
	return s
}

// labeledViewSortParams converts the caller's SortBy/OrderBy into sortParams
// for getRawTasksForProjects. When the caller hasn't specified anything, the
// labeled-view default of priority DESC, due_date ASC, id DESC is used. The
// id term is included explicitly so getRawTasksForProjects does not append
// its own `id ASC` default which would conflict with the spec.
func labeledViewSortParams(taskCollection *TaskCollection) []*sortParam {
	if taskCollection != nil && len(taskCollection.SortBy) > 0 {
		out := make([]*sortParam, 0, len(taskCollection.SortBy))
		for i, sortBy := range taskCollection.SortBy {
			param := &sortParam{
				sortBy:  sortBy,
				orderBy: orderAscending,
			}
			if i < len(taskCollection.OrderBy) {
				param.orderBy = getSortOrderFromString(taskCollection.OrderBy[i])
			}
			if param.orderBy == orderInvalid {
				param.orderBy = orderAscending
			}
			out = append(out, param)
		}
		return out
	}
	return []*sortParam{
		{sortBy: taskPropertyPriority, orderBy: orderDescending},
		{sortBy: taskPropertyDueDate, orderBy: orderAscending},
		{sortBy: taskPropertyID, orderBy: orderDescending},
	}
}

// paginateLabeledGroupTasks returns the page slice (1-based) of a sorted task
// slice using labeledViewTasksPerGroup as the page size. Returns an empty
// (non-nil) slice when the page is past the end so the JSON field is `[]`
// rather than `null`.
func paginateLabeledGroupTasks(tasks []*Task, page int) []*Task {
	if page < 1 {
		page = 1
	}
	start := (page - 1) * labeledViewTasksPerGroup
	if start >= len(tasks) {
		return []*Task{}
	}
	end := start + labeledViewTasksPerGroup
	if end > len(tasks) {
		end = len(tasks)
	}
	return tasks[start:end]
}

// sortLabeledGroups orders the groups per BucketConfigurationSortBy. Empty
// string defaults to "task_count" per the spec.
func sortLabeledGroups(groups []*LabeledGroup, sortBy string) {
	switch sortBy {
	case "title_asc":
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Label.Title < groups[j].Label.Title
		})
	case "title_desc":
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Label.Title > groups[j].Label.Title
		})
	default: // "task_count" or empty/unknown
		sort.SliceStable(groups, func(i, j int) bool {
			if groups[i].TaskCount != groups[j].TaskCount {
				return groups[i].TaskCount > groups[j].TaskCount
			}
			return groups[i].Label.Title < groups[j].Label.Title
		})
	}
}
