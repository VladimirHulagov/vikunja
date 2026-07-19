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
import {computed, onMounted, ref, watch} from 'vue'

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

// Available labels = labels actually used by tasks in this project, loaded
// from the project-scoped label endpoint (GET /labels?project_id=N). This
// replaces the previous workaround that derived the list from
// labeledStore.groups, which only worked when the Labeled view had been
// visited and had its groups cached.
const projectLabels = ref<ILabel[]>([])

const isEditMode = computed(() => !!props.category && typeof props.category.id === 'number' && props.category.id > 0)

async function loadProjectLabels() {
	if (!props.projectId) {
		projectLabels.value = []
		return
	}
	try {
		projectLabels.value = await labelStore.loadLabelsForProject(props.projectId)
	} catch {
		// Surface elsewhere; keep the form usable with whatever it had.
	}
}

const canSave = computed(() => title.value.trim() !== '')

onMounted(async () => {
	await loadProjectLabels()
	if (props.category) {
		title.value = props.category.title ?? ''
		selectedLabelIds.value = new Set(
			(props.category.labels ?? []).map(l => l.id),
		)
	}
})

watch(() => props.projectId, loadProjectLabels)

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
	// Also include any labels on the existing category that aren't in the
	// project-scoped list (e.g. they belong to tasks filtered out by the
	// `done = false` default). Without this, editing a category would
	// silently strip them.
	if (props.category?.labels) {
		const present = new Set(projectLabels.value.map(l => l.id))
		for (const label of props.category.labels) {
			if (selectedLabelIds.value.has(label.id) && !present.has(label.id)) {
				selectedLabels.push(label)
			}
		}
	}

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
		// Drop the project-scoped label cache so the next form open picks
		// up any membership changes that the backend will recompute.
		labelStore.invalidateProjectLabels(props.projectId)
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
