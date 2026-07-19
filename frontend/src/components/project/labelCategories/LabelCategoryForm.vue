<template>
	<div class="label-category-form">
		<div class="field">
			<label class="label">{{ $t('project.labeled.categories.nameLabel') }}</label>
			<div class="control">
				<input
					v-model="title"
					class="input"
					type="text"
					:placeholder="$t('project.labeled.categories.namePlaceholder')"
				>
			</div>
		</div>

		<div class="field">
			<label class="label">{{ $t('project.labeled.categories.labels') }}</label>
			<div
				v-if="projectLabels.length > 0"
				class="label-chips"
			>
				<button
					v-for="label in projectLabels"
					:key="label.id"
					type="button"
					class="label-chip"
					:class="{'is-selected': isSelected(label.id)}"
					:style="chipStyle(label)"
					@click="toggleLabel(label.id)"
				>
					<span
						class="color-dot"
						:style="{background: label.hexColor || '#999'}"
					/>
					{{ label.title }}
				</button>
			</div>
			<p
				v-else
				class="has-text-grey"
			>
				{{ $t('project.labeled.categories.empty') }}
			</p>
		</div>

		<div class="actions">
			<XButton
				variant="tertiary"
				@click="$emit('cancel')"
			>
				{{ $t('project.labeled.categories.cancel') }}
			</XButton>
			<XButton
				variant="primary"
				:shadow="false"
				:disabled="!canSave || isSaving"
				:loading="isSaving"
				@click="save"
			>
				{{ $t('project.labeled.categories.save') }}
			</XButton>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue'

import XButton from '@/components/input/Button.vue'

import {useLabelStore} from '@/stores/labels'
import {useLabelCategoriesStore} from '@/stores/labelCategories'

import {error as showError} from '@/message'

import type {ILabelCategory} from '@/modelTypes/ILabelCategory'
import type {ILabel} from '@/modelTypes/ILabel'

const props = defineProps<{
	projectId: number,
	category?: ILabelCategory | null,
}>()

const emit = defineEmits<{
	saved: [],
	cancel: [],
}>()

const labelStore = useLabelStore()
const categoriesStore = useLabelCategoriesStore()

const title = ref('')
const selectedLabelIds = ref<Set<number>>(new Set())
const isSaving = ref(false)

const isEditMode = computed(() => !!props.category && typeof props.category.id === 'number' && props.category.id > 0)

const projectLabels = computed<ILabel[]>(() => {
	// Vikunja labels are user-scoped (created by a user, usable on any task).
	// The `projectId` field on a label is just where it was originally created,
	// not where it's used. Show the user's full label library so they can pick
	// any of their existing labels for this category.
	return (labelStore.labelsArray as readonly ILabel[]) as ILabel[]
})

const canSave = computed(() => title.value.trim() !== '')

onMounted(async () => {
	// Make sure all labels are available for selection.
	try {
		await labelStore.loadAllLabels()
	} catch (e) {
		showError(e)
	}

	if (props.category) {
		title.value = props.category.title ?? ''
		selectedLabelIds.value = new Set(
			(props.category.labels ?? []).map(l => l.id),
		)
	}
})

function isSelected(labelId: number): boolean {
	return selectedLabelIds.value.has(labelId)
}

function toggleLabel(labelId: number) {
	const next = new Set(selectedLabelIds.value)
	if (next.has(labelId)) {
		next.delete(labelId)
	} else {
		next.add(labelId)
	}
	selectedLabelIds.value = next
}

function chipStyle(label: ILabel): Record<string, string> {
	if (!isSelected(label.id) || !label.hexColor) {
		return {}
	}
	return {
		backgroundColor: label.hexColor,
		color: label.textColor || '#fff',
	}
}

async function save() {
	if (!canSave.value || isSaving.value) {
		return
	}

	const selectedLabels: ILabel[] = projectLabels.value
		.filter(l => selectedLabelIds.value.has(l.id))

	isSaving.value = true
	try {
		if (isEditMode.value && props.category) {
			await categoriesStore.update(
				props.projectId,
				props.category.id,
				title.value.trim(),
				selectedLabels,
			)
		} else {
			await categoriesStore.create(
				props.projectId,
				title.value.trim(),
				selectedLabels,
			)
		}
		emit('saved')
	} catch (e) {
		showError(e)
	} finally {
		isSaving.value = false
	}
}
</script>

<style lang="scss" scoped>
.label-category-form {
	display: flex;
	flex-direction: column;
	gap: 1rem;

	.label-chips {
		display: flex;
		flex-wrap: wrap;
		gap: .5rem;
	}

	.label-chip {
		display: inline-flex;
		align-items: center;
		gap: .35rem;
		padding: .25rem .5rem;
		border-radius: $radius;
		border: 1px solid var(--grey-300);
		background: var(--grey-100);
		cursor: pointer;
		font-size: .85rem;

		&.is-selected {
			border-color: transparent;
			font-weight: 600;
		}

		.color-dot {
			inline-size: 10px;
			block-size: 10px;
			border-radius: 50%;
			display: inline-block;
		}
	}

	.actions {
		display: flex;
		justify-content: flex-end;
		gap: .5rem;
	}
}
</style>
