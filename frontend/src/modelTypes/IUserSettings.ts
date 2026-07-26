import type {IAbstract} from './IAbstract'
import type {IProject} from './IProject'
import type {ITaskReminder} from '@/modelTypes/ITaskReminder'
import type {PrefixMode} from '@/modules/quickAddMagic'
import type {BasicColorSchema} from '@vueuse/core'
import type {SupportedLocale} from '@/i18n'
import type {DefaultProjectViewKind} from '@/modelTypes/IProjectView'
import type {Priority} from '@/constants/priorities'
import type {DateDisplay} from '@/constants/dateDisplay'
import type {TimeFormat} from '@/constants/timeFormat'
import type {IRelationKind} from '@/types/IRelationKind'

export interface IFrontendSettings {
	playSoundWhenDone: boolean
	quickAddMagicMode: PrefixMode
	colorSchema: BasicColorSchema
	allowIconChanges: boolean
	filterIdUsedOnOverview: IProject['id'] | null
	defaultView?: DefaultProjectViewKind
	minimumPriority?: Priority
	dateDisplay: DateDisplay
	timeFormat: TimeFormat
	defaultTaskRelationType: IRelationKind
	backgroundBrightness: number | null
	alwaysShowBucketTaskCount: boolean
	showLastViewed: boolean
	sidebarWidth: number | null
	commentSortOrder: 'asc' | 'desc'
	desktopQuickEntryShortcut: string
	quickAddDefaultReminders: ITaskReminder[]
	// User's global preference for hiding done tasks in project views.
	// The Hide done toggle in ProjectWrapper reads/writes this; on each
	// project-view mount it forces `done = false` into the URL filter
	// when the preference is true. Optional because pre-existing users
	// don't have it set yet — defaults to true (matches Labeled-view
	// default behavior).
	hideDoneTasks?: boolean
}

export interface IExtraSettingsLink {
	text: string
	url: string
}

export interface IExtraSettingsLinks {
	[key: string]: IExtraSettingsLink
}

export interface IUserSettings extends IAbstract {
	name: string
	emailRemindersEnabled: boolean
	discoverableByName: boolean
	discoverableByEmail: boolean
	overdueTasksRemindersEnabled: boolean
	overdueTasksRemindersTime: undefined | string | Date
	defaultProjectId: undefined | IProject['id']
	weekStart: 0 | 1 | 2 | 3 | 4 | 5 | 6
	timezone: string
	language: SupportedLocale | null
	frontendSettings: IFrontendSettings
	extraSettingsLinks: IExtraSettingsLinks
}
