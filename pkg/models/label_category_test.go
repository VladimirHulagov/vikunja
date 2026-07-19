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
	"xorm.io/builder"
)

func TestLabelCategory_Create(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		owner := &user.User{ID: 100}
		lc := &LabelCategory{
			Title:     "New Category",
			ProjectID: 100,
			Position:  300,
		}
		err := lc.Create(s, owner)
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		assert.NotZero(t, lc.ID)
		assert.Equal(t, owner.ID, lc.CreatedByID)
		assert.NotNil(t, lc.CreatedBy)
		assert.Equal(t, owner.ID, lc.CreatedBy.ID)
		assert.Empty(t, lc.Labels)
		assert.Zero(t, lc.LabelCount)

		db.AssertExists(t, "label_categories", map[string]interface{}{
			"id":            lc.ID,
			"title":         "New Category",
			"project_id":    100,
			"created_by_id": owner.ID,
		}, false)
	})
}

func TestLabelCategory_Create_WithLabels(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	owner := &user.User{ID: 100}
	lc := &LabelCategory{
		Title:     "With Labels",
		ProjectID: 100,
		Labels: []*Label{
			{ID: 100},
			{ID: 101},
		},
	}
	err := lc.Create(s, owner)
	require.NoError(t, err)
	require.NoError(t, s.Commit())

	assert.Len(t, lc.Labels, 2)
	assert.Equal(t, int64(2), lc.LabelCount)

	// Member rows should exist for both labels.
	db.AssertCount(t, "label_category_members", builder.Eq{
		"label_category_id": lc.ID,
	}, 2)
	db.AssertExists(t, "label_category_members", map[string]interface{}{
		"label_category_id": lc.ID,
		"label_id":          100,
	}, false)
	db.AssertExists(t, "label_category_members", map[string]interface{}{
		"label_category_id": lc.ID,
		"label_id":          101,
	}, false)
}

func TestLabelCategory_ReadAll(t *testing.T) {
	t.Run("returns only this project's categories with labels preloaded", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		owner := &user.User{ID: 100}
		lc := &LabelCategory{ProjectID: 100}
		result, count, total, err := lc.ReadAll(s, owner, "", 0, 0)
		require.NoError(t, err)

		cats, ok := result.([]*LabelCategory)
		require.True(t, ok)
		assert.Len(t, cats, 2)
		assert.Equal(t, 2, count)
		assert.Equal(t, int64(2), total)

		// Fixture category 1 is "Rooms" with labels 100 + 101.
		var rooms *LabelCategory
		var workTypes *LabelCategory
		for _, c := range cats {
			switch c.Title {
			case "Rooms":
				rooms = c
			case "Work Types":
				workTypes = c
			}
		}
		require.NotNil(t, rooms)
		require.NotNil(t, workTypes)

		assert.Len(t, rooms.Labels, 2)
		assert.Equal(t, int64(2), rooms.LabelCount)
		assert.Len(t, workTypes.Labels, 1)
		assert.Equal(t, int64(1), workTypes.LabelCount)

		// CreatedBy should be preloaded.
		assert.NotNil(t, rooms.CreatedBy)
		assert.Equal(t, int64(100), rooms.CreatedBy.ID)
	})
	t.Run("user without read access is forbidden", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// User 1 has no access to project 100.
		outsider := &user.User{ID: 1}
		lc := &LabelCategory{ProjectID: 100}
		_, _, _, err := lc.ReadAll(s, outsider, "", 0, 0)
		require.Error(t, err)
		assert.True(t, IsErrGenericForbidden(err))
	})
}

func TestLabelCategory_ReadOne(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	owner := &user.User{ID: 100}
	lc := &LabelCategory{ID: 1, ProjectID: 100}
	err := lc.ReadOne(s, owner)
	require.NoError(t, err)

	assert.Equal(t, "Rooms", lc.Title)
	assert.Len(t, lc.Labels, 2)
	assert.Equal(t, int64(2), lc.LabelCount)
	assert.NotNil(t, lc.CreatedBy)
	assert.Equal(t, int64(100), lc.CreatedBy.ID)
}

func TestLabelCategory_Update_ReplacesLabels(t *testing.T) {
	t.Run("replaces full label set", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		owner := &user.User{ID: 100}
		lc := &LabelCategory{
			ID:        1,
			ProjectID: 100,
			Title:     "Rooms (renamed)",
			Position:  150,
			// Labels is non-nil → membership is fully replaced. The previous
			// set (labels 100, 101) must be removed and only 102 remain.
			Labels: []*Label{{ID: 102}},
		}
		err := lc.Update(s, owner)
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		assert.Equal(t, "Rooms (renamed)", lc.Title)
		assert.Len(t, lc.Labels, 1)
		assert.Equal(t, int64(102), lc.Labels[0].ID)

		db.AssertCount(t, "label_category_members", builder.Eq{
			"label_category_id": lc.ID,
		}, 1)
		db.AssertMissing(t, "label_category_members", map[string]interface{}{
			"label_category_id": lc.ID,
			"label_id":          100,
		})
		db.AssertMissing(t, "label_category_members", map[string]interface{}{
			"label_category_id": lc.ID,
			"label_id":          101,
		})
		db.AssertExists(t, "label_category_members", map[string]interface{}{
			"label_category_id": lc.ID,
			"label_id":          102,
		}, false)
	})
	t.Run("empty slice clears membership", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		owner := &user.User{ID: 100}
		lc := &LabelCategory{
			ID:        1,
			ProjectID: 100,
			Title:     "Rooms",
			Labels:    []*Label{},
		}
		err := lc.Update(s, owner)
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		assert.Empty(t, lc.Labels)
		db.AssertCount(t, "label_category_members", builder.Eq{
			"label_category_id": lc.ID,
		}, 0)
	})
	t.Run("nil labels leaves membership untouched", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		owner := &user.User{ID: 100}
		lc := &LabelCategory{
			ID:        1,
			ProjectID: 100,
			Title:     "Rooms",
			Labels:    nil,
		}
		err := lc.Update(s, owner)
		require.NoError(t, err)
		require.NoError(t, s.Commit())

		db.AssertCount(t, "label_category_members", builder.Eq{
			"label_category_id": lc.ID,
		}, 2)
	})
}

func TestLabelCategory_Delete_CascadesMembers(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	owner := &user.User{ID: 100}
	lc := &LabelCategory{ID: 1, ProjectID: 100}
	err := lc.Delete(s, owner)
	require.NoError(t, err)
	require.NoError(t, s.Commit())

	db.AssertMissing(t, "label_categories", map[string]interface{}{
		"id": 1,
	})
	db.AssertCount(t, "label_category_members", builder.Eq{
		"label_category_id": 1,
	}, 0)
}

func TestLabelCategory_CanCreate_NoWriteAccess(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	// User 1 has no access to project 100.
	outsider := &user.User{ID: 1}
	lc := &LabelCategory{ProjectID: 100}
	can, err := lc.CanCreate(s, outsider)
	require.NoError(t, err)
	assert.False(t, can)
}

func TestLabelCategory_CanCreate_Owner(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	owner := &user.User{ID: 100}
	lc := &LabelCategory{ProjectID: 100}
	can, err := lc.CanCreate(s, owner)
	require.NoError(t, err)
	assert.True(t, can)
}

func TestLabelCategory_CanRead_NoReadAccess(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	outsider := &user.User{ID: 1}
	lc := &LabelCategory{ID: 1, ProjectID: 100}
	can, _, err := lc.CanRead(s, outsider)
	require.NoError(t, err)
	assert.False(t, can)
}

func TestLabelCategory_CanRead_Owner(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	owner := &user.User{ID: 100}
	lc := &LabelCategory{ID: 1, ProjectID: 100}
	can, _, err := lc.CanRead(s, owner)
	require.NoError(t, err)
	assert.True(t, can)
}

func TestLabelCategory_CanUpdate_NoWriteAccess(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	outsider := &user.User{ID: 1}
	lc := &LabelCategory{ID: 1, ProjectID: 100}
	can, err := lc.CanUpdate(s, outsider)
	require.NoError(t, err)
	assert.False(t, can)
}

func TestLabelCategory_CanDelete_NoWriteAccess(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	outsider := &user.User{ID: 1}
	lc := &LabelCategory{ID: 1, ProjectID: 100}
	can, err := lc.CanDelete(s, outsider)
	require.NoError(t, err)
	assert.False(t, can)
}
