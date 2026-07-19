import {beforeEach, describe, expect, it, vi} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createPinia, setActivePinia} from 'pinia'
import {defineComponent, h, reactive} from 'vue'
import {createI18n} from 'vue-i18n'

const i18nStub = createI18n({
	legacy: false,
	locale: 'en',
	messages: {
		en: {
			task: {
				label: {
					placeholder: 'task.label.placeholder',
					createPlaceholder: 'task.label.createPlaceholder',
					addSuccess: 'task.label.addSuccess',
					removeSuccess: 'task.label.removeSuccess',
					addCreateSuccess: 'task.label.addCreateSuccess',
				},
			},
		},
	},
})

vi.mock('vue-i18n', async () => {
	const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
	return {
		...actual,
		useI18n: () => ({
			t: (key: string) => key,
			te: () => true,
		}),
	}
})

vi.mock('@kyvg/vue3-notification', () => ({
	notify: vi.fn(),
}))

vi.mock('@/message', () => ({
	success: vi.fn(),
	error: vi.fn(),
}))

interface ILabelShape {
	id: number
	title: string
	hexColor: string
	description: string
	textColor: string
}

const projectLabelsState = reactive<{value: ILabelShape[]}>({
	value: [],
})

const labelStoreMock = {
	isLoadingProjectLabels: false,
	loadLabelsForProject: vi.fn(async () => projectLabelsState.value),
	invalidateProjectLabels: vi.fn(),
	createLabel: vi.fn(),
}

vi.mock('@/stores/labels', () => ({
	useLabelStore: () => labelStoreMock,
}))

const taskStoreMock = {
	addLabel: vi.fn(),
	removeLabel: vi.fn(),
}

vi.mock('@/stores/tasks', () => ({
	useTaskStore: () => taskStoreMock,
}))

// Stub Multiselect so we don't pull in its full rendering pipeline. We
// re-emit `search`, `select`, and `create` events through dedicated slots
// the test can trigger.
vi.mock('@/components/input/Multiselect.vue', () => ({
	default: defineComponent({
		name: 'MultiselectStub',
		props: [
			'modelValue',
			'loading',
			'placeholder',
			'multiple',
			'searchResults',
			'label',
			'creatable',
			'createPlaceholder',
			'searchDelay',
			'closeAfterSelect',
			'disabled',
		],
		emits: ['update:modelValue', 'search', 'select', 'create'],
		render() {
			return h('div', {class: 'multiselect-stub'}, [
				h('ul', {class: 'search-results'}, (this.searchResults ?? []).map((label: ILabelShape) => h(
					'li',
					{
						key: label.id,
						class: 'search-result',
						'data-id': label.id,
					},
					label.title,
				))),
				h('button', {
					class: 'multiselect-stub-search',
					onClick: () => this.$emit('search', 'kitchen'),
				}, 'search-kitchen'),
				h('button', {
					class: 'multiselect-stub-select',
					onClick: () => {
						const first = (this.searchResults ?? [])[0]
						if (first) {
							this.$emit('select', first)
						}
					},
				}, 'select-first'),
				h('button', {
					class: 'multiselect-stub-create',
					onClick: () => this.$emit('create', 'New Label'),
				}, 'create'),
			])
		},
	}),
}))

vi.mock('@/composables/useLabelStyles', () => ({
	useLabelStyles: () => ({
		getLabelStyles: () => ({}),
	}),
}))

import EditLabels from '@/components/tasks/partials/EditLabels.vue'

function resetMockStores() {
	projectLabelsState.value = []
	labelStoreMock.isLoadingProjectLabels = false
	labelStoreMock.loadLabelsForProject.mockClear()
	labelStoreMock.loadLabelsForProject.mockImplementation(async () => projectLabelsState.value)
	labelStoreMock.invalidateProjectLabels.mockClear()
	labelStoreMock.createLabel.mockClear()
	labelStoreMock.createLabel.mockImplementation(async (label: ILabelShape) => ({...label, id: 999}))
	taskStoreMock.addLabel.mockClear()
	taskStoreMock.removeLabel.mockClear()
}

function mountComponent(props: Partial<InstanceType<typeof EditLabels>['$props']> = {}) {
	return mount(EditLabels, {
		props: {
			modelValue: [],
			taskId: 1,
			projectId: 5,
			...props,
		},
		global: {
			plugins: [createPinia(), i18nStub],
			stubs: {
				transition: false,
				transitionGroup: false,
			},
		},
	})
}

describe('EditLabels.vue', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		resetMockStores()
	})

	it('loads project-scoped labels on mount when projectId is set', async () => {
		projectLabelsState.value = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
		]
		const wrapper = mountComponent()
		await flushPromises()

		// On mount we fire the unfiltered load (search defaults to '').
		expect(labelStoreMock.loadLabelsForProject).toHaveBeenCalledWith(5, '')
		expect(labelStoreMock.loadLabelsForProject.mock.calls[0]).toEqual([5, ''])
		// search results should come from projectLabelsState
		const results = wrapper.findAll('.search-result')
		expect(results).toHaveLength(1)
		expect(results[0].text()).toContain('Kitchen')
	})

	it('does NOT call the loader when projectId is 0 (no project context)', async () => {
		const wrapper = mountComponent({projectId: 0})
		await flushPromises()

		expect(labelStoreMock.loadLabelsForProject).not.toHaveBeenCalled()
		const results = wrapper.findAll('.search-result')
		expect(results).toHaveLength(0)
	})

	it('debounces and re-queries the loader when the user types', async () => {
		vi.useFakeTimers()
		try {
			projectLabelsState.value = [
				{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
			]
			const wrapper = mountComponent()
			await flushPromises()

			labelStoreMock.loadLabelsForProject.mockClear()

			await wrapper.find('.multiselect-stub-search').trigger('click')
			// search event fires synchronously updating the query, but the
			// debounced fetch is pending — fast-forward timers to flush it.
			await vi.advanceTimersByTimeAsync(400)

			expect(labelStoreMock.loadLabelsForProject).toHaveBeenCalledWith(5, 'kitchen')
		} finally {
			vi.useRealTimers()
		}
	})

	it('invalidates the project cache after adding a label so the next open sees it', async () => {
		projectLabelsState.value = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
		]
		const wrapper = mountComponent()
		await flushPromises()

		await wrapper.find('.multiselect-stub-select').trigger('click')
		await flushPromises()

		expect(taskStoreMock.addLabel).toHaveBeenCalledWith({label: expect.objectContaining({id: 1}), taskId: 1})
		expect(labelStoreMock.invalidateProjectLabels).toHaveBeenCalledWith(5)
	})

	it('invalidates the project cache after creating a new label inline', async () => {
		projectLabelsState.value = []
		const wrapper = mountComponent()
		await flushPromises()

		labelStoreMock.invalidateProjectLabels.mockClear()
		await wrapper.find('.multiselect-stub-create').trigger('click')
		await flushPromises()

		expect(labelStoreMock.createLabel).toHaveBeenCalled()
		expect(taskStoreMock.addLabel).toHaveBeenCalled()
		expect(labelStoreMock.invalidateProjectLabels).toHaveBeenCalledWith(5)
	})

	it('removes a label from the task and invalidates the project cache', async () => {
		const initial = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
			{id: 2, title: 'Bath', hexColor: '', description: '', textColor: ''},
		]
		const wrapper = mountComponent({modelValue: initial as unknown as InstanceType<typeof EditLabels>['$props']['modelValue']})
		await flushPromises()

		// We trigger remove via the tag template, but the stub doesn't
		// render the tag slot — so we exercise removeLabel through a
		// direct component method call to verify the wiring.
		const vm = wrapper.vm as unknown as {
			removeLabel: (label: ILabelShape) => Promise<void>
		}
		await vm.removeLabel(initial[0])
		await flushPromises()

		expect(taskStoreMock.removeLabel).toHaveBeenCalledWith({label: expect.objectContaining({id: 1}), taskId: 1})
		expect(labelStoreMock.invalidateProjectLabels).toHaveBeenCalledWith(5)
	})
})
