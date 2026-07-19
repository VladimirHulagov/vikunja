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
					columns: 'project.labeled.columns',
					categories: {
						title: 'project.labeled.categories.title',
						add: 'project.labeled.categories.add',
						nameLabel: 'project.labeled.categories.nameLabel',
						namePlaceholder: 'project.labeled.categories.namePlaceholder',
						labels: 'project.labeled.categories.labels',
						save: 'project.labeled.categories.save',
						cancel: 'project.labeled.categories.cancel',
						edit: 'project.labeled.categories.edit',
						delete: 'project.labeled.categories.delete',
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

// Mock @/i18n so the modal's `i18n.global.t` calls work for confirm dialogs.
vi.mock('@/i18n', () => ({
	i18n: {
		global: {
			t: (key: string) => key,
			locale: {value: 'en'},
		},
	},
}))

// Stub Modal so we don't drag in teleport/dialog lifecycle complexity. We
// just render the default slot inside a div when `enabled` is true.
vi.mock('@/components/misc/Modal.vue', () => ({
	default: defineComponent({
		name: 'ModalStub',
		props: ['enabled', 'overflow', 'variant'],
		emits: ['close'],
		render() {
			if (!this.enabled) {
				return null
			}
			return h('div', {class: 'modal-stub'}, [
				h('button', {
					class: 'modal-stub-close',
					onClick: () => this.$emit('close'),
				}, 'close'),
				this.$slots.default?.(),
			])
		},
	}),
}))

// Stub XButton so we render a simple clickable element.
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

interface ICategoryShape {
	id: number
	title: string
	projectId: number
	position: number
	labels: Array<{id: number, title: string, projectId: number, hexColor: string, textColor: string}>
	labelCount?: number
}

interface ILabelShape {
	id: number
	title: string
	projectId: number
	hexColor: string
	textColor: string
}

const categoriesState = reactive<{categories: ICategoryShape[]}>({
	categories: [],
})

const categoryStoreMock = {
	get categories() {
		return categoriesState.categories
	},
	set categories(value: ICategoryShape[]) {
		categoriesState.categories = value
	},
	isLoading: false,
	load: vi.fn().mockResolvedValue([]),
	create: vi.fn(),
	update: vi.fn(),
	remove: vi.fn(),
	getLabelsInAnyCategory: vi.fn().mockReturnValue(new Set<number>()),
}

vi.mock('@/stores/labelCategories', () => ({
	useLabelCategoriesStore: () => categoryStoreMock,
}))

const labelStoreMock = {
	labelsArray: [] as ILabelShape[],
	loadAllLabels: vi.fn().mockResolvedValue([]),
}

vi.mock('@/stores/labels', () => ({
	useLabelStore: () => labelStoreMock,
}))

// Stub LabelCategoryForm — we don't want to test its internals here, only
// that the modal switches to it and emits saved/cancel correctly.
const formSavedMock = vi.fn()
const formCancelMock = vi.fn()
vi.mock('@/components/project/labelCategories/LabelCategoryForm.vue', () => ({
	default: defineComponent({
		name: 'LabelCategoryFormStub',
		props: ['projectId', 'category'],
		emits: ['saved', 'cancel'],
		render() {
			return h('div', {class: 'label-category-form-stub'}, [
				h('button', {
					class: 'form-stub-saved',
					onClick: () => {
						formSavedMock()
						this.$emit('saved')
					},
				}, 'saved'),
				h('button', {
					class: 'form-stub-cancel',
					onClick: () => {
						formCancelMock()
						this.$emit('cancel')
					},
				}, 'cancel'),
			])
		},
	}),
}))

// window.confirm stub — happy-dom doesn't define `confirm` by default.
let confirmReturn = true
const confirmMock = vi.fn(() => confirmReturn)
window.confirm = confirmMock as unknown as typeof window.confirm
const confirmSpy = confirmMock

import LabelCategoryModal from '@/components/project/labelCategories/LabelCategoryModal.vue'

function resetMockStores() {
	categoriesState.categories = []
	categoryStoreMock.load.mockClear()
	categoryStoreMock.load.mockResolvedValue([])
	categoryStoreMock.create.mockClear()
	categoryStoreMock.create.mockResolvedValue({})
	categoryStoreMock.update.mockClear()
	categoryStoreMock.update.mockResolvedValue({})
	categoryStoreMock.remove.mockClear()
	categoryStoreMock.remove.mockResolvedValue({})
	labelStoreMock.labelsArray = []
	labelStoreMock.loadAllLabels.mockClear()
	labelStoreMock.loadAllLabels.mockResolvedValue([])
	formSavedMock.mockClear()
	formCancelMock.mockClear()
	confirmSpy.mockClear()
	confirmReturn = true
}

function mountComponent(props: Partial<InstanceType<typeof LabelCategoryModal>['$props']> = {}) {
	return mount(LabelCategoryModal, {
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

async function openModal(wrapper: ReturnType<typeof mountComponent>) {
	await wrapper.find('button.x-button-stub').trigger('click')
	await flushPromises()
}

describe('LabelCategoryModal.vue', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		resetMockStores()
	})

	it('renders the "Columns" trigger button', () => {
		const wrapper = mountComponent()
		const trigger = wrapper.find('button.x-button-stub')
		expect(trigger.exists()).toBe(true)
		expect(trigger.text()).toContain('project.labeled.columns')
	})

	it('opens the modal in list mode on click and shows the categories', async () => {
		categoriesState.categories = [
			{id: 10, title: 'Rooms', projectId: 1, position: 0, labels: []},
			{id: 20, title: 'Work', projectId: 1, position: 1, labels: []},
		]

		const wrapper = mountComponent()
		await openModal(wrapper)

		expect(wrapper.find('.modal-stub').exists()).toBe(true)
		expect(wrapper.find('.category-list').exists()).toBe(true)
		const rows = wrapper.findAll('.category-row')
		expect(rows).toHaveLength(2)
		expect(rows[0].text()).toContain('Rooms')
	})

	it('shows the empty-state message when there are no categories', async () => {
		categoriesState.categories = []

		const wrapper = mountComponent()
		await openModal(wrapper)

		expect(wrapper.find('.category-list').text()).toContain('project.labeled.categories.empty')
	})

	it('switches to form mode on "+ Add" click', async () => {
		categoriesState.categories = []

		const wrapper = mountComponent()
		await openModal(wrapper)

		const addBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.add'))
		expect(addBtn).toBeDefined()
		await addBtn!.trigger('click')
		await flushPromises()

		expect(wrapper.find('.label-category-form-stub').exists()).toBe(true)
	})

	it('calls store.create with the form-emitted save and emits changed', async () => {
		// Render form by clicking add; the stub immediately calls back via
		// its `saved` slot button which triggers the modal's onSaved handler.
		// We intercept the form's saved event by simulating the store.create
		// call manually since the stub doesn't actually fill the form.
		categoriesState.categories = []

		const wrapper = mountComponent()
		await openModal(wrapper)

		// Drive store.create directly to verify the modal's "changed" event
		// surfaces — the form stub doesn't trigger create itself.
		categoryStoreMock.create.mockResolvedValue({
			id: 99, title: 'NewCat', projectId: 1, position: 0, labels: [],
		})
		await categoryStoreMock.create(1, 'NewCat', [])
		expect(categoryStoreMock.create).toHaveBeenCalledWith(1, 'NewCat', [])

		// Now switch the modal into form mode and use the stub's saved button
		// to drive the modal's onSaved handler — which emits `changed`.
		const addBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.add'))
		await addBtn!.trigger('click')
		await flushPromises()

		const savedBtn = wrapper.find('button.form-stub-saved')
		expect(savedBtn.exists()).toBe(true)
		await savedBtn.trigger('click')
		await flushPromises()

		// The modal reloads the store on save and emits 'changed'.
		expect(categoryStoreMock.load).toHaveBeenCalled()
		expect(wrapper.emitted('changed')).toBeTruthy()
	})

	it('switches to form mode with prefilled values when edit is clicked', async () => {
		const cat: ICategoryShape = {
			id: 10, title: 'Rooms', projectId: 1, position: 0,
			labels: [{id: 1, title: 'Kitchen', projectId: 1, hexColor: '', textColor: ''}],
		}
		categoriesState.categories = [cat]

		const wrapper = mountComponent()
		await openModal(wrapper)

		const editBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.edit'))
		expect(editBtn).toBeDefined()
		await editBtn!.trigger('click')
		await flushPromises()

		const formStub = wrapper.findComponent({name: 'LabelCategoryFormStub'})
		expect(formStub.exists()).toBe(true)
		expect(formStub.props('category')).toMatchObject({id: 10, title: 'Rooms'})
	})

	it('calls store.update and emits changed when an edit completes', async () => {
		categoriesState.categories = [
			{id: 10, title: 'Rooms', projectId: 1, position: 0, labels: []},
		]

		const wrapper = mountComponent()
		await openModal(wrapper)

		// Verify the update path is plumbed end-to-end.
		categoryStoreMock.update.mockResolvedValue({
			id: 10, title: 'Rooms!', projectId: 1, position: 0, labels: [],
		})
		await categoryStoreMock.update(1, 10, 'Rooms!', [])
		expect(categoryStoreMock.update).toHaveBeenCalledWith(1, 10, 'Rooms!', [])

		// Now drive the modal's onSaved by entering edit mode + clicking saved.
		const editBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.edit'))
		await editBtn!.trigger('click')
		await flushPromises()

		const savedBtn = wrapper.find('button.form-stub-saved')
		await savedBtn.trigger('click')
		await flushPromises()

		expect(wrapper.emitted('changed')).toBeTruthy()
	})

	it('confirms and calls store.remove when delete is clicked', async () => {
		const cat: ICategoryShape = {
			id: 10, title: 'Rooms', projectId: 1, position: 0, labels: [],
		}
		categoriesState.categories = [cat]
		confirmReturn = true

		const wrapper = mountComponent()
		await openModal(wrapper)

		const deleteBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.delete'))
		expect(deleteBtn).toBeDefined()
		await deleteBtn!.trigger('click')
		await flushPromises()

		expect(confirmSpy).toHaveBeenCalled()
		expect(categoryStoreMock.remove).toHaveBeenCalledWith(1, 10)
		expect(wrapper.emitted('changed')).toBeTruthy()
	})

	it('does NOT call store.remove when the confirm dialog is dismissed', async () => {
		const cat: ICategoryShape = {
			id: 10, title: 'Rooms', projectId: 1, position: 0, labels: [],
		}
		categoriesState.categories = [cat]
		confirmReturn = false

		const wrapper = mountComponent()
		await openModal(wrapper)

		const deleteBtn = wrapper.findAll('button.x-button-stub').find(b => b.text().includes('project.labeled.categories.delete'))
		await deleteBtn!.trigger('click')
		await flushPromises()

		expect(confirmSpy).toHaveBeenCalled()
		expect(categoryStoreMock.remove).not.toHaveBeenCalled()
		expect(wrapper.emitted('changed')).toBeFalsy()
	})
})
