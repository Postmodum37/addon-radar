- Act as orchestrator for the Addon Radar project, providing guidance on architecture, technology choices, and implementation strategies. Use agents and MCPs to break down tasks and manage complexity.
- Follow TDD principles, ensuring that all code is thoroughly tested and meets high quality standards.
- Keep markdown files up-to-date with project status, architecture decisions, and implementation details. This means that after every significant change or addition, the relevant markdown files should be updated to reflect the current state of the project.
  - CLAUDE.md - Main project overview and architecture
  - PLAN.md - Detailed project plan and implementation steps
  - TODO.md - List of tasks and research areas to explore


- Focus on creating a unique and comprehensive trendiness algorithm for displaying addons.
- Ensure the application is scalable, maintainable, and easy to extend in the future.

## Project Overview

Addon Radar is a website displaying trending World of Warcraft addons for Retail version. It syncs data hourly from the CurseForge API and stores historical download snapshots to calculate velocity-based trending scores.

## Tech Stack
TBD

## Environment Variables

- `DATABASE_URL` - PostgreSQL connection string
- `CURSEFORGE_API_KEY` - CurseForge API key
- `PORT` - Server port (default: 8080)

## Trending Algorithm

The core differentiator is a sophisticated trending algorithm with two categories:

**Hot Right Now** - Top addons by absolute download velocity
- Requires 500+ total downloads
- Sorted by downloads gained in last 24 hours

**Rising Stars** - Top addons by percentage growth
- Requires 50-5,000 total downloads
- Sorted by growth percentage (7-day preferred, 24-hour fallback)

**Tier Multipliers** prevent tiny addons from dominating:
- 0-10 downloads: 0.1x
- 11-50: 0.3x
- 51-100: 0.5x
- 101-500: 0.7x
- 501-1000: 0.85x
- 1001+: 1.0x

## External Resources

- [CurseForge Addons](https://www.curseforge.com/wow)
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)
