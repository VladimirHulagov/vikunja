import AbstractModel from './abstractModel'
import UserModel from './user'
import LabelModel from './label'

import type {ILabelCategory} from '@/modelTypes/ILabelCategory'
import type {ILabel} from '@/modelTypes/ILabel'
import type {IUser} from '@/modelTypes/IUser'

export default class LabelCategoryModel extends AbstractModel<ILabelCategory> implements ILabelCategory {
	id = 0
	title = ''
	projectId = 0
	position = 0
	labelCount = 0

	createdBy: IUser = {} as IUser
	created: Date = new Date()
	updated: Date = new Date()

	labels: ILabel[] = []

	constructor(data: Partial<ILabelCategory> = {}) {
		super()
		this.assignData(data)

		this.labels = (this.labels ?? []).map(l => new LabelModel(l))

		this.createdBy = new UserModel(this.createdBy ?? {})

		this.created = new Date(this.created)
		this.updated = new Date(this.updated)
	}
}
