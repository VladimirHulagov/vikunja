import {beforeEach, describe, expect, it, vi} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createPinia, setActivePinia} from 'pinia'
import {h, reactive} from 'vue'
import {createI18n} from 'vue-i18n'

const i18nStub = createI18n({
	legacy: false,
	locale: 'en',
	messages: {
		en: {
			project: {
				labeled: {
					categories: {
						all: 'project.labeled.categories.all',
						uncategorized: 'project.labeled.categories.uncategorized',
						empty: 'project.labeled.categories.empty',
					},
				},
			},
		},
	},
})

// Mock vue-i18n: many child components read translations during render.
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

// Mock @kyvg/vue3-notification so success()/error() don't crash.
vi.mock('@kyvg/vue3-notification', () => ({
	notify: vi.fn(),
}))

// Stub the message helpers so we don't depend on i18n notification plumbing.
vi.mock('@/message', () => ({
	success: vi.fn(),
	error: vi.fn(),
}))

interface CategoryStoreStub {
	categories: ICategoryShape[]
	isLoading: boolean
	load: ReturnType<typeof vi.fn>
	create: ReturnType<typeof vi.fn>
	update: ReturnType<typeof vi.fn>
	remove: ReturnType<typeof vi.fn>
	getLabelsInAnyCategory: ReturnType<typeof vi.fn>
}

interface ICategoryShape {
	id: number
	title: string
	projectId: number
	position: number
	labels: Array<{id: number, title: string, projectId: number}>
	labelCount?: number
}

interface ILabelShape {
	id: number
	title: string
	projectId: number
}

const categoriesState = reactive<{
	categories: ICategoryShape[]
	categorized: Set<number>
}>({
	categories: [],
	categorized: new Set<number>(),
})

const categoryStoreMock: CategoryStoreStub = {
	isLoading: false,
	load: vi.fn().mockResolvedValue([]),
	create: vi.fn(),
	update: vi.fn(),
	remove: vi.fn(),
	getLabelsInAnyCategory: vi.fn().mockImplementation(() => new Set(categoriesState.categorized)),
	get categories() {
		return categoriesState.categories
	},
	set categories(value: ICategoryShape[]) {
		categoriesState.categories = value
	},
}

vi.mock('@/stores/labelCategories', () => ({
	useLabelCategoriesStore: () => categoryStoreMock,
}))

const labelsState = reactive<{labels: ILabelShape[]}>({
	labels: [],
})

const labelStoreMock = {
	loadAllLabels: vi.fn().mockResolvedValue([]),
	get labelsArray() {
		return labelsState.labels
	},
	set labelsArray(value: ILabelShape[]) {
		labelsState.labels = value
	},
}

vi.mock('@/stores/labels', () => ({
	useLabelStore: () => labelStoreMock,
}))

import LabelCategoryCloud from '@/components/project/labelCategories/LabelCategoryCloud.vue'

function resetMockStores() {
	categoriesState.categories = []
	categoriesState.categorized = new Set()
	labelsState.labels = []
	categoryStoreMock.load.mockClear()
	categoryStoreMock.load.mockResolvedValue([])
	labelStoreMock.loadAllLabels.mockClear()
	labelStoreMock.loadAllLabels.mockResolvedValue([])
}

function mountComponent(props: Partial<InstanceType<typeof LabelCategoryCloud>['$props']> = {}) {
	return mount(LabelCategoryCloud, {
		props: {
			projectId: 1,
			activeCategoryId: 0,
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

describe('LabelCategoryCloud.vue', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		resetMockStores()
	})

	it('renders chips from store state with counts', async () => {
		categoriesState.categories = [
			{id: 10, title: 'Rooms', projectId: 1, position: 0, labels: [
				{id: 1, title: 'Kitchen', projectId: 1},
				{id: 2, title: 'Bath', projectId: 1},
			]},
			{id: 20, title: 'Work', projectId: 1, position: 1, labels: [
				{id: 3, title: 'Plumbing', projectId: 1},
			]},
		]

		const wrapper = mountComponent()
		await flushPromises()

		const chips = wrapper.findAll('.category-chip')
		// 1 "all" + 2 categories + 0 uncategorized (no labels in store)
		expect(chips.length).toBe(3)

		const texts = chips.map(c => c.text())
		expect(texts[0]).toContain('project.labeled.categories.all')
		expect(texts[1]).toContain('Rooms')
		expect(texts[1]).toContain('2')
		expect(texts[2]).toContain('Work')
		expect(texts[2]).toContain('1')
	})

	it('highlights the active chip via is-active class', async () => {
		categoriesState.categories = [
			{id: 10, title: 'Rooms', projectId: 1, position: 0, labels: []},
		]

		const wrapper = mountComponent({activeCategoryId: 10})
		await flushPromises()

		const chips = wrapper.findAll('.category-chip')
		expect(chips[0].classes()).not.toContain('is-active')
		expect(chips[1].classes()).toContain('is-active')
	})

	it('emits the correct id when a chip is clicked', async () => {
		categoriesState.categories = [
			{id: 10, title: 'Rooms', projectId: 1, position: 0, labels: []},
			{id: 20, title: 'Work', projectId: 1, position: 1, labels: []},
		]

		const wrapper = mountComponent()
		await flushPromises()

		const chips = wrapper.findAll('.category-chip')
		await chips[0].trigger('click') // "All"
		await chips[1].trigger('click') // category 10
		await chips[2].trigger('click') // category 20

		const emitted = wrapper.emitted('select')
		expect(emitted).toBeTruthy()
		expect(emitted).toHaveLength(3)
		expect(emitted![0]).toEqual([0])
		expect(emitted![1]).toEqual([10])
		expect(emitted![2]).toEqual([20])
	})

	it('hides the entire cloud when there are no categories and no uncategorized labels', async () => {
		categoriesState.categories = []
		labelsState.labels = []

		const wrapper = mountComponent()
		await flushPromises()

		expect(wrapper.find('.label-category-cloud').exists()).toBe(false)
	})

	it('computes the "Uncategorized" count from project labels minus categorized labels', async () => {
		// Category 10 owns labels 1, 2; project also has labels 3 and 4 unassigned.
		categoriesState.categories = [
			{id: 10, title: 'Rooms', projectId: 1, position: 0, labels: [
				{id: 1, title: 'Kitchen', projectId: 1},
				{id: 2, title: 'Bath', projectId: 1},
			]},
		]
		categoriesState.categorized = new Set([1, 2])
		labelsState.labels = [
			{id: 1, title: 'Kitchen', projectId: 1},
			{id: 2, title: 'Bath', projectId: 1},
			{id: 3, title: 'Plumbing', projectId: 1},
			{id: 4, title: 'Tiles', projectId: 1},
		]

		const wrapper = mountComponent()
		await flushPromises()

		const chips = wrapper.findAll('.category-chip')
		// 1 "all" + 1 category + 1 uncategorized
		expect(chips.length).toBe(3)
		const uncategorized = chips[2]
		expect(uncategorized.text()).toContain('project.labeled.categories.uncategorized')
		expect(uncategorized.text()).toContain('2')
	})

	it('emits -1 when the "Uncategorized" chip is clicked', async () => {
		categoriesState.categories = []
		categoriesState.categorized = new Set()
		labelsState.labels = [
			{id: 7, title: 'Lonely', projectId: 1},
		]

		const wrapper = mountComponent()
		await flushPromises()

		// Cloud should be visible because there is an uncategorized label.
		const chips = wrapper.findAll('.category-chip')
		expect(chips.length).toBe(2) // "all" + "uncategorized"

		await chips[1].trigger('click')
		const emitted = wrapper.emitted('select')
		expect(emitted).toBeTruthy()
		expect(emitted![0]).toEqual([-1])
	})

	it('reloads categories from the store when projectId changes', async () => {
		categoriesState.categories = []

		const wrapper = mountComponent({projectId: 1})
		await flushPromises()

		expect(categoryStoreMock.load).toHaveBeenCalledWith(1)

		await wrapper.setProps({projectId: 2})
		await flushPromises()

		expect(categoryStoreMock.load).toHaveBeenCalledWith(2)
	})
})

// Reference imports so TS doesn't elide them.
void (null as unknown as ReturnType<typeof h>)
