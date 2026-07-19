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

package migration

import (
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// projectView20260719025313 mirrors the ProjectView model in pkg/models/project_view.go.
// Only the new column is declared so xorm.Sync can add it without touching the rest.
type projectView20260719025313 struct {
	ID                        int64  `xorm:"autoincr not null unique pk" json:"id" param:"view"`
	BucketConfigurationSortBy string `xorm:"varchar(50) null" json:"bucket_configuration_sort_by" valid:"-"`
}

func (projectView20260719025313) TableName() string {
	return "project_views"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260719025313",
		Description: "add bucket_configuration_sort_by to project_views and seed a labeled view for each project",
		Migrate: func(tx *xorm.Engine) error {
			// 1. Add the new column. Sync handles MySQL/PostgreSQL/SQLite safely.
			if err := tx.Sync(projectView20260719025313{}); err != nil {
				return err
			}

			// 2. Backfill: for each existing project that doesn't yet have a
			// labeled view (view_kind = 4), insert one with the default filter
			// and sort. The filter column is stored as JSON
			// (see migration 20241118123644). CURRENT_TIMESTAMP is portable
			// across MySQL, PostgreSQL and SQLite.
			//
			// view_kind 4 == ProjectViewKindLabeled (iota after Kanban = 3).
			// bucket_configuration_mode 0 == BucketConfigurationModeNone.
			// position 500 matches the 5th default view in
			// CreateDefaultViewsForProject (100/200/300/400/500).
			_, err := tx.Exec(`
INSERT INTO project_views
    (project_id, title, view_kind, filter, position, bucket_configuration_mode, bucket_configuration_sort_by, created, updated)
SELECT
    p.id, 'Labeled', 4, '{"filter":"done = false"}', 500, 0, 'task_count', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM project_views pv
    WHERE pv.project_id = p.id AND pv.view_kind = 4
)
`)
			return err
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
