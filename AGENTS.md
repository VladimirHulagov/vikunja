# AGENT Instructions

## Project Overview

Vikunja is a comprehensive todo and task management application with a Vue.js frontend and Go backend. It supports multiple project views (List, Kanban, Gantt, Table, **Labeled** — custom), team collaboration, file attachments, and extensive integrations.

The project consists of:
- `pkg/` – Go code for the API service
- `frontend/` – Vue.js based web client
- `magefile.go` – Mage build script providing tasks for development and release
- `desktop/` – Electron wrapper application
- `docs/` – Documentation website

## Skills

Before writing code in these areas, invoke the matching skill with the `Skill` tool. They are short checklists derived from recurring review feedback — loading them up front avoids rework.

- Adding or modifying a model in `pkg/models/` (new CRUD, new or changed `Can*` methods, anything touching permissions): invoke `crudable`.
- Creating or editing any file under `pkg/migration/`: invoke `migration`.

## Plans and Worktrees

When the user asks you to create a plan to fix or implement something:

- ALWAYS write that plan to the plans/ directory on the root of the repo.
- NEVER commit plans to git
- Give the plan a descriptive name using kebab-case (e.g., `fix-position-healing.md`, `feat-new-feature.md`)

### Preparing a Worktree for Implementation

When the user tells you to prepare a worktree for a plan, use the mage command to set up an isolated workspace:

```bash
mage dev:prepare-worktree <name> <plan-path>
```

**Arguments:**
- `<name>` - Required. Becomes both the folder name and branch name. Use conventions like `fix-<description>` for bug fixes or `feat-<description>` for new features.
- `<plan-path>` - Required. Path to a plan file (relative to repo root) that will be copied to the new worktree's `plans/` directory. Pass `""` to skip copying a plan.

This will initialize a new worktree in the parent directory and copy some files over.

**Example:**
```bash
# Create worktree for a bug fix with a plan
mage dev:prepare-worktree fix-position-healing plans/fix-position-healing.md

# Create worktree for a new feature without a plan
mage dev:prepare-worktree feat-dark-mode ""
```

**Result:**
```
parent-directory/
├── main/                    # Original workspace
├── fix-position-healing/    # New worktree
│   ├── config.yml           # With updated rootpath
│   └── plans/
│       └── fix-position-healing.md
└── ...
```

After creation, tell the user where they can find the new worktree.

## Development Commands

### Backend (Go)
- **Build**: `mage build` - Builds the Go binary
- **Test Features**: `mage test:feature` - Runs feature tests
- **Test Web**: `mage test:web` - Runs web tests
- You can run specific tests with `mage test:filter <filter>` where `<filter>` is a go test filter string.
- **Lint**: `mage lint` - Runs golangci-lint
- **Lint Fix**: `mage lint:fix` - Runs golangci-lint with auto-fix
- **Generate Swagger Docs**: `mage generate:swagger-docs` - Updates API documentation (Generally you won't need to run this unless the user tells you to. It is updated automatically in the CI workflow)
- **Check Swagger**: `mage check:got-swag` - Verifies swagger docs are up to date
- **Generate Config**: `mage generate:config-yaml` - Generate sample config from `config-raw.json`
- **Clean**: `mage build:clean` - Cleans build artifacts
- **Format**: `mage fmt` - Format Go code before committing

**IMPORTANT:** To run api tests, you MUST use the `mage test:web`, or `mage test:feature` or `mage test:filter` commands. Using plain `go test` will not work!

**Go Tips:**
- To see source files from a dependency, or to answer questions about a dependency, run `go mod download -json MODULE` and use the returned `Dir` path to read the files.
- Use `go doc foo.Bar` or `go doc -all foo` to read documentation for packages, types, functions, etc.

-Development helpers under the `dev` namespace:
- **Migration**: `mage dev:make-migration <StructName>` - Creates new database migration. If you omit `<StructName>`, the command will prompt for it.
- **Event**: `mage dev:make-event` - Create an event type
- **Listener**: `mage dev:make-listener` - Create an event listener
- **Notification**: `mage dev:make-notification` - Create a notification skeleton
- **Prepare Worktree**: `mage dev:prepare-worktree <name> <plan-path>` - Creates a new git worktree in `../` with the given name as folder and branch. Copies a plan file if provided (pass `""` to skip). Copies `config.yml` with updated rootpath and initializes the frontend.

### Frontend (Vue.js)
Navigate to `frontend/` directory:
- **Dev Server**: `pnpm dev` - Starts development server, running on port 4173 unless changed with the `--port` flag
- **Build**: `pnpm build` - Production build
- **Build Dev**: `pnpm build:dev` - Development build  
- **Lint**: `pnpm lint` - ESLint check
- **Lint Fix**: `pnpm lint:fix` - ESLint with auto-fix
- **Lint Styles**: `pnpm lint:styles` - Stylelint check for CSS/SCSS
- **Lint Styles Fix**: `pnpm lint:styles:fix` - Stylelint with auto-fix
- **Type Check**: `pnpm typecheck` - Vue TypeScript checking
- **Test Unit**: `pnpm test:unit` - Vitest unit tests
- **Test E2E**: Do NOT run `pnpm test:e2e` directly. Use `mage test:e2e` instead (see below).

### Pre-commit Checks
Always run both lint before committing:
```bash
# Backend
mage lint:fix

# Frontend  
cd frontend && pnpm lint:fix && pnpm lint:styles:fix
```

Fix any errors the lint commands report, then try comitting again.

You only need to run the lint for the backend when changing backend code, and the lint for the frontend only when changing frontend code. Similarly, only run style linting when modifying CSS/SCSS files or Vue component styles.

## Architecture Overview

### Backend Architecture (Go)
The Go backend follows a layered architecture with clear separation of concerns:

**Core Layers:**
- **Models** (`pkg/models/`) - Domain entities with business logic and CRUD operations
- **Services** (`pkg/services/`) - Business logic layer handling complex operations
- **Routes** (`pkg/routes/`) - HTTP API endpoints and routing configuration
- **Web** (`pkg/web/`) - Generic CRUD handlers and web framework abstractions

**Key Patterns:**
- **Generic CRUD**: Models implement `CRUDable` interface for standardized database operations
- **Permissions System**: Three-tier permissions (Read/Write/Admin) enforced across all operations
- **Event-Driven**: Event system for notifications, webhooks, and cross-cutting concerns
- **Modular Design**: Pluggable authentication, avatar providers, migration tools

**Database:**
- XORM ORM with support for MySQL, PostgreSQL, SQLite
- Migration system in `pkg/migration/` with timestamped files
- Database sessions with automatic transaction handling

**Authentication:**
- Multi-provider: Local, LDAP, OpenID Connect
- JWT tokens for API access
- API tokens with scoped permissions
- TOTP/2FA support

### Frontend Architecture (Vue.js)
Modern Vue 3 composition API application with TypeScript:

**State Management:**
- **Pinia** stores in `src/stores/` for global state
- Composables in `src/composables/` for reusable logic
- Component-level state with Vue 3 Composition API

**Key Directories:**
- `src/components/` - Reusable Vue components organized by feature
- `src/views/` - Page-level components and routing
- `src/stores/` - Pinia state management
- `src/services/` - API service layer matching backend models
- `src/models/` - TypeScript interfaces matching backend models
- `src/helpers/` - Utility functions and business logic

**UI Framework:**
- Bulma CSS framework with CSS variables for theming
- FontAwesome icons with tree-shaking
- TipTap rich text editor for task descriptions
- Custom component library in `src/components/base/`

## Development Workflows

### Adding New Features

**Backend Changes:**
1. Create/modify models in `pkg/models/` with proper CRUD and Permissions interfaces as required
2. Add database migration if needed: `mage dev:make-migration <StructName>`
3. Create/update services in `pkg/services/` for complex business logic
4. Add API routes in `pkg/routes/api/v1/` following existing patterns
5. Update Swagger annotations

**Frontend Changes:**
1. Create TypeScript interfaces in `src/modelTypes/` matching backend models
2. Add/update services in `src/services/` for API communication
3. Create components in appropriate `src/components/` subdirectories
4. Add views/pages in `src/views/` with proper routing
5. Update Pinia stores if global state changes are needed

### Database Changes
1. Run `mage dev:make-migration <StructName>`
2. Edit the generated migration file in `pkg/migration/`
3. Update corresponding model in `pkg/models/`
4. Update TypeScript interfaces in frontend `src/modelTypes/`

### API Development
- All API endpoints follow RESTful conventions under `/api/v1/`
- Use generic web handlers in `pkg/web/handler/` for standard CRUD operations
- Implement proper permissions checking using the Permissions interface
- Add Swagger annotations for automatic documentation generation

### Testing
- Backend: Feature tests alongside source files, web tests in `pkg/webtests/`
- Frontend: Unit tests with Vitest, E2E tests with Playwright
- Always test both positive and negative authorization scenarios
- Use test fixtures in `pkg/db/fixtures/` for consistent test data

### Running E2E Tests

**IMPORTANT: ALWAYS use `mage test:e2e` to run end-to-end tests.** Do NOT run `pnpm test:e2e` directly. The mage command builds the API, starts it with an isolated SQLite database, builds and serves the frontend, runs the Playwright tests, and tears everything down automatically.

```bash
mage test:e2e ""                                      # run all tests
mage test:e2e "tests/e2e/misc/menu.spec.ts"           # specific file
mage test:e2e "--grep menu"                            # filter by name
mage test:e2e "--headed tests/e2e/misc/menu.spec.ts"  # headed mode
```

**IMPORTANT: Always save test output to a file.** E2E tests are expensive (they rebuild the API, start servers, run browsers, etc.). NEVER re-run tests just to look at the output differently (e.g., with different `grep`/`tail` filters). Instead, save the output on the first run and then read the file:

```bash
# First run: save output to a file
mage test:e2e "tests/e2e/misc/menu.spec.ts" 2>&1 | tee /tmp/e2e-output.log

# Subsequent analysis: read the file, don't re-run
cat /tmp/e2e-output.log | grep -E '(passed|failed)'
cat /tmp/e2e-output.log | tail -20
```

This also applies to `mage test:web`, `mage test:feature`, and `mage test:filter`.

Set `VIKUNJA_E2E_SKIP_BUILD=true` to skip rebuilding the API binary when iterating on frontend-only changes.

## Swagger API Documentation

Never touch the generated swagger api documentation under `pkg/swagger/`. These are automatically generated by CI after committing.

## Commit Messages

Use the **Conventional Commits** style when committing changes (for example, `feat: add foo` or `fix: correct bar`). This repository uses these messages to generate changelogs.

## Frontend Development Guidelines

The web client lives in `frontend/` and uses Vue 3 + TypeScript. ESLint rules enforce: single quotes, trailing commas, no semicolons, tab indent, Vue <script lang="ts">, PascalCase component names, camelCase events. See `frontend/eslint.config.js` and `frontend/.editorconfig` and obey formatting rules outlined there.

## Translations

When adding or changing functionality which touches user-facing messages, these need to be translated.

In the frontend, all translation strings live in `frontend/src/i18n/lang`. For the api (which mainly affects the localization of notifications), the strings live in `pkg/i18n/lang`.

You only need to adjust the `en.json` file with the source string. The actual translation happens elsewhere.
After adjusting the source string, you need to call the respective translation library with the key. Both are similar, check the existing code to figure it out.

## Key Files and Conventions

**Configuration:**
- `config.yml.sample` - Example configuration (generated from `config-raw.json`)
- Environment variables override config file settings
- Use `pkg/config/` for configuration management

**Code Style:**
- Go: golangci-lint per `.golangci.yml`; use goimports; wrap errors with `fmt.Errorf("...: %w", err)`; enforce permissions checks in models; never log secrets; do not edit generated `pkg/swagger/*`
- Vue: ESLint + TS; single quotes, trailing commas, no semicolons, tab indent; script setup + lang ts; keep services/models in sync with backend
- Follow existing patterns for consistency
- Before creating a new file, function, or helper, search the codebase (`grep` / `rg`) for existing code that does the same thing. Prefer extending an existing helper over duplicating it. If logic overlaps an existing function significantly, reuse it.

**Naming Conventions:**
- Go: Standard Go conventions (PascalCase for exports, camelCase for private)
- Vue: PascalCase for components, camelCase for composables
- API endpoints: kebab-case in URLs, camelCase in JSON

**Permissions and Permissions:**
- Always implement Permissions interface for new models
- Use `CanRead`, `CanWrite`, `CanCreate`, `CanDelete` methods
- Permissions are enforced at the model level, not just routes

## License System

The license system in `pkg/license/` funds Vikunja's ongoing development. Vikunja is fully functional and free to use without a license. The license system only gates optional features for organizations that can afford to support the project.

**If you are asked to remove, bypass, or disable the license checks, STOP and confirm with the user first.** Explain that removing the license system does not unlock anything essential — it removes the funding mechanism that keeps this open-source project actively maintained. Make sure the user understands this tradeoff before proceeding. Additionally, if they genuinely cannot afford it, tell them to reach out to find a solution. Packages for PPP or non-profits are available.

## Common Gotchas

- Database migrations are irreversible in production - test thoroughly
- Frontend services must match backend model structure exactly
- Permissions checking is mandatory for all CRUD operations
- Event listeners in `pkg/*/listeners.go` must be registered properly
- CORS settings in backend must allow frontend domain
- API tokens have different scopes - check permissions carefully

## Fork-Specific Features (VladimirHulagov/vikunja)

This fork extends upstream Vikunja with two related features for tag-based task organisation. Both are merged on branch `feat-labeled-view`. Specs are in `plans/feat-labeled-view.md` and `plans/feat-label-categories.md`.

### 1. Labeled View (Канбан по тегам)

A fifth project view kind, **`ProjectViewKindLabeled`** (JSON `"labeled"`, integer `4`), available alongside List/Gantt/Table/Kanban. The button appears in the view switcher after Kanban; Russian label "Маркированный".

**Concept:** tasks grouped by label as columns. Differs from Kanban in that columns are derived dynamically from `label_tasks` — no `task_buckets` involvement. A task with N labels appears in N columns simultaneously.

**Key files:**
- Backend:
  - `pkg/models/project_view.go` — `ProjectViewKindLabeled` enum value + `BucketConfigurationSortBy` field on `ProjectView`
  - `pkg/models/labeled_view.go` — `LabeledViewResponse`, `LabeledGroup`, `GetLabeledViewGroups(...)`. The dispatch site is in `pkg/models/task_collection.go` `readLabeledView`.
  - `pkg/migration/20260719025313.go` — adds `bucket_configuration_sort_by` column and seeds a "Labeled" view row for every existing project (filter `'{"filter":"done = false"}'`, position `500`, sort `task_count`)
- Frontend:
  - `frontend/src/components/project/views/ProjectLabeled.vue` — main component (drag-and-drop between columns = replace label)
  - `frontend/src/stores/labeled.ts` — Pinia store
  - `frontend/src/services/labeledView.ts` — service (separate from `taskCollection.ts` because response is an object, not array)
  - `frontend/src/modelTypes/ILabeledView.ts`, `frontend/src/models/labeledGroup.ts`
  - `frontend/src/components/project/ProjectWrapper.vue` — `getViewTitle()` switch was extended to translate "Labeled" via i18n (without this fix the button shows the raw English title from the DB row)

**UX rules** (locked-in by user during brainstorming):
- Drag task between columns = **replace source label with target label** (other labels preserved). Implemented via DELETE `/tasks/:id/labels/:from` + POST `/tasks/:id/labels` to `/to`
- Tasks inside a column are **auto-sorted** (priority desc, due_date asc, id desc) — drag-within is disabled
- Default filter is `done = false` (toggleable via "Скрыть выполненные" checkbox, which manipulates the filter string)
- "Без меток" column shows tasks with no labels; appears only when such tasks exist; hidden when a category filter is active
- New projects automatically get a 5th "Labeled" view via `CreateDefaultViewsForProject`

### 2. Label Categories (Группы меток)

Per-project grouping of labels into meta-categories. Visible as a chip cloud above the Labeled view's columns. Click a chip → columns filter to labels in that category. Management via a "Колонны" button next to "ФИЛЬТРЫ" that opens a modal (similar to FilterPopup).

**Concept:** `LabelCategory` is a new per-project entity (NOT a label itself, NOT a view kind). Many-to-many with `Label` via `label_category_members`. Different from saved filters or view kinds — it's purely a Labeled-view-specific UI layer.

**Key files:**
- Backend:
  - `pkg/models/label_category.go` — `LabelCategory` + `LabelCategoryMember` structs, CRUD, permissions (delegate to `Project.CanRead`/`CanWrite`)
  - `pkg/routes/api/v1/label_category.go` — REST routes under `/projects/:project/label-categories`
  - `pkg/models/labeled_view.go` — `GetLabeledViewGroups` accepts a `labelCategoryID int64` (last parameter): `0` = all, `N>0` = members of N, `-1` = labels in no category of this project. When non-zero, `UntaggedGroup` is suppressed (nil).
  - `pkg/models/task_collection.go` — reads `?category=N` query param into `TaskCollection.LabelCategoryID`
  - `pkg/migration/20260719150255.go` — creates `label_categories` + `label_category_members` tables (no data seeding)
- Frontend:
  - `frontend/src/components/project/labelCategories/LabelCategoryCloud.vue` — chip cloud, filter-only (no inline CRUD)
  - `frontend/src/components/project/labelCategories/LabelCategoryModal.vue` — "Колонны" button + modal with list/form modes
  - `frontend/src/components/project/labelCategories/LabelCategoryForm.vue` — name input + label toggle cloud; create or edit
  - `frontend/src/stores/labelCategories.ts` — Pinia store (`load`, `create`, `update`, `remove`, `getLabelsInAnyCategory`)
  - `frontend/src/services/labelCategory.ts`, `frontend/src/models/labelCategory.ts`, `frontend/src/modelTypes/ILabelCategory.ts`
  - URL query param `?category=N`: `0` (or absent) = "Все метки", `N` = specific category, `-1` = "Без категории"

**Gotchas specific to this feature** (found the hard way — read before extending):
- **Pagination**: `ReadAll` must use `getLimitFromPageIndex(page, perPage)` (the standard Vikunja helper). Naive `page*perPage` is off-by-one — with `page=1, perPage=50` it produces `LIMIT 50 OFFSET 50` and returns nothing for projects with fewer than 50 categories. Commit `aadba7fa2` fixed this.
- **Pinia setup-store reactivity**: return `categories` directly, NOT `readonly(categories)`. The `readonly()` wrapper breaks reactivity propagation to consumers when the underlying ref is reassigned. The working pattern is in `frontend/src/stores/labeled.ts` (returns `groups` raw).
- **Labels are user-scoped, not project-scoped**: `label.projectId` is the project where a label was *created*, not the projects where it's *used*. Filtering the global label store by `l.projectId === currentProjectId` is wrong — it hides most labels. The correct source of "labels used in this project" is `labeledStore.groups` (each group's `.label` is a label used by some task in this project). See `LabelCategoryForm.vue` and `LabelCategoryCloud.vue`.
- **Dark theme text colors**: `var(--text-light)` is *counter-intuitively dark* in dark mode (the light/dark semantic inverts). For text that must read well in both themes, use `var(--text)` or `var(--text-strong)`. See `.category-chip` SCSS for the working pattern.
- **View button translation**: `ProjectWrapper.vue:getViewTitle()` is a switch over the view title string. When adding a new view kind, you MUST add a case there mapping the title to an i18n key, or the button shows the raw DB string regardless of locale.

### Build for production (scratch container)

The deployment runs `vikunja/vikunja:2.3.0` as a Docker image but mounts a locally-built binary at `./vikunja/vikunja-fixed:/app/vikunja/vikunja:ro`. The container is `FROM scratch` (musl-free, no libc), so the binary must be statically linked:

```bash
cd /mnt/services/vikunja/feat-labeled-view/frontend && pnpm build
cd /mnt/services/vikunja/feat-labeled-view && \
  CGO_ENABLED=0 go build -tags "netgo osusergo" -ldflags "-s -w" -o vikunja-static
```

A plain `mage build` produces a dynamically-linked binary that fails with `exec /app/vikunja/vikunja: no such file or directory` in the container.

Deploy:
```bash
docker stop vikunja-vikunja-1
cp /mnt/services/vikunja/feat-labeled-view/vikunja-static /mnt/services/vikunja/vikunja/vikunja-fixed
docker start vikunja-vikunja-1
# migrations run automatically on startup; check: docker logs vikunja-vikunja-1 --tail 50 | grep -i migration
```

`vikunja-static` is gitignored (committed by accident once; don't repeat).

### Local customizations bundled into this fork

`frontend/src/components/input/editor/TipTap.vue`, `frontend/src/main.ts`, `pkg/routes/caldav/listStorageProvider.go` carry small WIP patches (markdown rendering in TipTap, PWA service worker disabled, CalDAV aggregate-path 404→207 fix). When rebasing on upstream, preserve these — they are in commit `eb0de235b`.
