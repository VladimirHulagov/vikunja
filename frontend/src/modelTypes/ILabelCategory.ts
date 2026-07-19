import type {IAbstract} from './IAbstract'
import type {ILabel} from './ILabel'
import type {IUser} from './IUser'

export interface ILabelCategory extends IAbstract {
	id: number
	title: string
	projectId: number
	position: number
	createdBy: IUser
	created: Date
	updated: Date
	labels?: ILabel[]
	labelCount?: number
}
