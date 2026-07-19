import {readonly, ref} from 'vue'
import {acceptHMRUpdate, defineStore} from 'pinia'

import LabelCategoryService from '@/services/labelCategory'
import {setModuleLoading} from '@/stores/helper'

import type {ILabelCategory} from '@/modelTypes/ILabelCategory'
import type {ILabel} from '@/modelTypes/ILabel'

/**
 * Holds the label categories (группы меток) for the currently active project.
 *
 * Categories are a per-project grouping construct used by the Labeled view to
 * filter which label-columns are shown. The "active" filter lives in the URL
 * (`?category=N`), not in this store.
 */
export const useLabelCategoriesStore = defineStore('labelCategories', () => {
	const categories = ref<ILabelCategory[]>([])
	const isLoading = ref(false)

	function setIsLoading(newIsLoading: boolean) {
		isLoading.value = newIsLoading
	}

	async function load(projectId: number): Promise<ILabelCategory[]> {
		if (!projectId) {
			categories.value = []
			return []
		}

		const cancel = setModuleLoading(setIsLoading)
		const service = new LabelCategoryService()
		try {
			const result = await service.getAll(
				{projectId} as unknown as ILabelCategory,
			)
			categories.value = result
			return result
		} finally {
			cancel()
		}
	}

	async function create(
		projectId: number,
		title: string,
		labels: ILabel[],
	): Promise<ILabelCategory> {
		const cancel = setModuleLoading(setIsLoading)
		const service = new LabelCategoryService()
		try {
			const created = await service.create({
				projectId,
				title,
				labels,
			} as ILabelCategory)
			categories.value = [...categories.value, created]
			return created
		} finally {
			cancel()
		}
	}

	async function update(
		projectId: number,
		id: number,
		title: string,
		labels: ILabel[],
	): Promise<ILabelCategory> {
		const cancel = setModuleLoading(setIsLoading)
		const service = new LabelCategoryService()
		try {
			const updated = await service.update({
				id,
				projectId,
				title,
				labels,
			} as ILabelCategory)
			categories.value = categories.value.map(c => (c.id === id ? updated : c))
			return updated
		} finally {
			cancel()
		}
	}

	async function remove(projectId: number, id: number): Promise<void> {
		const cancel = setModuleLoading(setIsLoading)
		const service = new LabelCategoryService()
		try {
			await service.delete({id, projectId} as ILabelCategory)
			categories.value = categories.value.filter(c => c.id !== id)
		} finally {
			cancel()
		}
	}

	/**
	 * Returns the set of label IDs that belong to at least one category in
	 * the given project. Used by the cloud to compute the "Без категории"
	 * (uncategorized) count.
	 */
	function getLabelsInAnyCategory(projectId: number): Set<number> {
		void projectId
		const ids = new Set<number>()
		for (const cat of categories.value) {
			for (const label of cat.labels ?? []) {
				ids.add(label.id)
			}
		}
		return ids
	}

	return {
		categories,
		isLoading: readonly(isLoading),

		load,
		create,
		update,
		remove,
		getLabelsInAnyCategory,
	}
})

// support hot reloading
if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useLabelCategoriesStore, import.meta.hot))
}
