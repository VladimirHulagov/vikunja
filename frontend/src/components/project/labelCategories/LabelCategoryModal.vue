<template>
	<XButton
		variant="secondary"
		icon="list"
		@click="open"
	>
		{{ $t('project.labeled.columns') }}
	</XButton>

	<Modal
		:enabled="isOpen"
		:overflow="true"
		variant="scrolling"
		@close="close"
	>
		<div class="label-category-modal">
			<header class="modal-header">
				<h3>{{ $t('project.labeled.categories.title') }}</h3>
				<XButton
					v-if="mode === 'list'"
					variant="primary"
					icon="plus"
					:shadow="false"
					@click="startCreate"
				>
					{{ $t('project.labeled.categories.add') }}
				</XButton>
			</header>

			<div
				v-if="mode === 'list'"
				class="category-list"
			>
				<p
					v-if="categories.length === 0"
					class="has-text-grey"
				>
					{{ $t('project.labeled.categories.empty') }}
				</p>

				<div
					v-for="cat in categories"
					:key="cat.id"
					class="category-row"
				>
					<div class="category-info">
						<span class="category-title">{{ cat.title }}</span>
						<span class="category-count">{{ countOf(cat) }}</span>
					</div>
					<div class="category-actions">
						<XButton
							variant="secondary"
							icon="pen"
							:shadow="false"
							@click="startEdit(cat)"
						>
							{{ $t('project.labeled.categories.edit') }}
						</XButton>
						<XButton
							variant="tertiary"
							class="has-text-danger"
							icon="trash-alt"
							:shadow="false"
							@click="confirmDelete(cat)"
						>
							{{ $t('project.labeled.categories.delete') }}
						</XButton>
					</div>
				</div>
			</div>

			<LabelCategoryForm
				v-else-if="mode === 'form'"
				:project-id="projectId"
				:category="editingCategory"
				@saved="onSaved"
				@cancel="cancelForm"
			/>
		</div>
	</Modal>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'

import Modal from '@/components/misc/Modal.vue'
import XButton from '@/components/input/Button.vue'
import LabelCategoryForm from './LabelCategoryForm.vue'

import {useLabelCategoriesStore} from '@/stores/labelCategories'

import {error as showError, success as showSuccess} from '@/message'
import {i18n} from '@/i18n'

import type {ILabelCategory} from '@/modelTypes/ILabelCategory'

const props = defineProps<{
	projectId: number,
}>()

const emit = defineEmits<{
	changed: [],
}>()

const categoriesStore = useLabelCategoriesStore()

const isOpen = ref(false)
type Mode = 'list' | 'form'
const mode = ref<Mode>('list')
const editingCategory = ref<ILabelCategory | null>(null)

const categories = computed<readonly ILabelCategory[]>(() => categoriesStore.categories as readonly ILabelCategory[])

function countOf(cat: ILabelCategory): number {
	return cat.labelCount ?? (cat.labels?.length ?? 0)
}

async function open() {
	if (!props.projectId) {
		return
	}
	try {
		await categoriesStore.load(props.projectId)
	} catch (e) {
		showError(e)
	}
	mode.value = 'list'
	editingCategory.value = null
	isOpen.value = true
}

function close() {
	isOpen.value = false
	mode.value = 'list'
	editingCategory.value = null
}

function startCreate() {
	editingCategory.value = null
	mode.value = 'form'
}

function startEdit(cat: ILabelCategory) {
	editingCategory.value = cat
	mode.value = 'form'
}

function cancelForm() {
	editingCategory.value = null
	mode.value = 'list'
}

async function onSaved() {
	// Refresh the in-modal list so the user sees the mutation immediately.
	try {
		await categoriesStore.load(props.projectId)
	} catch (e) {
		showError(e)
	}
	editingCategory.value = null
	mode.value = 'list'
	emit('changed')
}

async function confirmDelete(cat: ILabelCategory) {
	if (typeof window !== 'undefined') {
		const message = i18n.global.t('project.labeled.categories.deleteConfirm', {title: cat.title})
		if (!window.confirm(message)) {
			return
		}
	}

	try {
		await categoriesStore.remove(props.projectId, cat.id)
		showSuccess({message: i18n.global.t('project.labeled.categories.delete')})
		emit('changed')
	} catch (e) {
		showError(e)
	}
}
</script>

<style lang="scss" scoped>
.label-category-modal {
	display: flex;
	flex-direction: column;
	gap: 1rem;
	padding: 1rem 0;

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;

		h3 {
			margin: 0;
			font-size: 1.25rem;
			font-weight: 700;
		}
	}

	.category-list {
		display: flex;
		flex-direction: column;
		gap: .5rem;
	}

	.category-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: .5rem .75rem;
		border-radius: $radius;
		background: var(--grey-100);

		.category-info {
			display: flex;
			align-items: center;
			gap: .5rem;

			.category-title {
				font-weight: 600;
			}

			.category-count {
				color: var(--grey-500);
				font-size: .85rem;
			}
		}

		.category-actions {
			display: flex;
			gap: .35rem;
		}
	}
}
</style>
