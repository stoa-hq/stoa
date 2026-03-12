# Stoa Documentation Skill

You are documenting a feature or change for the Stoa e-commerce platform. All documentation lives in the VitePress-based docs repo at `/home/epoxx/Repository/claude-projects/stoa-docs`.

## Repo structure

```
stoa-docs/
└── docs/
    ├── .vitepress/
    │   └── config.mts        ← sidebar & nav configuration
    ├── guide/                 ← getting started, architecture, core concepts
    ├── api/                   ← REST API reference
    ├── plugins/               ← plugin system & individual plugins
    └── mcp/                   ← MCP server docs
```

## Where does new content go?

| What you are documenting | Target directory |
|---|---|
| New domain, core concept, or platform behaviour | `docs/guide/` |
| New or changed REST API endpoint | `docs/api/` |
| New or updated plugin | `docs/plugins/` |
| MCP tool or server change | `docs/mcp/` |

## Steps for every documentation task

1. **Read the existing page** in the relevant section first to match tone, structure, and depth.
2. **Create or update the `.md` file** in the correct directory.
3. **Update `docs/.vitepress/config.mts`** — add a sidebar entry if a new page was created.
4. Never create a page without a sidebar entry, and never add a sidebar entry without a page.

## Writing conventions

- Title: `# Title` (H1), one per page.
- Use H2 (`##`) for major sections, H3 (`###`) for subsections.
- Code blocks always include the language identifier (` ```go `, ` ```yaml `, ` ```bash `, etc.).
- Use VitePress callouts for important notes:
  ```
  ::: tip
  Short helpful hint.
  :::

  ::: warning
  Something the user must not forget.
  :::

  ::: danger
  Destructive or irreversible action.
  :::
  ```
- Prices are always documented as **integer cents** (e.g. `1999` = €19.99).
- Tax rates are **integer basis points** (e.g. `1900` = 19%).
- Cross-link related pages using relative VitePress links: `[Payment](/plugins/payment)`.

## Plugin documentation template

Every plugin gets its own page under `docs/plugins/<plugin-name>.md`:

```markdown
# <Plugin Name>

Short description of what the plugin does and why it exists.

## How it works

Architecture diagram or prose explanation of the data flow.

## Installation

go get / import path + registration snippet in internal/app/app.go.

## Configuration

config.yaml snippet + table of all config keys (key | required | default | description).

## Behaviour

What the plugin does at runtime — hooks it listens to, endpoints it registers, etc.

## Error behaviour

How errors are handled, what gets logged, what is propagated.

## Example

A minimal but realistic end-to-end usage example.
```

## Sidebar entry format (config.mts)

```ts
{ text: 'Display Name', link: '/section/filename' }
```

The `link` value has no `.md` extension and is relative to `docs/`.

## Checklist

- [ ] `.md` file created or updated in the correct directory
- [ ] Sidebar entry added/updated in `docs/.vitepress/config.mts`
- [ ] Code examples are complete and copy-pasteable
- [ ] VitePress callouts used where appropriate
- [ ] Related pages cross-linked
