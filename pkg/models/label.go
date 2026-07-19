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
	"strings"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/utils"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// Label represents a label
type Label struct {
	// The unique, numeric id of this label.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"label"`
	// The title of the label. You'll see this one on tasks associated with it.
	Title string `xorm:"varchar(250) not null" json:"title" valid:"runelength(1|250)" minLength:"1" maxLength:"250"`
	// The label description.
	Description string `xorm:"longtext null" json:"description"`
	// The color this label has in hex format.
	HexColor string `xorm:"varchar(6) null" json:"hex_color" valid:"runelength(0|7)" maxLength:"7"`

	CreatedByID int64 `xorm:"bigint not null" json:"-"`
	// The user who created this label
	CreatedBy *user.User `xorm:"-" json:"created_by"`

	// A timestamp when this label was created. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created"`
	// A timestamp when this label was last updated. You cannot change this value.
	Updated time.Time `xorm:"updated not null" json:"updated"`

	// ProjectIDFilter, when > 0, restricts ReadAll to labels actually used by
	// tasks in the given project. Bound from the `project_id` URL query
	// parameter by the generic web handler (same mechanism TaskCollection
	// uses for `category`). Not persisted — this is purely a query-binding
	// helper. Zero (or absent) preserves the existing user-scoped behaviour.
	ProjectIDFilter int64 `query:"project_id" json:"-" xorm:"-"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName makes a pretty table name
func (*Label) TableName() string {
	return "labels"
}

// Create creates a new label
// @Summary Create a label
// @Description Creates a new label.
// @tags labels
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param label body models.Label true "The label object"
// @Success 201 {object} models.Label "The created label object."
// @Failure 400 {object} web.HTTPError "Invalid label object provided."
// @Failure 500 {object} models.Message "Internal error"
// @Router /labels [put]
func (l *Label) Create(s *xorm.Session, a web.Auth) (err error) {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return
	}

	l.ID = 0
	l.HexColor = utils.NormalizeHex(l.HexColor)
	l.CreatedBy = u
	l.CreatedByID = u.ID

	_, err = s.Insert(l)
	return
}

// Update updates a label
// @Summary Update a label
// @Description Update an existing label. The user needs to be the creator of the label to be able to do this.
// @tags labels
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Label ID"
// @Param label body models.Label true "The label object"
// @Success 200 {object} models.Label "The created label object."
// @Failure 400 {object} web.HTTPError "Invalid label object provided."
// @Failure 403 {object} web.HTTPError "Not allowed to update the label."
// @Failure 404 {object} web.HTTPError "Label not found."
// @Failure 500 {object} models.Message "Internal error"
// @Router /labels/{id} [put]
func (l *Label) Update(s *xorm.Session, a web.Auth) (err error) {

	l.HexColor = utils.NormalizeHex(l.HexColor)

	_, err = s.
		ID(l.ID).
		Cols(
			"title",
			"description",
			"hex_color",
		).
		Update(l)
	if err != nil {
		return
	}

	err = l.ReadOne(s, a)
	return
}

// Delete deletes a label
// @Summary Delete a label
// @Description Delete an existing label. The user needs to be the creator of the label to be able to do this.
// @tags labels
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param id path int true "Label ID"
// @Success 200 {object} models.Label "The label was successfully deleted."
// @Failure 403 {object} web.HTTPError "Not allowed to delete the label."
// @Failure 404 {object} web.HTTPError "Label not found."
// @Failure 500 {object} models.Message "Internal error"
// @Router /labels/{id} [delete]
func (l *Label) Delete(s *xorm.Session, _ web.Auth) (err error) {
	_, err = s.ID(l.ID).Delete(&Label{})
	return err
}

// ReadAll gets all labels a user can use
// @Summary Get all labels a user has access to
// @Description Returns all labels which are either created by the user or associated with a task the user has at least read-access to.
// @tags labels
// @Accept json
// @Produce json
// @Param page query int false "The page number. Used for pagination. If not provided, the first page of results is returned."
// @Param per_page query int false "The maximum number of items per page. Note this parameter is limited by the configured maximum of items per page."
// @Param s query string false "Search labels by label text."
// @Param project_id query int false "When set to a project id, only returns labels that are used by at least one task in that project and owned by the current user. Without this parameter all user-accessible labels are returned."
// @Security JWTKeyAuth
// @Success 200 {array} models.Label "The labels"
// @Failure 500 {object} models.Message "Internal error"
// @Router /labels [get]
func (l *Label) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (ls interface{}, resultCount int, numberOfEntries int64, err error) {
	// Project-scoped mode: only labels owned by the current user that are
	// attached to at least one task in the given project. Strict by design —
	// labels created by other users are not surfaced here even if they are
	// attached to tasks the user can read. Falls through to the existing
	// user-scoped path when no project filter is supplied (preserving
	// backwards compatibility for /labels without the query param).
	if l.ProjectIDFilter > 0 {
		return getLabelsForProject(s, l.ProjectIDFilter, a, search, page, perPage)
	}
	return GetLabelsByTaskIDs(s, &LabelByTaskIDsOptions{
		Search:              []string{search},
		User:                a,
		Page:                page,
		PerPage:             perPage,
		GetUnusedLabels:     true,
		GroupByLabelIDsOnly: true,
		GetForUser:          true,
	})
}

// getLabelsForProject returns labels created by the user that are used by at
// least one task in the given project. Used by Label.ReadAll when the
// `project_id` query parameter is supplied.
//
// The join chain labels → label_tasks → tasks restricts the candidate set to
// labels actually attached to a task in this project; the created_by_id
// clause enforces user scope (other users' labels are never surfaced, even
// when they share the project). An optional ILIKE on labels.title mirrors
// the search behaviour of the unfiltered path.
func getLabelsForProject(s *xorm.Session, projectID int64, a web.Auth, search string, page int, perPage int) (ls []*LabelWithTaskID, resultCount int, numberOfEntries int64, err error) {
	userID := a.GetID()

	cond := builder.And(
		builder.Expr("tasks.project_id = ?", projectID),
		builder.Eq{"labels.created_by_id": userID},
	)

	if search = strings.TrimSpace(search); search != "" {
		cond = builder.And(cond, db.ILIKE("labels.title", search))
	}

	limit, start := getLimitFromPageIndex(page, perPage)

	query := s.Table("labels").
		Select("labels.*").
		Join("INNER", "label_tasks", "label_tasks.label_id = labels.id").
		Join("INNER", "tasks", "tasks.id = label_tasks.task_id").
		Where(cond).
		GroupBy("labels.id").
		OrderBy("labels.id ASC")
	if limit > 0 {
		query = query.Limit(limit, start)
	}

	var labels []*LabelWithTaskID
	if err = query.Find(&labels); err != nil {
		return
	}

	if len(labels) == 0 {
		return []*LabelWithTaskID{}, 0, 0, nil
	}

	// Resolve creators so CreatedBy matches the unfiltered path. With the
	// created_by_id filter this is always a single user, but we follow the
	// same shape as GetLabelsByTaskIDs to keep the response consistent.
	creatorIDs := make([]int64, 0, len(labels))
	for _, l := range labels {
		creatorIDs = append(creatorIDs, l.CreatedByID)
	}
	users := make(map[int64]*user.User)
	if err = s.In("id", creatorIDs).Find(&users); err != nil {
		return
	}
	for _, u := range users {
		u.Email = ""
	}
	for i, l := range labels {
		if c, ok := users[l.CreatedByID]; ok {
			labels[i].CreatedBy = c
		}
	}

	numberOfEntries, err = s.Table("labels").
		Select("count(DISTINCT labels.id)").
		Join("INNER", "label_tasks", "label_tasks.label_id = labels.id").
		Join("INNER", "tasks", "tasks.id = label_tasks.task_id").
		Where(cond).
		Count(&Label{})
	if err != nil {
		return
	}

	return labels, len(labels), numberOfEntries, nil
}

// ReadOne gets one label
// @Summary Gets one label
// @Description Returns one label by its ID.
// @tags labels
// @Accept json
// @Produce json
// @Param id path int true "Label ID"
// @Security JWTKeyAuth
// @Success 200 {object} models.Label "The label"
// @Failure 403 {object} web.HTTPError "The user does not have access to the label"
// @Failure 404 {object} web.HTTPError "Label not found"
// @Failure 500 {object} models.Message "Internal error"
// @Router /labels/{id} [get]
func (l *Label) ReadOne(s *xorm.Session, _ web.Auth) (err error) {
	label, err := getLabelByIDSimple(s, l.ID)
	if err != nil {
		return
	}
	*l = *label

	u, err := user.GetUserByID(s, l.CreatedByID)
	if err != nil {
		return
	}

	l.CreatedBy = u
	return
}

func getLabelByIDSimple(s *xorm.Session, labelID int64) (*Label, error) {
	return GetLabelSimple(s, &Label{ID: labelID})
}

func GetLabelSimple(s *xorm.Session, l *Label) (*Label, error) {
	exists, err := s.Get(l)
	if err != nil {
		return l, err
	}
	if !exists {
		return &Label{}, ErrLabelDoesNotExist{l.ID}
	}
	return l, err
}
