import {computed, readonly, ref} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'

import LabelService from '@/services/label'
import {success} from '@/message'
import {i18n} from '@/i18n'
import {setModuleLoading} from '@/stores/helper'
import type {ILabel} from '@/modelTypes/ILabel'

async function getAllLabels(page = 1): Promise<ILabel[]> {
	const labelService = new LabelService()
	const labels  = await labelService.getAll({}, {}, page) as ILabel[]
	if (page < labelService.totalPages) {
		const nextLabels = await getAllLabels(page + 1)
		return labels.concat(nextLabels)
	} else {
		return labels
	}
}

// Walks every page of project-scoped labels in parallel until the service
// reports no further pages. Mirrors getAllLabels, but seeds the request with
// `projectId` so the backend filters to labels actually attached to tasks in
// that project.
async function getAllProjectLabels(projectId: number, search: string, page = 1): Promise<ILabel[]> {
	const labelService = new LabelService()
	const labels = await labelService.getAll(
		{},
		{projectId, s: search},
		page,
	) as ILabel[]
	if (page < labelService.totalPages) {
		const nextLabels = await getAllProjectLabels(projectId, search, page + 1)
		return labels.concat(nextLabels)
	}
	return labels
}

export const useLabelStore = defineStore('label', () => {
	const labels = ref<{ [id: ILabel['id']]: ILabel }>({})

	// Per-project cache of "labels used by tasks in project N". Keyed by
	// projectId so multiple projects can coexist without trampling each
	// other. Search results are not cached — only the unfiltered snapshot.
	const projectLabels = ref<Map<number, ILabel[]>>(new Map())

	// Alphabetically sort the labels
	const labelsArray = computed(() => Object.values(labels.value)
		.sort((a, b) => a.title.localeCompare(
			b.title, i18n.global.locale.value,
			{ ignorePunctuation: true },
		)),
	)

	const isLoading = ref(false)
	const isLoadingProjectLabels = ref(false)
	
	const getLabelById = computed(() => {
		return (labelId: ILabel['id']) => labels.value[labelId]
	})

	const getLabelsByIds = computed(() => (ids: ILabel['id'][]) =>
		ids.map(id => labels.value[id]).filter(Boolean),
	)


	// **
	// * Checks if a list of labels is available in the store and filters them then query
	// **
	const filterLabelsByQuery = computed(() => {
		return (labelsToHide: ILabel[], query: string) => {
			if (query === '') return []
			const labelIdsToHide: number[] = labelsToHide.map(({id}) => id)
			const q = query.toLowerCase()
			return labelsArray.value
				.filter(l => !labelIdsToHide.includes(l.id))
				.filter(l => l.title.toLowerCase().includes(q) || (l.description ?? '').toLowerCase().includes(q))
		}
	})

	const getLabelsByExactTitles = computed(() => {
		return (labelTitles: string[]) => labelsArray.value
			.filter(({title}) => labelTitles.some(l => l.toLowerCase() === title.toLowerCase()))
	})
	
	const getLabelByExactTitle = computed(() => {
		return (labelTitle: string) => labelsArray.value
			.find(l => l.title.toLowerCase() === labelTitle.toLowerCase())
	})


	function setIsLoading(newIsLoading: boolean) {
		isLoading.value = newIsLoading
	}

	function setIsLoadingProjectLabels(newIsLoading: boolean) {
		isLoadingProjectLabels.value = newIsLoading
	}

	function setLabels(newLabels: ILabel[]) {
		newLabels.forEach(l => {
			labels.value[l.id] = l
		})
	}

	function setLabel(label: ILabel) {
		labels.value[label.id] = {...label}
	}

	function removeLabelById(label: ILabel) {
		delete labels.value[label.id]
	}

	async function loadAllLabels({forceLoad} : {forceLoad?: boolean} = {}) {
		if (isLoading.value && !forceLoad) {
			return
		}

		const cancel = setModuleLoading(setIsLoading)

		try {
			const newLabels = await getAllLabels()
			setLabels(newLabels)
			return newLabels
		} finally {
			cancel()
		}
	}

	/**
	 * Loads labels that are actually used by tasks in the given project
	 * (backend: `GET /labels?project_id=N`). Results for the unfiltered
	 * pass are cached per-project so the picker doesn't re-fetch on every
	 * open. Search results are returned to the caller but never cached —
	 * they're strictly ephemeral so a stale query can't poison the picker.
	 */
	async function loadLabelsForProject(projectId: number, search: string = ''): Promise<ILabel[]> {
		if (!projectId) {
			return []
		}

		if (!search && projectLabels.value.has(projectId)) {
			return projectLabels.value.get(projectId)!
		}

		const cancel = setModuleLoading(setIsLoadingProjectLabels)

		try {
			const result = await getAllProjectLabels(projectId, search)
			if (!search) {
				const newMap = new Map(projectLabels.value)
				newMap.set(projectId, result)
				projectLabels.value = newMap
			}
			return result
		} finally {
			cancel()
		}
	}

	/**
	 * Drops the cached label list for a project (or the whole cache when
	 * no id is given). Call after creating / deleting / assigning labels
	 * so the next picker open reflects fresh state.
	 */
	function invalidateProjectLabels(projectId?: number) {
		if (projectId) {
			const newMap = new Map(projectLabels.value)
			newMap.delete(projectId)
			projectLabels.value = newMap
		} else {
			projectLabels.value = new Map()
		}
	}

	async function deleteLabel(label: ILabel) {
		const cancel = setModuleLoading(setIsLoading)
		const labelService = new LabelService()

		try {
			const result = await labelService.delete(label)
			removeLabelById(label)
			success({message: i18n.global.t('label.deleteSuccess')})
			return result
		} finally {
			cancel()
		}
	}

	async function updateLabel(label: ILabel) {
		const cancel = setModuleLoading(setIsLoading)
		const labelService = new LabelService()

		try {
			const newLabel = await labelService.update(label)
			setLabel(newLabel)
			success({message: i18n.global.t('label.edit.success')})
			return newLabel
		} finally {
			cancel()
		}
	}

	async function createLabel(label: ILabel) {
		const cancel = setModuleLoading(setIsLoading)
		const labelService = new LabelService()

		try {
			const newLabel = await labelService.create(label) as ILabel
			setLabel(newLabel)
			return newLabel
		} finally {
			cancel()
		}
	}

	return {
		labels: readonly(labels),
		labelsArray: readonly(labelsArray),
		isLoading,
		projectLabels,
		isLoadingProjectLabels,

		getLabelById,
		getLabelsByIds,
		filterLabelsByQuery,
		getLabelsByExactTitles,
		getLabelByExactTitle,

		setLabels,
		setLabel,
		removeLabelById,
		loadAllLabels,
		loadLabelsForProject,
		invalidateProjectLabels,
		deleteLabel,
		updateLabel,
		createLabel,
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useLabelStore, import.meta.hot))
}
