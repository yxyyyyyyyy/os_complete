# AORT-R Bilingual Cockpit Dashboard Design

## Goal

Upgrade the AORT-R dashboard from a plain admin console into a competition-ready OS runtime cockpit. The default UI language is Chinese, with an immediate Chinese/English switch in the top-right control area.

## Visual Direction

- Dark cockpit background with restrained cyan, blue, green, and amber accents.
- Dense but readable operational layout: no marketing hero, no decorative card nesting.
- Runtime evidence is the visual priority: worker PID, cgroup/capsule, syscall timeline, CVM savings, scheduler decisions, and experiments.
- Cards keep 8px radius or less, stable dimensions, and no overlapping text.

## Language Model

- Add `dashboard/src/stores/i18n.ts`.
- Use a reactive `language` value with `zh` as default and `en` as optional.
- Expose `t` as the active dictionary and `setLanguage`.
- Convert user-facing labels in App, pages, and common components to dictionary keys.

## Page Changes

- App shell: add status topbar, language segmented control, polished sidebar labels.
- Overview: Chinese cockpit headline, runtime metrics, DAG, scheduler decision table.
- AVP: Chinese table headings and action tooltips with English fallback.
- Context: CVM metrics and page table labels in both languages.
- Timeline: event stream lane styling with bilingual labels.
- Experiments: E1/E2/E3 bilingual chart labels and table labels.

## Verification

- `npm run test`
- `npm run build`
- Browser visual smoke at `http://127.0.0.1:5173/` for desktop and mobile widths.
