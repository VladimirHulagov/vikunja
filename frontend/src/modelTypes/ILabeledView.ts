import type {IAbstract} from '@/modelTypes/IAbstract'
import type {ILabel} from '@/modelTypes/ILabel'
import type {ITask} from '@/modelTypes/ITask'

export interface ILabeledGroup extends IAbstract {
	label: ILabel | null
	taskCount: number
	tasks: ITask[]
}

export interface ILabeledViewResponse extends IAbstract {
	groups: ILabeledGroup[]
	untaggedGroup: ILabeledGroup | null
}
