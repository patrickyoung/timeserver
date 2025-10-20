# Implementation Plans

This directory contains detailed implementation plans for features built into the Time Service. Each plan documents the step-by-step approach, tasks, deliverables, testing strategy, and commit messages for a specific feature or capability.

## Purpose

Implementation plans serve multiple purposes:

1. **Project Planning**: Break down complex features into manageable phases
2. **Historical Record**: Document how features were built (for completed plans)
3. **Blueprint**: Serve as templates for future similar implementations
4. **Onboarding**: Help new contributors understand the project's evolution
5. **Knowledge Transfer**: Explain architectural decisions and tradeoffs

## Plan Structure

Each plan follows a consistent structure:

- **Overview**: High-level description of the feature
- **Phases**: Implementation broken into logical phases
- **Tasks**: Specific work items for each phase
- **Deliverables**: Files and artifacts produced
- **Testing Strategy**: How to verify the implementation
- **Commit Messages**: Suggested commit message templates
- **Success Criteria**: Measurable goals
- **Timeline**: Estimated and actual time spent

## Plans

### [001: Foundation and Authentication](001-foundation-and-auth.md) ✅ COMPLETED

**Status**: Historical record (reverse-engineered from completed work)

**Features Implemented**:
- Basic HTTP server with REST API
- Model Context Protocol (MCP) server (stdio + HTTP)
- OAuth2/OIDC authentication and authorization
- Middleware stack (CORS, logging, metrics, recovery, auth)
- Prometheus observability
- Container hardening and deployment
- DevSecOps pipeline

**Total Time**: 16-22 hours across 7 phases

### [002: Named Locations](002-named-locations.md) 🚧 IN PROGRESS

**Status**: Phase 0 complete, implementing Phase 1

**Features**:
- SQLite database for location storage
- Location management API endpoints
- Location MCP tools
- Database configuration and metrics
- Container persistence support

**Total Estimated Time**: 8-14 hours across 6 phases

**Current Phase**: Phase 1 - Database Models and Repository

## Using These Plans

### For New Features

1. Copy an existing plan as a template
2. Number it sequentially (003, 004, etc.)
3. Adapt the phases and tasks to your feature
4. Update as you implement (mark phases complete, note actual time)
5. Create one commit per phase when possible

### For Historical Reference

- Plans marked "COMPLETED" are historical records
- They document **what was actually built**, not future plans
- Use them to understand why certain decisions were made
- Reference them when building similar features

### For Onboarding

New contributors should:
1. Read Plan 001 to understand the foundation
2. Review Plan 002 to see the current work
3. Check the status markers (✅ ⏸️ 🚧) to know what's active

## Conventions

### Status Markers

- ✅ **COMPLETED**: Historical record of finished work
- 🚧 **IN PROGRESS**: Currently being implemented
- ⏸️ **PAUSED**: Started but temporarily on hold
- 📋 **PLANNED**: Not yet started, ready for implementation
- 💡 **PROPOSED**: Under consideration, not approved

### Naming Convention

Plans use a three-digit number followed by a kebab-case description:

```
001-foundation-and-auth.md
002-named-locations.md
003-prometheus-dashboards.md
004-rate-limiting.md
```

### Phase Markers in Plans

Within each plan, phases are marked:

- ✅ **Phase N: Name** - Completed phase
- 🚧 **Phase N: Name** - Currently working on this phase
- 📋 **Phase N: Name** - Upcoming phase (not started)

## Guidelines for Writing Plans

### Level of Detail

Plans should be detailed enough that:
- A developer can implement the feature independently
- Another developer can review progress
- The feature can be picked up mid-implementation if needed

Include:
- Specific file paths to create/modify
- Code structure (interfaces, key functions)
- Testing requirements
- Configuration changes
- Documentation updates

### Time Estimates

Provide estimates for:
- Each phase individually
- Total implementation time
- Range (min-max hours)

Update with actuals as you go:
- **Estimated Time**: 2-3 hours
- **Actual Time**: 2.5 hours

### Commit Strategy

Each phase should produce a **coherent, reviewable commit**:
- One logical unit of work
- Builds successfully
- Tests pass
- Includes documentation updates
- Clear commit message

Avoid:
- Mixing multiple phases in one commit
- Breaking commits (doesn't build)
- Incomplete commits (half-implemented features)

### Testing Strategy

Every phase must include:
- What to test (unit, integration, manual)
- How to verify it works
- Expected results
- Edge cases to cover

## Related Documentation

- [docs/DESIGN.md](../DESIGN.md) - Technical design blueprint
- [docs/adr/](../adr/) - Architecture decision records
- [docs/TESTING.md](../TESTING.md) - Testing strategy
- [docs/DEVSECOPS.md](../DEVSECOPS.md) - Security practices

## Questions?

If you have questions about implementation plans or how to use them:
1. Review existing plans for examples
2. Check the DESIGN.md for architectural context
3. Review relevant ADRs for decision rationale
4. Ask in team discussions

---

**Last Updated**: 2025-10-19
