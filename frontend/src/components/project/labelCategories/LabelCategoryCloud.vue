<template>
	<div
		v-if="shouldShow"
		class="label-category-cloud"
	>
		<button
			type="button"
			class="category-chip"
			:class="{'is-active': activeCategoryId === ALL}"
			@click="$emit('select', ALL)"
		>
			{{ $t('project.labeled.categories.all') }}
		</button>

		<button
			v-for="cat in categories"
			:key="cat.id"
			type="button"
			class="category-chip"
			:class="{'is-active': activeCategoryId === cat.id}"
			@click="$emit('select', cat.id)"
		>
			{{ cat.title }}
			<span class="count">{{ countOf(cat) }}</span>
		</button>

		<button
			v-if="uncategorizedCount > 0"
			type="button"
			class="category-chip"
			:class="{'is-active': activeCategoryId === UNCATEGORIZED}"
			@click="$emit('select', UNCATEGORIZED)"
		>
			{{ $t('project.labeled.categories.uncategorized') }}
			<span class="count">{{ uncategorizedCount }}</span>
		</button>
	</div>
</template>

<script setup lang="ts">
import {computed, onMounted, watch} from 'vue'

import {useLabelCategoriesStore} from '@/stores/labelCategories'
import {useLabelStore} from '@/stores/labels'

import type {ILabelCategory} from '@/modelTypes/ILabelCategory'

const props = defineProps<{
	projectId: number,
	activeCategoryId: number,
}>()

defineEmits<{
	select: [id: number],
}>()

// Sentinel values for the virtual chips. `0` is "all labels",
// `-1` is the "uncategorized" virtual chip.
const ALL = 0
const UNCATEGORIZED = -1

const categoriesStore = useLabelCategoriesStore()
const labelStore = useLabelStore()

const categories = computed<readonly ILabelCategory[]>(() => categoriesStore.categories as readonly ILabelCategory[])

const projectLabels = computed(() => {
	const all = labelStore.labelsArray as unknown as Array<{id: number, projectId: number}>
	return all.filter(l => l.projectId === props.projectId)
})

const categorizedIds = computed(() => categoriesStore.getLabelsInAnyCategory(props.projectId))

const uncategorizedCount = computed(() => {
	const categorized = categorizedIds.value
	let count = 0
	for (const label of projectLabels.value) {
		if (!categorized.has(label.id)) {
			count++
		}
	}
	return count
})

const shouldShow = computed(() => {
	if (categories.value.length > 0) {
		return true
	}
	// Show the cloud if there are uncategorized labels in this project
	// (so the user can pick the "Без категории" filter); hide it when the
	// project has no labels at all.
	return uncategorizedCount.value > 0
})

function countOf(cat: ILabelCategory): number {
	return cat.labelCount ?? (cat.labels?.length ?? 0)
}

async function reload() {
	if (!props.projectId) {
		return
	}
	try {
		await categoriesStore.load(props.projectId)
	} catch {
		// Errors are surfaced via the store's loading state; we don't pop
		// a duplicate toast here.
	}
}

onMounted(reload)

watch(() => props.projectId, reload)

// Keep the label store loaded so the "uncategorized" count is accurate.
onMounted(async () => {
	if (props.projectId) {
		try {
			await labelStore.loadAllLabels()
		} catch {
			// best-effort; counts will simply be off if this fails.
		}
	}
})
</script>

<style lang="scss" scoped>
.label-category-cloud {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
	align-items: center;
	padding: .5rem 0;
}

.category-chip {
	display: inline-flex;
	align-items: center;
	gap: .35rem;
	padding: .25rem .65rem;
	border-radius: $radius;
	border: 1px solid var(--grey-300);
	background: var(--grey-100);
	cursor: pointer;
	font-size: .85rem;

	&:hover {
		background: var(--grey-200);
	}

	&.is-active {
		background: var(--primary);
		color: var(--grey-900);
		border-color: var(--primary);
		font-weight: 600;
	}

	.count {
		background: rgba(0, 0, 0, .08);
		border-radius: $radius;
		padding: 0 .35rem;
		font-size: .75rem;

		.is-active & {
			background: rgba(255, 255, 255, .25);
			color: inherit;
		}
	}
}
</style>
