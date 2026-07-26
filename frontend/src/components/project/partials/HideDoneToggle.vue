<template>
	<FancyCheckbox
		v-model="hideDone"
		class="hide-done-toggle"
	>
		{{ $t('project.views.hideDone') }}
	</FancyCheckbox>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useRouteQuery} from '@vueuse/router'

import FancyCheckbox from '@/components/input/FancyCheckbox.vue'

// Matches `done = false` either standalone or inside an && chain.
const DONE_FALSE_RE = /(^|\s)&&\s*done\s*=\s*false(?=\s|$)|(^|\s)done\s*=\s*false(\s*&&)?/i

const filter = useRouteQuery('filter')

// Reads/writes the `?filter=` URL query param. Every project view
// (List/Gantt/Table/Kanban/Labeled) already syncs its local filter state
// from this URL param, so editing it here propagates to whatever view is
// currently active.
const hideDone = computed<boolean>({
	get() {
		const f = (filter.value ?? '').toString().trim()
		if (f === '') return false
		return /\bdone\s*=\s*false\b/i.test(f)
	},
	set(v: boolean) {
		const raw = filter.value
		const f = (Array.isArray(raw) ? raw[0] ?? '' : raw ?? '').toString().trim()
		let next: string
		if (v) {
			next = f === '' ? 'done = false' : `(${f}) && done = false`
		} else {
			next = f
				.replace(DONE_FALSE_RE, '$1')
				.replace(/\(\s*\)&&/i, '')
				.replace(/\(\s*&&/i, '(')
				.trim()
				.replace(/^\(\s*(.*?)\s*\)$/, '$1')
				.trim()
		}
		filter.value = next === '' ? undefined : next
	},
})
</script>

<style lang="scss" scoped>
.hide-done-toggle {
	margin: 0;
}
</style>
