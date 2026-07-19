import AbstractService from '@/services/abstractService'
import {LabeledViewResponseModel} from '@/models/labeledGroup'

import type {ILabeledViewResponse} from '@/modelTypes/ILabeledView'

export default class LabeledViewService extends AbstractService<ILabeledViewResponse> {
	constructor() {
		super({
			// Reuses the same task-collection endpoint as Kanban/List/Table,
			// but the backend returns a LabeledViewResponse object (not an
			// array) when the view's kind is "labeled". Because getAll()
			// expects an array, we route through get()/getM() instead.
			get: '/projects/{projectId}/views/{viewId}/tasks',
		})
	}

	modelFactory(data: Partial<ILabeledViewResponse>) {
		return new LabeledViewResponseModel(data)
	}
}
