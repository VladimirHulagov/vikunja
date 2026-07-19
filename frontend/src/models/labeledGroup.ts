import AbstractModel from './abstractModel'
import LabelModel from './label'
import TaskModel from './task'

import type {ILabeledGroup, ILabeledViewResponse} from '@/modelTypes/ILabeledView'
import type {ILabel} from '@/modelTypes/ILabel'
import type {ITask} from '@/modelTypes/ITask'

export default class LabeledGroupModel extends AbstractModel<ILabeledGroup> implements ILabeledGroup {
	label: ILabel | null = null
	taskCount = 0
	tasks: ITask[] = []

	constructor(data: Partial<ILabeledGroup> = {}) {
		super()
		this.assignData(data)

		this.tasks = this.tasks.map(t => new TaskModel(t))

		if (this.label !== null) {
			this.label = new LabelModel(this.label)
		}
	}
}

export class LabeledViewResponseModel extends AbstractModel<ILabeledViewResponse> implements ILabeledViewResponse {
	groups: ILabeledGroup[] = []
	untaggedGroup: ILabeledGroup | null = null

	constructor(data: Partial<ILabeledViewResponse> = {}) {
		super()
		this.assignData(data)

		this.groups = this.groups.map(g => new LabeledGroupModel(g))

		if (this.untaggedGroup !== null && typeof this.untaggedGroup !== 'undefined') {
			this.untaggedGroup = new LabeledGroupModel(this.untaggedGroup)
		} else {
			this.untaggedGroup = null
		}
	}
}
