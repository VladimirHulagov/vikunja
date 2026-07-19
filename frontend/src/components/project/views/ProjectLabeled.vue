<template>
	<ProjectWrapper
		class="project-labeled"
		:is-loading-project="isLoadingProject"
		:project-id="projectId"
		:view-id
	>
		<template #header>
			<div class="filter-container">
				<FancyCheckbox
					v-model="hideDone"
					class="hide-done-toggle"
				>
					{{ $t('project.labeled.hideDone') }}
				</FancyCheckbox>
				<LabelCategoryModal
					v-if="!projectIsSavedFilter"
					:project-id="projectId"
					@changed="onCategoriesChanged"
				/>
				<FilterPopup
					v-if="!projectIsSavedFilter"
					v-model="params"
					:view-id="viewId"
					:project-id="projectId"
					@update:modelValue="updateFilters"
				/>
			</div>
		</template>

		<template #default>
			<div class="labeled-view">
				<LabelCategoryCloud
					:project-id="projectId"
					:active-category-id="activeCategoryId"
					@select="onSelectCategory"
				/>
				<div
					:class="{ 'is-loading': isLoading }"
					class="labeled labeled-columns-container loader-container"
				>
					<div
						v-if="hasAnyTasks"
						class="labeled-columns"
					>
						<div
							v-for="group in allGroups"
							:key="groupKeyOf(group)"
							class="labeled-column"
							:data-group-key="groupKeyOf(group)"
						>
							<header class="labeled-column-header">
								<span
									class="color-dot"
									:style="{background: group.label?.hexColor || '#999'}"
								/>
								<h2 class="label-title">
									{{ group.label?.title || $t('project.labeled.noTags') }}
								</h2>
								<span class="task-count">{{ group.taskCount }}</span>
							</header>

							<draggable
								v-model="localTasksByGroup[groupKeyOf(group)]"
								:group="{name: 'tasks', pull: true, put: true}"
								:sort="false"
								:animation="150"
								ghost-class="ghost"
								:disabled="!canWrite"
								:data-group-key="groupKeyOf(group)"
								tag="ul"
								:item-key="(task: ITask) => `group${groupKeyOf(group)}-task${task.id}`"
								@end="onDragEnd"
							>
								<template #item="{element: task}">
									<div
										class="task-item"
										:data-task-id="task.id"
									>
										<KanbanCard
											class="kanban-card"
											:task="task"
											:project-id="projectId"
										/>
									</div>
								</template>
							</draggable>

							<div
								v-if="hasMoreTasks(group)"
								class="column-footer"
							>
								<XButton
									:shadow="false"
									class="is-fullwidth has-text-centered"
									variant="secondary"
									:disabled="isLoadingMore"
									@click="loadMore(group)"
								>
									{{ $t('project.labeled.loadMore') }}
								</XButton>
							</div>

							<div
								v-if="canWrite && group.label"
								class="add-task"
							>
								<div
									class="control"
									:class="{'is-loading': creatingForGroup === groupKeyOf(group)}"
								>
									<input
										v-model="newTaskTitles[groupKeyOf(group)]"
										class="input"
										:placeholder="$t('project.labeled.addTaskPlaceholder')"
										type="text"
										@keyup.enter="addTask(group)"
									>
								</div>
							</div>
						</div>
					</div>

					<div
						v-else-if="!isLoading"
						class="empty-state"
					>
						{{ $t('project.labeled.emptyState') }}
					</div>
				</div>
			</div>
		</template>
	</ProjectWrapper>
</template>

<script setup lang="ts">
import {computed, ref, toRef, watch} from 'vue'
import {useRouteQuery} from '@vueuse/router'
import {klona} from 'klona/lite'
import draggable from 'zhyswan-vuedraggable'

import {PERMISSIONS as Permissions} from '@/constants/permissions'
import LabelTaskService from '@/services/labelTask'
import LabelTaskModel from '@/models/labelTask'
import {isSavedFilter} from '@/services/savedFilter'

import type {ITask} from '@/modelTypes/ITask'
import type {IProjectView} from '@/modelTypes/IProjectView'
import type {ILabeledGroup} from '@/modelTypes/ILabeledView'
import type {TaskFilterParams} from '@/services/taskCollection'

import {useBaseStore} from '@/stores/base'
import {useLabeledStore} from '@/stores/labeled'
import {useLabelCategoriesStore} from '@/stores/labelCategories'
import {useTaskStore} from '@/stores/tasks'
import {useProjectStore} from '@/stores/projects'

import ProjectWrapper from '@/components/project/ProjectWrapper.vue'
import FilterPopup from '@/components/project/partials/FilterPopup.vue'
import LabelCategoryCloud from '@/components/project/labelCategories/LabelCategoryCloud.vue'
import LabelCategoryModal from '@/components/project/labelCategories/LabelCategoryModal.vue'
import KanbanCard from '@/components/tasks/partials/KanbanCard.vue'
import XButton from '@/components/input/Button.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'

import {error as showError} from '@/message'

const props = defineProps<{
	isLoadingProject: boolean,
	projectId: number,
	viewId: IProjectView['id'],
}>()

const projectId = toRef(props, 'projectId')

const baseStore = useBaseStore()
const labeledStore = useLabeledStore()
const categoriesStore = useLabelCategoriesStore()
const taskStore = useTaskStore()
const projectStore = useProjectStore()

const labelTaskService = new LabelTaskService()

// 0 is reserved for the untagged pseudo-group.
const UNTAGGED_GROUP_KEY = 0

const project = computed(() => projectId.value ? projectStore.projects[projectId.value] : null)
const projectIsSavedFilter = computed(() => {
	const p = project.value
	if (!p) return false
	return isSavedFilter({...p, id: p.id} as Parameters<typeof isSavedFilter>[0])
})
const canWrite = computed(() => {
	const perm = baseStore.currentProject?.maxPermission
	return typeof perm === 'number' && perm > Permissions.READ
})

const groups = computed(() => labeledStore.groups)
const untaggedGroup = computed(() => labeledStore.untaggedGroup)
const isLoading = computed(() => labeledStore.isLoading)
const isLoadingMore = ref(false)
const creatingForGroup = ref<number | null>(null)

const allGroups = computed<ILabeledGroup[]>(() => {
	const list: ILabeledGroup[] = [...groups.value]
	if (untaggedGroup.value !== null) {
		list.push(untaggedGroup.value)
	}
	return list
})

const hasAnyTasks = computed(() => allGroups.value.length > 0)

// Local mutable mirrors of each group's task list. zhyswan-vuedraggable
// requires a writable v-model; the store's arrays are not safe to mutate
// directly because drag-end may need to roll back.
const localTasksByGroup = ref<Record<number, ITask[]>>({})

function syncLocalTasks() {
	const next: Record<number, ITask[]> = {}
	for (const group of allGroups.value) {
		next[groupKeyOf(group)] = group.tasks.slice()
	}
	localTasksByGroup.value = next
}

watch(allGroups, syncLocalTasks, {immediate: true, deep: true})

const newTaskTitles = ref<Record<number, string>>({})

function groupKeyOf(group: ILabeledGroup): number {
	return group.label?.id ?? UNTAGGED_GROUP_KEY
}

function hasMoreTasks(group: ILabeledGroup): boolean {
	return group.tasks.length < group.taskCount
}

// URL-synchronized filter parameters — same pattern as ProjectKanban.
const filter = useRouteQuery('filter')
const s = useRouteQuery('s')
const categoryQuery = useRouteQuery('category')

function parseCategory(value: string | string[] | undefined | null): number {
	if (value === undefined || value === null) return 0
	const raw = Array.isArray(value) ? value[0] : value
	if (raw === undefined || raw === null || raw === '') return 0
	const parsed = Number.parseInt(String(raw), 10)
	if (Number.isNaN(parsed)) return 0
	return parsed
}

const activeCategoryId = ref<number>(parseCategory(categoryQuery.value))

watch(categoryQuery, (value) => {
	activeCategoryId.value = parseCategory(value)
})

function onSelectCategory(id: number) {
	activeCategoryId.value = id
	// `useRouteQuery` treats `undefined` as "remove the param"; we keep 0
	// out of the URL to keep shares clean.
	categoryQuery.value = id === 0 ? undefined : String(id)
}

function onCategoriesChanged() {
	// Reload both the category list (so the cloud above the columns refreshes)
	// and the task groups (so the columns reflect any category membership
	// changes). The modal updates the store itself, but we re-load explicitly
	// to be safe against any reactive edge cases.
	if (projectId.value) {
		categoriesStore.load(projectId.value)
		labeledStore.loadGroups(projectId.value, props.viewId, {
			...params.value,
			category: activeCategoryId.value,
		})
	}
}

const params = ref<TaskFilterParams>({
	sort_by: [],
	order_by: [],
	filter: '',
	filter_include_nulls: false,
	s: '',
})

watch([filter, s], ([filterValue, sValue]) => {
	params.value.filter = Array.isArray(filterValue) ? filterValue[0] ?? '' : filterValue ?? ''
	params.value.s = Array.isArray(sValue) ? sValue[0] ?? '' : sValue ?? ''
}, {immediate: true})

function updateFilters(newParams: TaskFilterParams) {
	params.value = {...newParams}
	filter.value = newParams.filter || undefined
	s.value = newParams.s || undefined
}

// Hide-done toggle. The view's default filter is `done = false` (set by
// migration); we expose it as a single checkbox next to the filter popup so
// users don't have to hand-edit the filter string.
const DONE_FALSE_RE = /(^|\s)&&\s*done\s*=\s*false(?=\s|$)|(^|\s)done\s*=\s*false(\s*&&)?/i

const hideDone = computed<boolean>({
	get() {
		const f = (params.value.filter ?? '').trim()
		if (f === '') return false
		// Match `done = false` standalone or as part of an && chain.
		return /\bdone\s*=\s*false\b/i.test(f)
	},
	set(v: boolean) {
		const f = (params.value.filter ?? '').trim()
		let next: string
		if (v) {
			next = f === '' ? 'done = false' : `(${f}) && done = false`
		} else {
			// Remove `done = false` (and the wrapping parens we may have added).
			next = f.replace(DONE_FALSE_RE, '$1').replace(/\(\s*\)&&/i, '').replace(/\(\s*&&/i, '(').trim()
			next = next.replace(/^\(\s*(.*?)\s*\)$/, '$1').trim()
		}
		const newParams = {...params.value, filter: next}
		updateFilters(newParams)
	},
})

watch(
	() => ({
		params: params.value,
		projectId: projectId.value,
		viewId: props.viewId,
		categoryId: activeCategoryId.value,
	}),
	({params, projectId, viewId, categoryId}) => {
		if (projectId === undefined || Number(projectId) === 0) {
			return
		}
		labeledStore.loadGroups(projectId, viewId, {
			...params,
			category: categoryId,
		})
	},
	{immediate: true, deep: true},
)

interface DragEventLike {
	item: HTMLElement,
	from: HTMLElement,
	to: HTMLElement,
	oldIndex: number,
	newIndex: number,
}

function readGroupKey(el: HTMLElement | null): number | null {
	if (el === null) {
		return null
	}
	const raw = el.dataset.groupKey
	if (typeof raw === 'undefined') {
		return null
	}
	const parsed = parseInt(raw, 10)
	return Number.isNaN(parsed) ? null : parsed
}

async function onDragEnd(evt: DragEventLike) {
	const fromKey = readGroupKey(evt.from)
	const toKey = readGroupKey(evt.to)
	const taskId = parseInt(evt.item.dataset.taskId ?? '', 10)

	if (fromKey === null || toKey === null || Number.isNaN(taskId)) {
		return
	}
	if (fromKey === toKey) {
		// Internal reordering is disabled (`:sort="false"`); reset local mirror.
		syncLocalTasks()
		return
	}

	const sourceLabelId = fromKey === UNTAGGED_GROUP_KEY ? null : fromKey
	const targetLabelId = toKey === UNTAGGED_GROUP_KEY ? null : toKey

	// Optimistic UI update in the store.
	labeledStore.moveTaskBetweenGroups(fromKey, toKey, taskId)

	const operations: Promise<unknown>[] = []
	if (sourceLabelId !== null) {
		operations.push(
			labelTaskService.delete(new LabelTaskModel({
				taskId,
				labelId: sourceLabelId,
			})),
		)
	}
	if (targetLabelId !== null) {
		operations.push(
			labelTaskService.create(new LabelTaskModel({
				taskId,
				labelId: targetLabelId,
			})),
		)
	}

	try {
		await Promise.all(operations)
	} catch (e) {
		// Roll back the optimistic move and surface the error.
		labeledStore.moveTaskBetweenGroups(toKey, fromKey, taskId)
		showError(e)
	}
}

async function addTask(group: ILabeledGroup) {
	if (group.label === null) {
		// The "No tags" column doesn't pre-assign a label.
		return
	}
	const label = group.label
	const key = groupKeyOf(group)
	const title = (newTaskTitles.value[key] ?? '').trim()
	if (title === '') {
		return
	}

	creatingForGroup.value = key
	try {
		const created = await taskStore.createNewTask({
			title,
			projectId: projectId.value,
		})

		// Attach the column's label so the task shows up here on reload.
		try {
			await labelTaskService.create(new LabelTaskModel({
				taskId: created.id,
				labelId: label.id,
			}))
			created.labels = [...(created.labels ?? []), klona(label)]
		} catch (e) {
			showError(e)
		}

		labeledStore.addTaskToGroup(key, created)
		newTaskTitles.value[key] = ''
	} catch (e) {
		showError(e)
	} finally {
		creatingForGroup.value = null
	}
}

async function loadMore(group: ILabeledGroup) {
	const key = groupKeyOf(group)
	if (isLoadingMore.value) {
		return
	}
	isLoadingMore.value = true
	try {
		await labeledStore.loadMoreForGroup(projectId.value, props.viewId, key, {
			...params.value,
			category: activeCategoryId.value,
		})
	} finally {
		isLoadingMore.value = false
	}
}
</script>

<style lang="scss" scoped>
.labeled-view {
	.labeled-columns-container {
		block-size: 100%;
	}
}
</style>

<style lang="scss">
$column-width: 300px;
$column-header-height: 60px;
$column-right-margin: 1rem;
$crazy-height-calculation: '100vh - 4.5rem - 1.5rem - 1rem - 1.5rem - 11px';
$filter-container-height: '1rem - #{$switch-view-height}';

.filter-container {
	display: flex;
	align-items: center;
	gap: 1rem;
	flex-wrap: wrap;
}

.hide-done-toggle {
	margin: 0;
}

.labeled {
	overflow-x: auto;
	overflow-y: hidden;
	block-size: calc(#{$crazy-height-calculation});
	margin: 0 -1.5rem;
	padding: 0 1.5rem;

	@media screen and (max-width: $tablet) {
		block-size: calc(#{$crazy-height-calculation} - #{$filter-container-height} + 9px);
		scroll-snap-type: x mandatory;
		margin: 0 -0.5rem;
	}

	.labeled-columns {
		display: flex;
	}

	.ghost {
		position: relative;

		* {
			opacity: 0;
		}

		&::after {
			content: '';
			position: absolute;
			display: block;
			inset-block-start: 0.25rem;
			inset-inline-end: 0.5rem;
			inset-block-end: 0.25rem;
			inset-inline-start: 0.5rem;
			border: 3px dashed var(--grey-300);
			border-radius: $radius;
		}
	}

	.labeled-column {
		border-radius: $radius;
		background-color: var(--grey-100);
		margin: 0 $column-right-margin 0 0;
		max-block-size: calc(100% - 1rem);
		min-block-size: 20px;
		inline-size: $column-width;
		display: flex;
		flex-direction: column;
		overflow: hidden;

		@media screen and (max-width: $tablet) {
			scroll-snap-align: center;
		}

		ul {
			overflow-y: auto;
			flex: 1 1 auto;
			block-size: 100%;
		}

		.task-item {
			padding: .25rem .5rem;

			&:first-of-type {
				padding-block-start: .5rem;
			}

			&:last-of-type {
				padding-block-end: .5rem;
			}
		}
	}

	.labeled-column-header {
		background-color: var(--grey-100);
		display: flex;
		align-items: center;
		gap: .5rem;
		padding: .5rem .75rem;
		block-size: $column-header-height;

		.color-dot {
			inline-size: 12px;
			block-size: 12px;
			border-radius: 50%;
			flex: 0 0 auto;
			display: inline-block;
		}

		.label-title {
			font-size: 1rem;
			margin: 0;
			font-weight: 600 !important;
			flex: 1 1 auto;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		.task-count {
			padding: 0 .5rem;
			font-weight: bold;
			color: var(--grey-500);
		}
	}

	.column-footer {
		padding: .5rem;
	}

	.add-task {
		padding: .5rem;
		border-end-start-radius: $radius;
		border-end-end-radius: $radius;

		.control.is-loading::after {
			inset-block-start: 30%;
			inset-inline-end: 50%;
			transform: translate(-50%, 0);
		}
	}

	.empty-state {
		text-align: center;
		padding: 2rem;
		color: var(--grey-500);
	}
}
</style>
