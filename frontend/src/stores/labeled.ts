import {readonly, ref} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'

import {findIndexById} from '@/helpers/utils'

import LabeledViewService from '@/services/labeledView'
import type {TaskFilterParams} from '@/services/taskCollection'

import {setModuleLoading} from '@/stores/helper'

import type {ITask} from '@/modelTypes/ITask'
import type {IProject} from '@/modelTypes/IProject'
import type {IProjectView} from '@/modelTypes/IProjectView'
import type {ILabeledGroup, ILabeledViewResponse} from '@/modelTypes/ILabeledView'

const TASKS_PER_GROUP = 25

// groupKey 0 is reserved for the untagged pseudo-group. Real label columns
// use the label id as their key.
const UNTAGGED_GROUP_KEY = 0

/**
 * Holds the currently active labeled view. Mirrors the shape of the kanban
 * store, but with groups keyed by label instead of buckets keyed by id.
 */
export const useLabeledStore = defineStore('labeled', () => {
	const groups = ref<ILabeledGroup[]>([])
	const untaggedGroup = ref<ILabeledGroup | null>(null)
	const loadedPagesPerGroup = ref<Record<number, number>>({})
	const allTasksLoadedPerGroup = ref<Record<number, boolean>>({})
	const isLoading = ref(false)

	function setIsLoading(newIsLoading: boolean) {
		isLoading.value = newIsLoading
	}

	function groupKeyOf(group: ILabeledGroup): number {
		return group?.label?.id ?? UNTAGGED_GROUP_KEY
	}

	function findGroupIndex(groupKey: number): number {
		return groups.value.findIndex(g => groupKeyOf(g) === groupKey)
	}

	function setGroups(newGroups: ILabeledGroup[], newUntagged: ILabeledGroup | null) {
		groups.value = newGroups
		untaggedGroup.value = newUntagged
		loadedPagesPerGroup.value = {}
		allTasksLoadedPerGroup.value = {}
		newGroups.forEach(g => {
			const key = groupKeyOf(g)
			loadedPagesPerGroup.value[key] = 1
			// Mark "all loaded" when the first page already contains every
			// task (count <= TASKS_PER_GROUP).
			allTasksLoadedPerGroup.value[key] = g.tasks.length >= g.taskCount
		})
		if (newUntagged !== null) {
			loadedPagesPerGroup.value[UNTAGGED_GROUP_KEY] = 1
			allTasksLoadedPerGroup.value[UNTAGGED_GROUP_KEY] =
				newUntagged.tasks.length >= newUntagged.taskCount
		}
	}

	function ensureUntaggedGroup(): ILabeledGroup {
		if (untaggedGroup.value === null) {
			untaggedGroup.value = {
				label: null,
				taskCount: 0,
				tasks: [],
				maxPermission: null,
			}
		}
		return untaggedGroup.value
	}

	function appendTasksToGroup(groupKey: number, tasks: ITask[]) {
		if (groupKey === UNTAGGED_GROUP_KEY) {
			if (untaggedGroup.value === null) {
				return
			}
			untaggedGroup.value.tasks.push(...tasks)
			return
		}
		const idx = findGroupIndex(groupKey)
		if (idx === -1) {
			return
		}
		groups.value[idx].tasks.push(...tasks)
	}

	function addTaskToGroup(groupKey: number, task: ITask) {
		if (groupKey === UNTAGGED_GROUP_KEY) {
			const group = ensureUntaggedGroup()
			group.taskCount += 1
			group.tasks.unshift(task)
			return
		}
		const idx = findGroupIndex(groupKey)
		if (idx === -1) {
			return
		}
		const group = groups.value[idx]
		group.taskCount += 1
		group.tasks.unshift(task)
	}

	function removeTaskFromGroup(groupKey: number, taskId: ITask['id']) {
		if (groupKey === UNTAGGED_GROUP_KEY) {
			if (untaggedGroup.value === null) {
				return
			}
			const taskIndex = findIndexById(untaggedGroup.value.tasks, taskId)
			if (taskIndex !== -1) {
				untaggedGroup.value.tasks.splice(taskIndex, 1)
				untaggedGroup.value.taskCount = Math.max(0, untaggedGroup.value.taskCount - 1)
			}
			return
		}
		const idx = findGroupIndex(groupKey)
		if (idx === -1) {
			return
		}
		const group = groups.value[idx]
		const taskIndex = findIndexById(group.tasks, taskId)
		if (taskIndex !== -1) {
			group.tasks.splice(taskIndex, 1)
			group.taskCount = Math.max(0, group.taskCount - 1)
		}
	}

	function moveTaskBetweenGroups(fromKey: number, toKey: number, taskId: ITask['id']) {
		// Find the task object in the source group before removing it,
		// so we can hand the same instance to the target group.
		let task: ITask | undefined
		if (fromKey === UNTAGGED_GROUP_KEY) {
			task = untaggedGroup.value?.tasks.find(t => t.id === taskId)
		} else {
			const idx = findGroupIndex(fromKey)
			if (idx !== -1) {
				task = groups.value[idx].tasks.find(t => t.id === taskId)
			}
		}
		if (typeof task === 'undefined') {
			return
		}
		removeTaskFromGroup(fromKey, taskId)
		addTaskToGroup(toKey, task)
	}

	async function loadGroups(
		projectId: IProject['id'],
		viewId: IProjectView['id'],
		params: Partial<TaskFilterParams> = {},
	) {
		const cancel = setModuleLoading(setIsLoading)

		// Reset state so we don't show stale groups from a previous view.
		setGroups([], null)

		const service = new LabeledViewService()
		try {
			const response = await service.get(
				// The service only uses projectId/viewId for URL templating;
				// the cast mirrors the loose pattern used by TaskCollectionService
				// consumers (kanban store).
				{projectId, viewId} as unknown as ILabeledViewResponse,
				{
					...params,
					per_page: TASKS_PER_GROUP,
				},
			)
			setGroups(response.groups, response.untaggedGroup)
			return response
		} finally {
			cancel()
		}
	}

	async function loadMoreForGroup(
		projectId: IProject['id'],
		viewId: IProjectView['id'],
		groupKey: number,
		params: Partial<TaskFilterParams> = {},
	) {
		// The backend paginates all groups in lockstep via the `page` param
		// (each call returns the next page of every group). We re-request
		// the next page and only merge the requested group's tasks. This is
		// slightly wasteful for projects with many labels, but it matches
		// the backend's single-page-for-all model and lets us avoid having
		// to express "labels is empty" in the filter syntax.
		if (allTasksLoadedPerGroup.value[groupKey]) {
			return
		}

		const page = (loadedPagesPerGroup.value[groupKey] ?? 1) + 1
		const cancel = setModuleLoading(setIsLoading)

		const service = new LabeledViewService()
		try {
			const response = await service.get(
				{projectId, viewId} as unknown as ILabeledViewResponse,
				{
					...params,
					per_page: TASKS_PER_GROUP,
					page,
				},
			)

			let newTasks: ITask[] = []
			let totalForGroup = 0
			if (groupKey === UNTAGGED_GROUP_KEY) {
				newTasks = response.untaggedGroup?.tasks ?? []
				totalForGroup = response.untaggedGroup?.taskCount ?? 0
			} else {
				const matching = response.groups.find(g => groupKeyOf(g) === groupKey)
				newTasks = matching?.tasks ?? []
				totalForGroup = matching?.taskCount ?? 0
			}

			if (newTasks.length === 0) {
				allTasksLoadedPerGroup.value[groupKey] = true
				return
			}

			appendTasksToGroup(groupKey, newTasks)
			loadedPagesPerGroup.value[groupKey] = page

			// Mark "all loaded" once we've pulled every task Vikunja knows
			// about for this group, or once a page comes back short.
			const currentGroup: ILabeledGroup | null | undefined = groupKey === UNTAGGED_GROUP_KEY
				? untaggedGroup.value
				: groups.value[findGroupIndex(groupKey)]
			const loadedAllKnown = totalForGroup > 0
				&& typeof currentGroup !== 'undefined'
				&& currentGroup !== null
				&& currentGroup.tasks.length >= totalForGroup
			if (loadedAllKnown || newTasks.length < TASKS_PER_GROUP) {
				allTasksLoadedPerGroup.value[groupKey] = true
			}

			return newTasks
		} finally {
			cancel()
		}
	}

	return {
		groups,
		untaggedGroup,
		isLoading: readonly(isLoading),
		loadedPagesPerGroup,
		allTasksLoadedPerGroup,

		setGroups,
		addTaskToGroup,
		removeTaskFromGroup,
		moveTaskBetweenGroups,
		loadGroups,
		loadMoreForGroup,
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useLabeledStore, import.meta.hot))
}
