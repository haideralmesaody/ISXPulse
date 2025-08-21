# Data Module Migration Plan

## Executive Summary

This document outlines the systematic migration of inline static data to separate data modules in the ISX Pulse frontend application. The migration aims to improve code organization, enable reusability, enhance type safety, and reduce bundle sizes through better tree-shaking.

**Target Completion**: Q1 2025  
**Risk Level**: Low (backward compatible, gradual migration)  
**Priority**: Medium (performance and maintainability improvement)

## Table of Contents

1. [Current State Analysis](#current-state-analysis)
2. [Target Architecture](#target-architecture)
3. [Migration Strategy](#migration-strategy)
4. [File-by-File Migration Tasks](#file-by-file-migration-tasks)
5. [Testing Requirements](#testing-requirements)
6. [Documentation Updates](#documentation-updates)
7. [Progress Tracking](#progress-tracking)

## Current State Analysis

### Issues with Current Approach
- Static data arrays defined inside components are recreated on every render
- No reusability of common data structures across components
- Larger component files mixing data and UI logic
- Harder to maintain and update static content
- No centralized type definitions for shared data

### Files with Inline Data
1. `app/page.tsx` - navigationCards array (4 items)
2. `app/analysis/page.tsx` - comingFeatures array (4 items)
3. `app/operations/page.tsx` - operationIcons object (5 mappings)
4. `app/license/page.tsx` - potential static data for features

## Target Architecture

### Directory Structure
```
dev/frontend/
└── lib/
    └── data/
        ├── README.md
        ├── navigation.ts
        ├── analysis-features.ts
        ├── operation-icons.ts
        └── license-features.ts
```

### Naming Conventions
- Use kebab-case for file names: `analysis-features.ts`
- Use camelCase for exported constants: `comingFeatures`
- Use PascalCase for interfaces: `ComingFeature`
- Add `as const` assertions for immutable data

### Type Definition Pattern
```typescript
// lib/data/analysis-features.ts
import type { LucideIcon } from 'lucide-react'

export interface ComingFeature {
  title: string
  description: string
  icon: LucideIcon
  color: string
  bgColor: string
}

export const EXPECTED_RELEASE = 'Q2 2025' as const

export const comingFeatures: readonly ComingFeature[] = [
  // ... feature objects
] as const
```

## Migration Strategy

### Phase 1: Foundation (Week 1)
- [ ] Create `lib/data/` directory
- [ ] Create `lib/data/README.md` with usage guidelines
- [ ] Implement pilot migration with `analysis/page.tsx`
- [ ] Update build and test processes

### Phase 2: Core Pages (Week 2)
- [ ] Migrate `app/page.tsx` navigation data
- [ ] Migrate `app/operations/page.tsx` icon mappings
- [ ] Update all imports and test each migration

### Phase 3: Documentation (Week 3)
- [ ] Update `dev/frontend/README.md`
- [ ] Update `CLAUDE.md` with data module pattern
- [ ] Create migration guide for future developers

### Phase 4: Testing & Validation (Week 4)
- [ ] Run full test suite
- [ ] Performance benchmarks
- [ ] Bundle size analysis
- [ ] Code review and cleanup

## File-by-File Migration Tasks

### 1. app/analysis/page.tsx

**Status**: 🟡 In Progress

#### Before
```typescript
const comingFeatures: ComingFeature[] = [
  {
    title: 'Market Trends Analysis',
    description: 'Track and analyze market trends with advanced visualizations',
    icon: TrendingUp,
    color: 'text-blue-600',
    bgColor: 'bg-blue-50'
  },
  // ... more features
]
```

#### After
```typescript
// In lib/data/analysis-features.ts
export const comingFeatures: readonly ComingFeature[] = [...]

// In app/analysis/page.tsx
import { comingFeatures, EXPECTED_RELEASE } from '@/lib/data/analysis-features'
```

#### Tests Required
- [ ] Component renders with imported data
- [ ] TypeScript compilation passes
- [ ] No runtime errors
- [ ] Bundle size comparison

#### Success Criteria
- [ ] Data successfully extracted to separate module
- [ ] All type safety maintained
- [ ] No visual regressions
- [ ] Tests pass

---

### 2. app/page.tsx (Home)

**Status**: 🔴 Not Started

#### Before
```typescript
const navigationCards = [
  {
    title: 'Operations',
    description: 'Manage and monitor data processing operations',
    icon: Activity,
    href: '/operations',
    color: 'text-blue-600',
    bgColor: 'bg-blue-50',
    available: true
  },
  // ... more cards
]
```

#### After
```typescript
// In lib/data/navigation.ts
export interface NavigationCard {
  title: string
  description: string
  icon: LucideIcon
  href: string
  color: string
  bgColor: string
  available: boolean
}

export const navigationCards: readonly NavigationCard[] = [...]

// In app/page.tsx
import { navigationCards } from '@/lib/data/navigation'
```

#### Tests Required
- [ ] Navigation renders correctly
- [ ] Links work as expected
- [ ] Styling preserved
- [ ] TypeScript types correct

#### Success Criteria
- [ ] All navigation items display correctly
- [ ] No broken links
- [ ] Type safety maintained
- [ ] Tests pass

---

### 3. app/operations/page.tsx

**Status**: 🔴 Not Started

#### Before
```typescript
const operationIcons: Record<string, React.ReactNode> = {
  scraping: <Download className="h-6 w-6" />,
  processing: <FileSpreadsheet className="h-6 w-6" />,
  indices: <BarChart3 className="h-6 w-6" />,
  analysis: <Database className="h-6 w-6" />,
  full_pipeline: <Workflow className="h-6 w-6" />
}
```

#### After
```typescript
// In lib/data/operation-icons.ts
import { Download, FileSpreadsheet, BarChart3, Database, Workflow } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export const operationIcons: Record<string, LucideIcon> = {
  scraping: Download,
  processing: FileSpreadsheet,
  indices: BarChart3,
  analysis: Database,
  full_pipeline: Workflow
} as const

// In app/operations/page.tsx
import { operationIcons } from '@/lib/data/operation-icons'

// Usage
const Icon = operationIcons[type]
return <Icon className="h-6 w-6" />
```

#### Tests Required
- [ ] Icons render correctly
- [ ] All operation types have icons
- [ ] No missing icon errors
- [ ] TypeScript types valid

#### Success Criteria
- [ ] All icons display properly
- [ ] No runtime errors
- [ ] Better type safety
- [ ] Tests pass

---

### 4. Quick Fixes for app/analysis/page.tsx

**Status**: 🟡 Ready to Implement

#### Tasks
- [ ] Replace `<input>` with `<Input>` component
- [ ] Add `aria-readonly="true"` to Input
- [ ] Remove `prefetch` prop from Link
- [ ] Add `tabIndex={-1}` to disabled Button
- [ ] Export page metadata
- [ ] Wrap feature cards in `<article>` tags
- [ ] Type `EXPECTED_RELEASE` as const

#### Tests Required
- [ ] Accessibility audit passes
- [ ] Component renders correctly
- [ ] SEO metadata present
- [ ] No TypeScript errors

## Testing Requirements

### Unit Tests
```typescript
// __tests__/lib/data/analysis-features.test.ts
describe('Analysis Features Data', () => {
  it('should export comingFeatures array', () => {
    expect(comingFeatures).toBeDefined()
    expect(comingFeatures.length).toBe(4)
  })
  
  it('should have valid feature structure', () => {
    comingFeatures.forEach(feature => {
      expect(feature).toHaveProperty('title')
      expect(feature).toHaveProperty('description')
      expect(feature).toHaveProperty('icon')
      expect(feature).toHaveProperty('color')
      expect(feature).toHaveProperty('bgColor')
    })
  })
})
```

### Integration Tests
- Test that pages render with imported data
- Verify no hydration errors
- Check bundle size impacts

### Manual Testing Checklist
- [ ] Visual regression testing
- [ ] Responsive design check
- [ ] Accessibility testing
- [ ] Performance metrics

## Documentation Updates

### 1. Create lib/data/README.md
```markdown
# Data Modules

Central location for static data and configuration used across components.

## Usage
Import data in your components:
\`\`\`typescript
import { navigationCards } from '@/lib/data/navigation'
\`\`\`

## Guidelines
- Keep data immutable with `as const`
- Export TypeScript interfaces
- Document data structure
- Use semantic naming
```

### 2. Update dev/frontend/README.md
- Add `lib/data/` to directory structure
- Document data module pattern
- Add migration examples

### 3. Update CLAUDE.md
Add to TypeScript/React Standards section:
```
- **Data Modules**: Static data in `lib/data/`, separate from UI components
- **Immutable Data**: Use `as const` assertions for static data
- **Type Exports**: Export interfaces alongside data
```

## Progress Tracking

### Overall Progress: 10% Complete

| Phase | Status | Completion | Notes |
|-------|--------|------------|-------|
| Foundation | 🟡 In Progress | 25% | Directory structure planned |
| Core Pages | 🔴 Not Started | 0% | Waiting on foundation |
| Documentation | 🔴 Not Started | 0% | Waiting on implementation |
| Testing | 🔴 Not Started | 0% | Waiting on implementation |

### File Migration Status

| File | Status | Extracted | Imported | Tested | Notes |
|------|--------|-----------|----------|---------|-------|
| app/analysis/page.tsx | 🟡 In Progress | ❌ | ❌ | ❌ | Quick fixes implemented |
| app/page.tsx | 🔴 Not Started | ❌ | ❌ | ❌ | Navigation data |
| app/operations/page.tsx | 🔴 Not Started | ❌ | ❌ | ❌ | Icon mappings |
| app/license/page.tsx | 🔴 Not Started | ❌ | ❌ | ❌ | To be analyzed |

### Legend
- 🟢 Complete
- 🟡 In Progress
- 🔴 Not Started
- ✅ Done
- ❌ Pending

## Rollback Procedures

If issues arise during migration:

1. **Immediate Rollback**: Revert the specific commit
2. **Partial Rollback**: Keep lib/data structure but move data back to components
3. **Investigation**: Check for:
   - Import path issues
   - Type mismatches
   - Build configuration problems
   - Bundle size regressions

## Success Metrics

- [ ] All static data extracted to data modules
- [ ] No increase in bundle size (target: 5% reduction)
- [ ] All tests passing
- [ ] No runtime errors in production
- [ ] Improved developer experience (measured by survey)

## Next Steps

1. Review and approve this plan
2. Create `lib/data/` directory structure
3. Begin Phase 1 pilot migration
4. Monitor progress and update this document

---

*Last Updated: [Current Date]*  
*Version: 1.0*  
*Owner: Development Team*