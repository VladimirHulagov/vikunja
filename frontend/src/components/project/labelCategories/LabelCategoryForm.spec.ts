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
			project: {
				labeled: {
					categories: {
						nameLabel: 'project.labeled.categories.nameLabel',
						namePlaceholder: 'project.labeled.categories.namePlaceholder',
						labels: 'project.labeled.categories.labels',
						save: 'project.labeled.categories.save',
						cancel: 'project.labeled.categories.cancel',
						empty: 'project.labeled.categories.empty',
					},
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
}

vi.mock('@/stores/labels', () => ({
	useLabelStore: () => labelStoreMock,
}))

const categoryStoreMock = {
	create: vi.fn(),
	update: vi.fn(),
}

vi.mock('@/stores/labelCategories', () => ({
	useLabelCategoriesStore: () => categoryStoreMock,
}))

vi.mock('@/components/input/Button.vue', () => ({
	default: defineComponent({
		name: 'XButtonStub',
		props: ['variant', 'icon', 'disabled', 'loading', 'shadow'],
		emits: ['click'],
		render() {
			return h('button', {
				class: ['x-button-stub', `variant-${this.variant ?? 'primary'}`],
				disabled: this.disabled,
				onClick: () => this.$emit('click'),
			}, this.$slots.default?.())
		},
	}),
}))

import LabelCategoryForm from '@/components/project/labelCategories/LabelCategoryForm.vue'

function resetMockStores() {
	projectLabelsState.value = []
	labelStoreMock.isLoadingProjectLabels = false
	labelStoreMock.loadLabelsForProject.mockClear()
	labelStoreMock.loadLabelsForProject.mockImplementation(async () => projectLabelsState.value)
	labelStoreMock.invalidateProjectLabels.mockClear()
	categoryStoreMock.create.mockClear()
	categoryStoreMock.create.mockResolvedValue({})
	categoryStoreMock.update.mockClear()
	categoryStoreMock.update.mockResolvedValue({})
}

function mountComponent(props: Partial<InstanceType<typeof LabelCategoryForm>['$props']> = {}) {
	return mount(LabelCategoryForm, {
		props: {
			projectId: 1,
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

describe('LabelCategoryForm.vue', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		resetMockStores()
	})

	it('loads project-scoped labels on mount', async () => {
		projectLabelsState.value = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
			{id: 2, title: 'Bath', hexColor: '', description: '', textColor: ''},
		]

		const wrapper = mountComponent()
		await flushPromises()

		expect(labelStoreMock.loadLabelsForProject).toHaveBeenCalledWith(1)
		const chips = wrapper.findAll('.label-chip')
		expect(chips).toHaveLength(2)
		expect(chips[0].text()).toContain('Kitchen')
		expect(chips[1].text()).toContain('Bath')
	})

	it('shows the empty state when project has no labels', async () => {
		projectLabelsState.value = []

		const wrapper = mountComponent()
		await flushPromises()

		expect(wrapper.findAll('.label-chip')).toHaveLength(0)
		expect(wrapper.text()).toContain('project.labeled.categories.empty')
	})

	it('disables save until a title is entered', async () => {
		projectLabelsState.value = []
		const wrapper = mountComponent()
		await flushPromises()

		const saveBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.save'))!
		expect(saveBtn.attributes('disabled')).toBeDefined()

		await wrapper.find('input.input').setValue('Rooms')
		expect(saveBtn.attributes('disabled')).toBeUndefined()
	})

	it('selects and deselects labels via chip clicks', async () => {
		projectLabelsState.value = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
			{id: 2, title: 'Bath', hexColor: '', description: '', textColor: ''},
		]
		const wrapper = mountComponent()
		await flushPromises()

		const chips = wrapper.findAll('.label-chip')
		await chips[0].trigger('click')
		expect(chips[0].classes()).toContain('is-selected')

		await chips[0].trigger('click')
		// Vue re-renders the chip list; query again to read fresh class state
		const refreshed = wrapper.findAll('.label-chip')
		expect(refreshed[0].classes()).not.toContain('is-selected')
	})

	it('pre-selects labels from the existing category in edit mode', async () => {
		projectLabelsState.value = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
			{id: 2, title: 'Bath', hexColor: '', description: '', textColor: ''},
		]
		const existing = {
			id: 10,
			title: 'Rooms',
			projectId: 1,
			position: 0,
			labels: [{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''}],
		} as unknown as InstanceType<typeof LabelCategoryForm>['$props']['category']

		const wrapper = mountComponent({category: existing})
		await flushPromises()

		const chips = wrapper.findAll('.label-chip')
		expect(chips[0].classes()).toContain('is-selected')
		expect(chips[1].classes()).not.toContain('is-selected')
		expect((wrapper.find('input.input').element as HTMLInputElement).value).toBe('Rooms')
	})

	it('creates via the store and invalidates the project label cache on save', async () => {
		projectLabelsState.value = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
		]
		const wrapper = mountComponent()
		await flushPromises()

		await wrapper.find('input.input').setValue('Rooms')
		const chips = wrapper.findAll('.label-chip')
		await chips[0].trigger('click')

		const saveBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.save'))!
		await saveBtn.trigger('click')
		await flushPromises()

		expect(categoryStoreMock.create).toHaveBeenCalledWith(
			1,
			'Rooms',
			[expect.objectContaining({id: 1, title: 'Kitchen'})],
		)
		expect(labelStoreMock.invalidateProjectLabels).toHaveBeenCalledWith(1)
		expect(wrapper.emitted('saved')).toBeTruthy()
	})

	it('updates via the store and invalidates the project label cache on save', async () => {
		projectLabelsState.value = [
			{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''},
			{id: 2, title: 'Bath', hexColor: '', description: '', textColor: ''},
		]
		const existing = {
			id: 10,
			title: 'Rooms',
			projectId: 1,
			position: 0,
			labels: [{id: 1, title: 'Kitchen', hexColor: '', description: '', textColor: ''}],
		} as unknown as InstanceType<typeof LabelCategoryForm>['$props']['category']

		const wrapper = mountComponent({category: existing})
		await flushPromises()

		// Add Bath
		const chips = wrapper.findAll('.label-chip')
		await chips[1].trigger('click')

		const saveBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.save'))!
		await saveBtn.trigger('click')
		await flushPromises()

		expect(categoryStoreMock.update).toHaveBeenCalledWith(
			1,
			10,
			'Rooms',
			expect.arrayContaining([
				expect.objectContaining({id: 1}),
				expect.objectContaining({id: 2}),
			]),
		)
		expect(labelStoreMock.invalidateProjectLabels).toHaveBeenCalledWith(1)
	})

	it('reloads labels when projectId prop changes', async () => {
		projectLabelsState.value = []
		const wrapper = mountComponent({projectId: 1})
		await flushPromises()

		labelStoreMock.loadLabelsForProject.mockClear()
		await wrapper.setProps({projectId: 2})
		await flushPromises()

		expect(labelStoreMock.loadLabelsForProject).toHaveBeenCalledWith(2)
	})
})
