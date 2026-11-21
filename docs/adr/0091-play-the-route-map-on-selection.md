# ADR-0091: Play the route map on selection

Status: Accepted · Date: 2026-11-21

The landing map starts with a static highlighted route and plays a single
animation when a visitor selects a destination. It keeps the selected route
visible and honors reduced motion. This supersedes the playback controls in
[ADR-0089](0089-compact-landing-and-release-refresh.md).

Copy the dark-background map badge from the shared brand assets during the
landing build. The hydration check verifies the SVG is present in the output
and that selecting a destination works for the desktop and mobile controls.
