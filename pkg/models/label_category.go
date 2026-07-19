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
	"errors"
	"net/http"
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

// LabelCategory groups labels within a project so that the Labeled view can
// filter columns by category. Categories are scoped to a single project and
// relate to labels via LabelCategoryMember.
type LabelCategory struct {
	// The unique, numeric id of this category.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"category"`
	// The title of this category.
	Title string `xorm:"varchar(250) not null" json:"title" valid:"required,runelength(1|250)"`
	// The project this category belongs to.
	ProjectID int64 `xorm:"bigint not null index" json:"project_id" param:"project"`
	// The position of this category when displayed in the cloud. See Position on other entities for usage.
	Position float64 `xorm:"double null" json:"position"`
	// The ID of the user who created this category.
	CreatedByID int64 `xorm:"bigint not null" json:"-"`
	// The user who created this category.
	CreatedBy *user.User `xorm:"-" json:"created_by"`
	// A timestamp when this category was created. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created"`
	// A timestamp when this category was last updated. You cannot change this value.
	Updated time.Time `xorm:"updated not null" json:"updated"`

	// Labels belonging to this category. Populated at read time, not persisted on this struct.
	Labels []*Label `xorm:"-" json:"labels,omitempty"`
	// The number of labels in this category. Populated at read time.
	LabelCount int64 `xorm:"-" json:"label_count"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for label categories.
func (*LabelCategory) TableName() string {
	return "label_categories"
}

// LabelCategoryMember is the join row between a LabelCategory and a Label.
// The composite unique index on (LabelCategoryID, LabelID) prevents a label
// from being added to the same category twice. A label may still belong to
// multiple categories and a category may hold many labels.
type LabelCategoryMember struct {
	ID              int64     `xorm:"bigint autoincr not null unique pk"`
	LabelCategoryID int64     `xorm:"bigint not null unique(cat_label) index" json:"label_category_id"`
	LabelID         int64     `xorm:"bigint not null unique(cat_label) index" json:"label_id"`
	Created         time.Time `xorm:"created not null" json:"created"`
}

// TableName returns the table name for label category members.
func (*LabelCategoryMember) TableName() string {
	return "label_category_members"
}

// ErrLabelCategoryDoesNotExist is returned when a LabelCategory lookup misses.
type ErrLabelCategoryDoesNotExist struct {
	CategoryID int64
}

// IsErrLabelCategoryDoesNotExist checks if an error is an ErrLabelCategoryDoesNotExist.
func IsErrLabelCategoryDoesNotExist(err error) bool {
	var target ErrLabelCategoryDoesNotExist
	return errors.As(err, &target)
}

func (err ErrLabelCategoryDoesNotExist) Error() string {
	return "Label category does not exist"
}

// ErrCodeLabelCategoryDoesNotExist holds the unique world-error code of this error.
const ErrCodeLabelCategoryDoesNotExist = 17008

// HTTPError holds the http error description.
func (err ErrLabelCategoryDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrCodeLabelCategoryDoesNotExist,
		Message:  "This label category does not exist.",
	}
}

// getLabelCategoryByID fetches a raw category row by id. It does not preload
// any virtual fields.
func getLabelCategoryByID(s *xorm.Session, id int64) (*LabelCategory, error) {
	cat := &LabelCategory{}
	exists, err := s.Where("id = ?", id).NoAutoCondition().Get(cat)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrLabelCategoryDoesNotExist{CategoryID: id}
	}
	return cat, nil
}

// preloadLabelCategoryLabels populates Labels and LabelCount for each category
// in the slice. Labels are loaded in bulk to avoid N+1 queries.
func preloadLabelCategoryLabels(s *xorm.Session, cats []*LabelCategory) error {
	if len(cats) == 0 {
		return nil
	}

	catIDs := make([]int64, 0, len(cats))
	for _, c := range cats {
		catIDs = append(catIDs, c.ID)
	}

	members := []*LabelCategoryMember{}
	err := s.In("label_category_id", catIDs).OrderBy("label_category_id asc, id asc").Find(&members)
	if err != nil {
		return err
	}

	labelIDsByCat := make(map[int64][]int64)
	allLabelIDs := make(map[int64]struct{})
	for _, m := range members {
		labelIDsByCat[m.LabelCategoryID] = append(labelIDsByCat[m.LabelCategoryID], m.LabelID)
		allLabelIDs[m.LabelID] = struct{}{}
	}

	labelsByID := make(map[int64]*Label)
	if len(allLabelIDs) > 0 {
		ids := make([]int64, 0, len(allLabelIDs))
		for id := range allLabelIDs {
			ids = append(ids, id)
		}
		found := []*Label{}
		err = s.In("id", ids).Find(&found)
		if err != nil {
			return err
		}
		for _, l := range found {
			labelsByID[l.ID] = l
		}
	}

	for _, c := range cats {
		ids := labelIDsByCat[c.ID]
		c.Labels = make([]*Label, 0, len(ids))
		for _, lid := range ids {
			if l, ok := labelsByID[lid]; ok {
				c.Labels = append(c.Labels, l)
			}
		}
		c.LabelCount = int64(len(c.Labels))
	}
	return nil
}

// preloadLabelCategoryCreators populates CreatedBy for each category in the
// slice using a single bulk lookup. Supports both real users and link shares.
func preloadLabelCategoryCreators(s *xorm.Session, cats []*LabelCategory) error {
	if len(cats) == 0 {
		return nil
	}
	userIDs := make([]int64, 0, len(cats))
	for _, c := range cats {
		userIDs = append(userIDs, c.CreatedByID)
	}
	users, err := getUsersOrLinkSharesFromIDs(s, userIDs)
	if err != nil {
		return err
	}
	for _, c := range cats {
		if u, has := users[c.CreatedByID]; has {
			c.CreatedBy = u
		}
	}
	return nil
}

// replaceLabelCategoryMembers deletes every existing member row for the
// category and inserts the provided label IDs. Passing an empty (or nil)
// slice clears membership.
func replaceLabelCategoryMembers(s *xorm.Session, categoryID int64, labelIDs []int64) error {
	_, err := s.Where("label_category_id = ?", categoryID).Delete(&LabelCategoryMember{})
	if err != nil {
		return err
	}
	if len(labelIDs) == 0 {
		return nil
	}
	members := make([]*LabelCategoryMember, 0, len(labelIDs))
	for _, lid := range labelIDs {
		members = append(members, &LabelCategoryMember{
			LabelCategoryID: categoryID,
			LabelID:         lid,
		})
	}
	_, err = s.Insert(&members)
	return err
}

// ReadAll returns all label categories for the project set on the receiver.
// The project id comes from the :project URL param via the param binder.
//
// @Summary Get all label categories for a project
// @Description Returns all label categories for a specific project, with their labels preloaded.
// @tags label
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Param page query int false "The page number. Used for pagination. If not provided, the first page of results is returned."
// @Param per_page query int false "The maximum number of items per page. Note this parameter is limited by the configured maximum of items per page."
// @Param s query string false "Search categories by title."
// @Success 200 {array} models.LabelCategory "The label categories"
// @Failure 403 {object} web.HTTPError "The user does not have access to the project"
// @Failure 500 {object} models.Message "Internal error"
// @Router /projects/{project}/label-categories [get]
func (lc *LabelCategory) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	pp := &Project{ID: lc.ProjectID}
	can, _, err := pp.CanRead(s, a)
	if err != nil {
		return nil, 0, 0, err
	}
	if !can {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	query := s.Where("project_id = ?", lc.ProjectID)
	if search != "" {
		query = query.And("title LIKE ?", "%"+search+"%")
	}
	limit, start := getLimitFromPageIndex(page, perPage)
	if limit > 0 {
		query = query.Limit(limit, start)
	}

	cats := []*LabelCategory{}
	err = query.OrderBy("position asc, id asc").Find(&cats)
	if err != nil {
		return nil, 0, 0, err
	}

	err = preloadLabelCategoryLabels(s, cats)
	if err != nil {
		return nil, 0, 0, err
	}

	err = preloadLabelCategoryCreators(s, cats)
	if err != nil {
		return nil, 0, 0, err
	}

	total, err := s.Where("project_id = ?", lc.ProjectID).Count(&LabelCategory{})
	if err != nil {
		return nil, 0, 0, err
	}

	return cats, len(cats), total, nil
}

// ReadOne fetches a single category by ID and project, preloading its labels
// and creator.
//
// @Summary Get one label category
// @Description Returns a label category by its ID.
// @tags label
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Param category path int true "Label Category ID"
// @Success 200 {object} models.LabelCategory "The label category"
// @Failure 403 {object} web.HTTPError "The user does not have access to this category"
// @Failure 404 {object} web.HTTPError "The category does not exist"
// @Failure 500 {object} models.Message "Internal error"
// @Router /projects/{project}/label-categories/{category} [get]
func (lc *LabelCategory) ReadOne(s *xorm.Session, _ web.Auth) error {
	cat, err := getLabelCategoryByID(s, lc.ID)
	if err != nil {
		return err
	}
	if cat.ProjectID != lc.ProjectID {
		return ErrLabelCategoryDoesNotExist{CategoryID: lc.ID}
	}
	*lc = *cat

	cats := []*LabelCategory{lc}
	if err = preloadLabelCategoryLabels(s, cats); err != nil {
		return err
	}
	return preloadLabelCategoryCreators(s, cats)
}

// Create persists a new category. The CreatedByID is taken from the auth user.
// When Labels is provided in the request body, join rows are inserted for each
// label.
//
// @Summary Create a label category
// @Description Create a new label category in a project.
// @tags label
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Param category body models.LabelCategory true "The label category to create"
// @Success 200 {object} models.LabelCategory "The created label category"
// @Failure 400 {object} web.HTTPError "Invalid label category object provided."
// @Failure 403 {object} web.HTTPError "The user does not have write access to the project"
// @Failure 500 {object} models.Message "Internal error"
// @Router /projects/{project}/label-categories [put]
func (lc *LabelCategory) Create(s *xorm.Session, a web.Auth) error {
	creator, err := GetUserOrLinkShareUser(s, a)
	if err != nil {
		return err
	}
	lc.CreatedBy = creator
	lc.CreatedByID = creator.ID
	lc.ID = 0

	_, err = s.Insert(lc)
	if err != nil {
		return err
	}

	labelIDs := lc.labelIDsFromLabels()
	if len(labelIDs) > 0 {
		if err = replaceLabelCategoryMembers(s, lc.ID, labelIDs); err != nil {
			return err
		}
	}

	// Refresh virtual fields so the returned object mirrors what ReadOne
	// would produce, keeping the API consistent for callers.
	cats := []*LabelCategory{lc}
	if err = preloadLabelCategoryLabels(s, cats); err != nil {
		return err
	}
	return preloadLabelCategoryCreators(s, cats)
}

// Update modifies the title/position of an existing category and, when Labels
// is provided in the request body, fully replaces the member set.
//
// @Summary Update a label category
// @Description Updates an existing label category. When labels are provided, the full set is replaced.
// @tags label
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Param category path int true "Label Category ID"
// @Param body body models.LabelCategory true "The label category with updated values."
// @Success 200 {object} models.LabelCategory "The updated label category"
// @Failure 400 {object} web.HTTPError "Invalid label category object provided."
// @Failure 403 {object} web.HTTPError "The user does not have write access to the project"
// @Failure 404 {object} web.HTTPError "The category does not exist"
// @Failure 500 {object} models.Message "Internal error"
// @Router /projects/{project}/label-categories/{category} [post]
func (lc *LabelCategory) Update(s *xorm.Session, _ web.Auth) error {
	existing, err := getLabelCategoryByID(s, lc.ID)
	if err != nil {
		return err
	}
	if existing.ProjectID != lc.ProjectID {
		return ErrLabelCategoryDoesNotExist{CategoryID: lc.ID}
	}

	_, err = s.
		ID(lc.ID).
		Cols("title", "position").
		Update(lc)
	if err != nil {
		return err
	}

	// Labels is a virtual field, so its presence in the request body
	// distinguishes "client wants to replace" from "client did not specify".
	// nil == unchanged, empty slice == clear membership, non-empty == replace.
	if lc.Labels != nil {
		if err = replaceLabelCategoryMembers(s, lc.ID, lc.labelIDsFromLabels()); err != nil {
			return err
		}
	}

	updated, err := getLabelCategoryByID(s, lc.ID)
	if err != nil {
		return err
	}
	lc.Created = updated.Created
	lc.CreatedByID = updated.CreatedByID
	lc.Updated = updated.Updated

	cats := []*LabelCategory{lc}
	if err = preloadLabelCategoryLabels(s, cats); err != nil {
		return err
	}
	return preloadLabelCategoryCreators(s, cats)
}

// Delete removes a category and cascades the deletion to its member rows.
//
// @Summary Delete a label category
// @Description Deletes an existing label category. Member rows are removed as well.
// @tags label
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param project path int true "Project ID"
// @Param category path int true "Label Category ID"
// @Success 200 {object} models.Message "The category was successfully deleted."
// @Failure 403 {object} web.HTTPError "The user does not have write access to the project"
// @Failure 404 {object} web.HTTPError "The category does not exist"
// @Failure 500 {object} models.Message "Internal error"
// @Router /projects/{project}/label-categories/{category} [delete]
func (lc *LabelCategory) Delete(s *xorm.Session, _ web.Auth) error {
	existing, err := getLabelCategoryByID(s, lc.ID)
	if err != nil {
		return err
	}
	if existing.ProjectID != lc.ProjectID {
		return ErrLabelCategoryDoesNotExist{CategoryID: lc.ID}
	}

	_, err = s.Where("label_category_id = ?", lc.ID).Delete(&LabelCategoryMember{})
	if err != nil {
		return err
	}

	_, err = s.ID(lc.ID).Delete(&LabelCategory{})
	return err
}

// labelIDsFromLabels collects the IDs of the Labels set on the receiver.
// Resolves any label that has a zero ID by its title within the project so a
// caller can submit either existing ids or new titles.
func (lc *LabelCategory) labelIDsFromLabels() []int64 {
	if len(lc.Labels) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(lc.Labels))
	for _, l := range lc.Labels {
		if l != nil && l.ID != 0 {
			ids = append(ids, l.ID)
		}
	}
	return ids
}

// canDoCategory returns whether the auth user has the requested project
// permission for the category's project. It looks up the category row when
// only the id is set (the Can*/CRUD methods are invoked before the body is
// fetched). Project access is delegated to Project.CanWrite / CanRead so the
// full permission chain (teams, link shares, hierarchy) is reused.
func (lc *LabelCategory) canDoCategory(s *xorm.Session, a web.Auth, wantWrite bool) (bool, error) {
	projectID := lc.ProjectID
	if projectID == 0 {
		existing, err := getLabelCategoryByID(s, lc.ID)
		if err != nil {
			return false, err
		}
		projectID = existing.ProjectID
	}

	p := &Project{ID: projectID}
	if wantWrite {
		return p.CanWrite(s, a)
	}
	can, _, err := p.CanRead(s, a)
	return can, err
}

// CanCreate checks whether the user has write access to the project set on
// the receiver. Only ProjectID is populated at create time.
func (lc *LabelCategory) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	return lc.canDoCategory(s, a, true)
}

// CanRead checks whether the user has read access to the category's project.
func (lc *LabelCategory) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	can, err := lc.canDoCategory(s, a, false)
	if err != nil {
		return false, 0, err
	}
	return can, 0, nil
}

// CanUpdate checks whether the user has write access to the category's project.
func (lc *LabelCategory) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return lc.canDoCategory(s, a, true)
}

// CanDelete checks whether the user has write access to the category's project.
func (lc *LabelCategory) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return lc.canDoCategory(s, a, true)
}
