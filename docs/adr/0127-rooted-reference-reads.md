# ADR-0127: Anchor OpenVPN reference reads to opened directories

Status: Accepted · Date: 2026-11-26

Refines ADR-0105 and ADR-0112. Research:
[Rooted reference reads](../research/2026-11-26-rooted-reference-reads.md).

## Decision

For referenced files, use `os.Root` and open each directory relative to the
previously opened directory. Check each component for links and compare the
opened directory with its `Lstat` result. Open the final file relative to its
verified parent, compare its identity before reading, and retain the existing
size and content checks. On Unix use `openat` with `O_NOFOLLOW` and
`O_NONBLOCK`; on Windows `os.Root` plus the identity check rejects a changed
final file. Absolute and parent references retain the restrictions of
ADR-0112. The selected base directory remains the trust anchor for inside
references; outside references start at the filesystem root.

This prevents an ancestor replaced after its check from redirecting the later
read. It does not prevent mount traversal, and does not change the separately
implemented elevated import or original-file search. The change does not
alter the user-facing import flow. Native macOS testing is pending.
