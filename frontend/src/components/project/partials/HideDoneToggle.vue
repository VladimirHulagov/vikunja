<template>
	<FancyCheckbox
		v-model="hideDone"
		class="hide-done-toggle"
	>
		{{ $t('project.views.hideDone') }}
	</FancyCheckbox>
</template>

<script setup lang="ts">
import {computed, onMounted, watch} from 'vue'
import {useRouteQuery} from '@vueuse/router'

import FancyCheckbox from '@/components/input/FancyCheckbox.vue'

import {useAuthStore} from '@/stores/auth'
import type {IUserSettings} from '@/modelTypes/IUserSettings'

// Matches `done = false` either standalone or inside an && chain.
const DONE_FALSE_RE = /(^|\s)&&\s*done\s*=\s*false(?=\s|$)|(^|\s)done\s*=\s*false(\s*&&)?/i

const filter = useRouteQuery('filter')
const authStore = useAuthStore()

function isFilterDoneHidden(f: string): boolean {
	if (!f) return false
	return /\bdone\s*=\s*false\b/i.test(f)
}

function addDoneFalse(f: string): string {
	if (isFilterDoneHidden(f)) return f
	return f === '' ? 'done = false' : `(${f}) && done = false`
}

function removeDoneFalse(f: string): string {
	if (!isFilterDoneHidden(f)) return f
	return f
		.replace(DONE_FALSE_RE, '$1')
		.replace(/\(\s*\)&&/i, '')
		.replace(/\(\s*&&/i, '(')
		.trim()
		.replace(/^\(\s*(.*?)\s*\)$/, '$1')
		.trim()
}

function currentFilterString(): string {
	const raw = filter.value
	return (Array.isArray(raw) ? raw[0] ?? '' : raw ?? '').toString().trim()
}

function setFilterString(next: string) {
	filter.value = next === '' ? undefined : next
}

// Persist the user-level preference to the backend. The settings object
// returned by the auth store is DeepReadonly, so we clone it into a
// mutable copy (matching the pattern in views/user/settings/General.vue).
async function persistUserPref(hidden: boolean) {
	const current = authStore.info?.settings
	if (!current) return
	// Local mutable copy. Spread frontendSettings explicitly because
	// saveUserSettings expects a plain object, not a readonly proxy.
	const update: IUserSettings = {
		...current,
		frontendSettings: {
			...current.frontendSettings,
			hideDoneTasks: hidden,
		},
	}
	await authStore.saveUserSettings({
		settings: update,
		showMessage: false,
	})
}

// The user-level preference is the source of truth (persists across page
// reloads and project switches). Default to true (hide done) for users who
// haven't set the pref yet — matches the Labeled view's default filter.
function readUserPref(): boolean {
	return authStore.info?.settings?.frontendSettings?.hideDoneTasks ?? true
}

// The checkbox binds to `hideDone` — a computed that bridges the user
// preference and the URL filter. Setting it updates BOTH.
const hideDone = computed<boolean>({
	get() {
		// Reflect what's actually in the URL filter (so manual edits via
		// FilterPopup keep the checkbox in sync within a session).
		return isFilterDoneHidden(currentFilterString())
	},
	set(v: boolean) {
		persistUserPref(v)
		const next = v
			? addDoneFalse(currentFilterString())
			: removeDoneFalse(currentFilterString())
		setFilterString(next)
	},
})

// When the URL filter changes externally (e.g. user navigates to a new
// project where the filter is empty, or edits the filter string in
// FilterPopup), reconcile the user preference to match reality.
watch(filter, (raw) => {
	const f = (Array.isArray(raw) ? raw[0] ?? '' : raw ?? '').toString().trim()
	const currentlyHidden = isFilterDoneHidden(f)
	if (currentlyHidden !== readUserPref()) {
		persistUserPref(currentlyHidden)
	}
})

// On mount (e.g. when switching projects or first page load), if the user
// preference says "hide done" but the URL filter doesn't have `done = false`
// yet, inject it. This is what makes the preference global across projects.
onMounted(() => {
	if (readUserPref() && !isFilterDoneHidden(currentFilterString())) {
		setFilterString(addDoneFalse(currentFilterString()))
	}
})
</script>

<style lang="scss" scoped>
.hide-done-toggle {
	margin: 0;
}
</style>
