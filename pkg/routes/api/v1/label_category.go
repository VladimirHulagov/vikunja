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

package v1

import (
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/labstack/echo/v5"
)

// RegisterLabelCategoryRoutes wires up the generic web handler for
// LabelCategory under /projects/:project/label-categories. The handler takes
// care of binding URL params, invoking the Can* permission methods on the
// model, calling the matching CRUD method and rendering the response.
func RegisterLabelCategoryRoutes(g *echo.Group) {
	labelCategoryHandler := &handler.WebHandler{
		EmptyStruct: func() handler.CObject {
			return &models.LabelCategory{}
		},
	}
	g.GET("/projects/:project/label-categories", labelCategoryHandler.ReadAllWeb)
	g.GET("/projects/:project/label-categories/:category", labelCategoryHandler.ReadOneWeb)
	g.PUT("/projects/:project/label-categories", labelCategoryHandler.CreateWeb)
	g.POST("/projects/:project/label-categories/:category", labelCategoryHandler.UpdateWeb)
	g.DELETE("/projects/:project/label-categories/:category", labelCategoryHandler.DeleteWeb)
}
