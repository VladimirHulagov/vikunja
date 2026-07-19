import {beforeEach, describe, expect, it, vi} from 'vitest'
import {mount, flushPromises, type VueWrapper} from '@vue/test-utils'
import {createPinia, setActivePinia} from 'pinia'
import {defineComponent, h, reactive} from 'vue'
import {createI18n, type Composer} from 'vue-i18n'

// Mock vue-router: ProjectWrapper uses useRouter/useRoute, FilterPopup uses
// the project store, and our component uses useRouteQuery from @vueuse/router.
vi.mock('vue-router', () => ({
	useRouter: () => ({push: vi.fn(), currentRoute: {value: {fullPath: '/'}}}),
	useRoute: () => ({query: {}, name: 'project.project', params: {id: '1'}}),
}))

// Mock @vueuse/router so useRouteQuery returns stable refs.
vi.mock('@vueuse/router', () => ({
	useRouteQuery: () => ({value: undefined}),
}))

// Mock vue-i18n: many child components read translations during render.
const i18nStub = createI18n({
	legacy: false,
	locale: 'en',
	messages: {
		en: {
			project: {
				labeled: {
					noTags: 'project.labeled.noTags',
					emptyState: 'project.labeled.emptyState',
					addTaskPlaceholder: 'project.labeled.addTaskPlaceholder',
					loadMore: 'project.labeled.loadMore',
					title: 'project.labeled.title',
				},
			},
		},
	},
})
vi.mock('vue-i18n', async () => {
	const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
	return {
		...actual,
		useI18n: () => ({
			t: (key: string) => key,
			te: () => true,
		}),
	}
})

// Mock @kyvg/vue3-notification so success()/error() don't crash.
vi.mock('@kyvg/vue3-notification', () => ({
	notify: vi.fn(),
}))

// Stub the message helpers so we don't depend on i18n notification plumbing.
vi.mock('@/message', () => ({
	success: vi.fn(),
	error: vi.fn(),
}))

// Stub FilterPopup to avoid mounting its complex Filters child.
vi.mock('@/components/project/partials/FilterPopup.vue', () => ({
	default: defineComponent({
		name: 'FilterPopupStub',
		render: () => h('div', {class: 'filter-popup-stub'}),
	}),
}))

// Stub ProjectWrapper to render its default slot without the network
// loading, view switcher, etc.
vi.mock('@/components/project/ProjectWrapper.vue', () => ({
	default: defineComponent({
		name: 'ProjectWrapperStub',
		props: ['isLoadingProject', 'projectId', 'viewId'],
		render() {
			return h('div', {class: 'project-wrapper-stub'}, [
				h('div', {class: 'slot-default'}, this.$slots.default?.()),
			])
		},
	}),
}))

// Stub KanbanCard so we don't pull in the entire task-card dependency tree.
vi.mock('@/components/tasks/partials/KanbanCard.vue', () => ({
	default: defineComponent({
		name: 'KanbanCardStub',
		props: ['task', 'projectId'],
		render() {
			return h('div', {
				class: 'kanban-card-stub',
				'data-task-id': this.task.id,
			}, this.task.title)
		},
	}),
}))

// Stub XButton — the Bulma-flavoured button component pulls in BaseButton.
vi.mock('@/components/input/Button.vue', () => ({
	default: defineComponent({
		name: 'XButtonStub',
		props: ['disabled', 'variant', 'shadow'],
		emits: ['click'],
		render() {
			return h('button', {
				class: 'x-button-stub',
				disabled: this.disabled,
				onClick: () => this.$emit('click'),
			}, this.$slots.default?.())
		},
	}),
}))

// Shape of the stubbed labeled store. Pinia setup stores auto-unwrap refs
// accessed via the store proxy; we mimic that with getters backed by a
// reactive object so tests can mutate state directly.
interface LabeledStoreStub {
	groups: unknown[]
	untaggedGroup: unknown | null
	isLoading: boolean
	loadGroups: ReturnType<typeof vi.fn>
	loadMoreForGroup: ReturnType<typeof vi.fn>
	addTaskToGroup: ReturnType<typeof vi.fn>
	removeTaskFromGroup: ReturnType<typeof vi.fn>
	moveTaskBetweenGroups: ReturnType<typeof vi.fn>
}

const labeledState = reactive({
	groups: [] as ILabeledGroup[],
	untaggedGroup: null as ILabeledGroup | null,
	isLoading: false,
})
const labeledStoreMock: LabeledStoreStub = {
	loadGroups: vi.fn(),
	loadMoreForGroup: vi.fn(),
	addTaskToGroup: vi.fn(),
	removeTaskFromGroup: vi.fn(),
	moveTaskBetweenGroups: vi.fn(),
	get groups() {
		return labeledState.groups
	},
	set groups(value: unknown[]) {
		labeledState.groups = value as ILabeledGroup[]
	},
	get untaggedGroup() {
		return labeledState.untaggedGroup
	},
	set untaggedGroup(value: unknown) {
		labeledState.untaggedGroup = value as ILabeledGroup | null
	},
	get isLoading() {
		return labeledState.isLoading
	},
	set isLoading(value: boolean) {
		labeledState.isLoading = value
	},
}
vi.mock('@/stores/labeled', () => ({
	useLabeledStore: () => labeledStoreMock,
}))

interface ProjectLike {
	id?: number
	maxPermission?: number | null
}
const baseState = reactive<{currentProject: ProjectLike}>({
	currentProject: {maxPermission: 1},
})
const baseStoreMock = {
	get currentProject() {
		return baseState.currentProject
	},
	set currentProject(value: ProjectLike) {
		baseState.currentProject = value
	},
}
vi.mock('@/stores/base', () => ({
	useBaseStore: () => baseStoreMock,
}))

interface ProjectViewStub {
	id: number
	viewKind: string
}
interface ProjectRecord {
	id: number
	title: string
	views: ProjectViewStub[]
}
const projectsState = reactive<Record<number, ProjectRecord>>({
	1: {id: 1, title: 'Test Project', views: [{id: 1, viewKind: 'labeled'}]},
})
const projectStoreMock = {
	get projects() {
		return projectsState
	},
}
vi.mock('@/stores/projects', () => ({
	useProjectStore: () => projectStoreMock,
}))

const taskStoreMock = {
	createNewTask: vi.fn(),
}
vi.mock('@/stores/tasks', () => ({
	useTaskStore: () => taskStoreMock,
}))

interface LabelTaskServiceStub {
	create: ReturnType<typeof vi.fn>
	delete: ReturnType<typeof vi.fn>
}
const labelTaskServiceMock: LabelTaskServiceStub = {
	create: vi.fn(),
	delete: vi.fn(),
}
vi.mock('@/services/labelTask', () => ({
	default: class LabelTaskServiceMock {
		constructor() {
			return labelTaskServiceMock
		}
	},
}))

vi.mock('@/models/labelTask', () => ({
	default: function LabelTaskModelMock(this: unknown, data: unknown) {
		if (typeof data === 'object' && data !== null) {
			Object.assign(this as object, data)
		}
		return this
	},
}))

vi.mock('@/services/savedFilter', () => ({
	isSavedFilter: () => false,
}))

// A minimal subset of the HTMLElement interface that ProjectLabeled's
// onDragEnd handler actually reads.
interface DragDomEl {
	dataset: {
		groupKey?: string
		taskId?: string
	}
}

interface DragEventPayload {
	item: DragDomEl
	from: DragDomEl
	to: DragDomEl
	oldIndex: number
	newIndex: number
}

interface DraggableComponentProps {
	onEnd?: (evt: DragEventPayload) => void | Promise<void>
}

// Replace zhyswan-vuedraggable with a thin stub that renders its #item slot
// for every entry and exposes its bound props (group, disabled, etc.) and
// the onEnd callback via the test handle below.
vi.mock('zhyswan-vuedraggable', () => ({
	default: defineComponent({
		name: 'DraggableStub',
		props: [
			'modelValue',
			'group',
			'sort',
			'animation',
			'ghostClass',
			'disabled',
			'dataGroupKey',
			'tag',
			'itemKey',
			// Declare `onEnd` as a real prop so tests can read it back from
			// $props after the parent binds `@end`. We intentionally do NOT
			// declare it via `emits`, otherwise Vue would route it to $attrs.
			'onEnd',
		],
		render() {
			const items = (this.modelValue ?? []) as Array<{id: number}>
			const itemSlot = this.$slots.item
			const children = items.map((element, index) =>
				itemSlot
					? itemSlot({element, index})
					: h('li', {key: index}, JSON.stringify(element)),
			)
			return h('ul', {
				class: 'draggable-stub',
				'data-group-key': this.dataGroupKey,
				'data-disabled': this.disabled ? 'true' : 'false',
			}, children)
		},
	}),
}))

function resetMockStores() {
	labeledState.groups = []
	labeledState.untaggedGroup = null
	labeledState.isLoading = false
	labeledStoreMock.loadGroups.mockClear()
	labeledStoreMock.loadMoreForGroup.mockClear()
	labeledStoreMock.addTaskToGroup.mockClear()
	labeledStoreMock.removeTaskFromGroup.mockClear()
	labeledStoreMock.moveTaskBetweenGroups.mockClear()

	baseState.currentProject = {maxPermission: 1}

	projectsState[1] = {
		id: 1,
		title: 'Test Project',
		views: [{id: 1, viewKind: 'labeled'}],
	}

	taskStoreMock.createNewTask.mockReset()
	labelTaskServiceMock.create.mockReset()
	labelTaskServiceMock.delete.mockReset()
}

import ProjectLabeled from '@/components/project/views/ProjectLabeled.vue'

import type {ILabeledGroup} from '@/modelTypes/ILabeledView'
import type {ITask} from '@/modelTypes/ITask'
import type {ILabel} from '@/modelTypes/ILabel'

function makeLabel(id: number, title = `Label ${id}`, hexColor = '#ff0000'): ILabel {
	return {id, title, hexColor} as ILabel
}

function makeTask(id: number, title = `Task ${id}`, labels: ILabel[] = []): ITask {
	return {
		id,
		title,
		labels,
		projectId: 1,
	} as ITask
}

function makeGroup(
	label: ILabel | null,
	tasks: ITask[],
	taskCount?: number,
): ILabeledGroup {
	return {
		label,
		tasks,
		taskCount: typeof taskCount === 'undefined' ? tasks.length : taskCount,
	} as ILabeledGroup
}

interface ProjectLabeledProps {
	isLoadingProject: boolean
	projectId: number
	viewId: number
}

function mountComponent(overrides: Partial<ProjectLabeledProps> = {}) {
	return mount(ProjectLabeled, {
		props: {
			isLoadingProject: false,
			projectId: 1,
			viewId: 1,
			...overrides,
		},
		global: {
			plugins: [createPinia(), i18nStub],
			stubs: {
				transition: false,
				transitionGroup: false,
			},
		},
	})
}

function getDragEnd(wrapper: VueWrapper, index: number) {
	const draggables = wrapper.findAllComponents({name: 'DraggableStub'})
	const props = draggables[index].vm.$props as DraggableComponentProps
	return {
		onEnd: props.onEnd as (evt: DragEventPayload) => Promise<void>,
		from: draggables[index].element as unknown as DragDomEl,
	}
}

function buildItem(taskId: number): DragDomEl {
	return {dataset: {taskId: String(taskId)}}
}

// Use the composer so TS doesn't elide the import as unused.
void (null as unknown as Composer)

describe('ProjectLabeled.vue', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		resetMockStores()
	})

	it('renders groups with label titles, color dots, and task counts', async () => {
		const label = makeLabel(7, 'Urgent', '#00aa00')
		labeledStoreMock.groups = [
			makeGroup(label, [makeTask(1, 'First', [label]), makeTask(2, 'Second', [label])], 2),
		]
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		const headers = wrapper.findAll('.labeled-column-header')
		expect(headers).toHaveLength(1)
		expect(headers[0].text()).toContain('Urgent')
		expect(headers[0].text()).toContain('2')

		const dot = wrapper.find('.color-dot')
		expect(dot.exists()).toBe(true)
		expect(dot.attributes('style')).toContain('#00aa00')

		expect(wrapper.findAll('.kanban-card-stub')).toHaveLength(2)
	})

	it('shows the empty-state message when there are no groups or tasks', async () => {
		labeledStoreMock.groups = []
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		expect(wrapper.find('.empty-state').exists()).toBe(true)
		expect(wrapper.find('.empty-state').text()).toBe('project.labeled.emptyState')
		expect(wrapper.find('.labeled-columns').exists()).toBe(false)
	})

	it('renders the "No tags" column when untaggedGroup is non-null', async () => {
		labeledStoreMock.groups = []
		labeledStoreMock.untaggedGroup = makeGroup(null, [makeTask(10)], 1)

		const wrapper = mountComponent()
		await flushPromises()

		const headers = wrapper.findAll('.labeled-column-header')
		expect(headers).toHaveLength(1)
		expect(headers[0].text()).toContain('project.labeled.noTags')
	})

	it('does not render the "No tags" column when untaggedGroup is null', async () => {
		labeledStoreMock.groups = [makeGroup(makeLabel(1), [makeTask(1)], 1)]
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		const titles = wrapper.findAll('.label-title').map(el => el.text())
		expect(titles).not.toContain('project.labeled.noTags')
	})

	it('fires DELETE and POST when dragging between two label columns', async () => {
		const sourceLabel = makeLabel(11, 'Source')
		const targetLabel = makeLabel(22, 'Target')
		const movedTask = makeTask(99, 'Moved', [sourceLabel])
		labeledStoreMock.groups = [
			makeGroup(sourceLabel, [movedTask], 1),
			makeGroup(targetLabel, [], 0),
		]
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		const columns = wrapper.findAll('.labeled-column')
		expect(columns).toHaveLength(2)

		const draggables = wrapper.findAllComponents({name: 'DraggableStub'})
		expect(draggables).toHaveLength(2)

		const source = getDragEnd(wrapper, 0)
		const target = getDragEnd(wrapper, 1)
		expect(typeof source.onEnd).toBe('function')

		labelTaskServiceMock.delete.mockResolvedValue({})
		labelTaskServiceMock.create.mockResolvedValue({})

		await source.onEnd({
			item: buildItem(99),
			from: source.from,
			to: target.from,
			oldIndex: 0,
			newIndex: 0,
		})
		await flushPromises()

		expect(labelTaskServiceMock.delete).toHaveBeenCalledWith(
			expect.objectContaining({taskId: 99, labelId: 11}),
		)
		expect(labelTaskServiceMock.create).toHaveBeenCalledWith(
			expect.objectContaining({taskId: 99, labelId: 22}),
		)
		expect(labeledStoreMock.moveTaskBetweenGroups).toHaveBeenCalledWith(11, 22, 99)
	})

	it('only POSTs (no DELETE) when dragging from "No tags" into a label column', async () => {
		const targetLabel = makeLabel(33, 'Target')
		const movedTask = makeTask(77, 'Untagged')
		labeledStoreMock.groups = [makeGroup(targetLabel, [], 0)]
		labeledStoreMock.untaggedGroup = makeGroup(null, [movedTask], 1)

		const wrapper = mountComponent()
		await flushPromises()

		// Index 0 is the label column, index 1 is the untagged column
		// (untagged is always appended last).
		const untagged = getDragEnd(wrapper, 1)
		const target = getDragEnd(wrapper, 0)

		labelTaskServiceMock.create.mockResolvedValue({})
		labelTaskServiceMock.delete.mockResolvedValue({})

		await untagged.onEnd({
			item: buildItem(77),
			from: untagged.from,
			to: target.from,
			oldIndex: 0,
			newIndex: 0,
		})
		await flushPromises()

		expect(labelTaskServiceMock.delete).not.toHaveBeenCalled()
		expect(labelTaskServiceMock.create).toHaveBeenCalledWith(
			expect.objectContaining({taskId: 77, labelId: 33}),
		)
	})

	it('only DELETEs (no POST) when dragging from a label column to "No tags"', async () => {
		const sourceLabel = makeLabel(44, 'Source')
		const movedTask = makeTask(55, 'Tagged', [sourceLabel])
		labeledStoreMock.groups = [makeGroup(sourceLabel, [movedTask], 1)]
		labeledStoreMock.untaggedGroup = makeGroup(null, [], 0)

		const wrapper = mountComponent()
		await flushPromises()

		const source = getDragEnd(wrapper, 0)
		const untagged = getDragEnd(wrapper, 1)

		labelTaskServiceMock.create.mockResolvedValue({})
		labelTaskServiceMock.delete.mockResolvedValue({})

		await source.onEnd({
			item: buildItem(55),
			from: source.from,
			to: untagged.from,
			oldIndex: 0,
			newIndex: 0,
		})
		await flushPromises()

		expect(labelTaskServiceMock.delete).toHaveBeenCalledWith(
			expect.objectContaining({taskId: 55, labelId: 44}),
		)
		expect(labelTaskServiceMock.create).not.toHaveBeenCalled()
	})

	it('creates a task with the column label pre-assigned on add-task submit', async () => {
		const label = makeLabel(8, 'Work')
		// Seed an existing task so hasAnyTasks is true and the columns render.
		labeledStoreMock.groups = [makeGroup(label, [makeTask(1)], 1)]
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		const input = wrapper.find('.add-task input')
		expect(input.exists()).toBe(true)
		await input.setValue('New bug report')

		const created = makeTask(123, 'New bug report')
		taskStoreMock.createNewTask.mockResolvedValue(created)
		labelTaskServiceMock.create.mockResolvedValue({})

		await input.trigger('keyup.enter')
		await flushPromises()

		expect(taskStoreMock.createNewTask).toHaveBeenCalledWith(expect.objectContaining({
			title: 'New bug report',
			projectId: 1,
		}))
		expect(labelTaskServiceMock.create).toHaveBeenCalledWith(
			expect.objectContaining({taskId: 123, labelId: 8}),
		)
		expect(labeledStoreMock.addTaskToGroup).toHaveBeenCalledWith(
			8,
			expect.objectContaining({id: 123}),
		)
		expect((input.element as HTMLInputElement).value).toBe('')
	})

	it('disables draggable when canWrite is false (read-only permission)', async () => {
		baseState.currentProject = {maxPermission: 0}
		const label = makeLabel(1, 'ReadOnly')
		labeledStoreMock.groups = [makeGroup(label, [makeTask(1)], 1)]

		const wrapper = mountComponent()
		await flushPromises()

		const draggables = wrapper.findAll('.draggable-stub')
		expect(draggables).toHaveLength(1)
		expect(draggables[0].attributes('data-disabled')).toBe('true')
	})

	it('shows the "Load more" button only when tasks.length < taskCount', async () => {
		const label = makeLabel(2, 'Big')
		labeledStoreMock.groups = [
			makeGroup(label, [makeTask(1)], 5),
		]
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		expect(wrapper.find('.column-footer').exists()).toBe(true)
		expect(wrapper.find('.x-button-stub').exists()).toBe(true)

		// Reload with all tasks loaded — button should disappear.
		resetMockStores()
		labeledStoreMock.groups = [
			makeGroup(label, [makeTask(1), makeTask(2)], 2),
		]
		const wrapper2 = mountComponent()
		await flushPromises()
		expect(wrapper2.find('.column-footer').exists()).toBe(false)
	})

	it('triggers the store loadMoreForGroup action on "Load more" click', async () => {
		const label = makeLabel(3, 'More')
		labeledStoreMock.groups = [
			makeGroup(label, [makeTask(1)], 30),
		]
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		await wrapper.find('.x-button-stub').trigger('click')
		await flushPromises()

		expect(labeledStoreMock.loadMoreForGroup).toHaveBeenCalledTimes(1)
		const [projectIdArg, viewIdArg, groupKeyArg] =
			labeledStoreMock.loadMoreForGroup.mock.calls[0]
		expect(projectIdArg).toBe(1)
		expect(viewIdArg).toBe(1)
		expect(groupKeyArg).toBe(3)
	})

	it('rolls back the optimistic move when an API call fails', async () => {
		const sourceLabel = makeLabel(50, 'A')
		const targetLabel = makeLabel(60, 'B')
		const movedTask = makeTask(70, 'WillFail', [sourceLabel])
		labeledStoreMock.groups = [
			makeGroup(sourceLabel, [movedTask], 1),
			makeGroup(targetLabel, [], 0),
		]
		labeledStoreMock.untaggedGroup = null

		const wrapper = mountComponent()
		await flushPromises()

		const source = getDragEnd(wrapper, 0)
		const target = getDragEnd(wrapper, 1)

		labelTaskServiceMock.delete.mockResolvedValue({})
		labelTaskServiceMock.create.mockRejectedValue(new Error('boom'))

		await source.onEnd({
			item: buildItem(70),
			from: source.from,
			to: target.from,
			oldIndex: 0,
			newIndex: 0,
		})
		await flushPromises()

		expect(labeledStoreMock.moveTaskBetweenGroups).toHaveBeenCalledTimes(2)
		expect(labeledStoreMock.moveTaskBetweenGroups).toHaveBeenNthCalledWith(1, 50, 60, 70)
		expect(labeledStoreMock.moveTaskBetweenGroups).toHaveBeenNthCalledWith(2, 60, 50, 70)
	})
})
