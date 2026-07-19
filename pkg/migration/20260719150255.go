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
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// labelCategory20260719150255 mirrors pkg/models.LabelCategory for table
// creation. Field names, types and xorm tags must stay in sync with the model
// so tx.Sync creates the schema exactly as the ORM expects it at runtime.
type labelCategory20260719150255 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	Title       string    `xorm:"varchar(250) not null"`
	ProjectID   int64     `xorm:"bigint not null index"`
	Position    float64   `xorm:"double null"`
	CreatedByID int64     `xorm:"bigint not null"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (labelCategory20260719150255) TableName() string {
	return "label_categories"
}

// labelCategoryMember20260719150255 mirrors pkg/models.LabelCategoryMember.
// The unique(cat_label) composite index prevents a label from being added to
// the same category twice.
type labelCategoryMember20260719150255 struct {
	ID              int64     `xorm:"bigint autoincr not null unique pk"`
	LabelCategoryID int64     `xorm:"bigint not null unique(cat_label) index"`
	LabelID         int64     `xorm:"bigint not null unique(cat_label) index"`
	Created         time.Time `xorm:"created not null"`
}

func (labelCategoryMember20260719150255) TableName() string {
	return "label_category_members"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260719150255",
		Description: "create label_categories and label_category_members tables",
		Migrate: func(tx *xorm.Engine) error {
			// tx.Sync handles CREATE TABLE / indexes across MySQL, PostgreSQL
			// and SQLite. Each call must surface its error so a failed DDL
			// rolls the migration back instead of reporting success.
			if err := tx.Sync(labelCategory20260719150255{}); err != nil {
				return err
			}
			if err := tx.Sync(labelCategoryMember20260719150255{}); err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *xorm.Engine) error {
			// Drop the dependent (members) table first so the categories
			// table can be dropped without dangling FK references on
			// dialects that enforce them.
			if err := tx.DropTables(labelCategoryMember20260719150255{}); err != nil {
				return err
			}
			return tx.DropTables(labelCategory20260719150255{})
		},
	})
}
