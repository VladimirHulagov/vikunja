import AbstractService from '@/services/abstractService'
import LabelCategoryModel from '@/models/labelCategory'
import type {ILabelCategory} from '@/modelTypes/ILabelCategory'

export default class LabelCategoryService extends AbstractService<ILabelCategory> {
	constructor() {
		super({
			getAll: '/projects/{projectId}/label-categories',
			get: '/projects/{projectId}/label-categories/{id}',
			create: '/projects/{projectId}/label-categories',
			update: '/projects/{projectId}/label-categories/{id}',
			delete: '/projects/{projectId}/label-categories/{id}',
		})
	}

	modelFactory(data: Partial<ILabelCategory>): LabelCategoryModel {
		return new LabelCategoryModel(data)
	}
}
